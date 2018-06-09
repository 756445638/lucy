package parser

import (
	"bytes"
	"fmt"

	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/lex"
)

func Parse(tops *[]*ast.Node, filename string, bs []byte, onlyimport bool, nerr int) []error {
	p := &Parser{
		bs:         bs,
		tops:       tops,
		filename:   filename,
		onlyimport: onlyimport,
		nerr:       nerr,
	}
	return p.Parse()
}

type Parser struct {
	onlyimport bool
	bs         []byte
	lines      [][]byte
	tops       *[]*ast.Node
	scanner    *lex.LucyLexer
	filename   string
	lastToken  *lex.Token
	token      *lex.Token
	errs       []error
	imports    map[string]*ast.Import
	nerr       int
	// parsers
	Expression *Expression
	Function   *Function
	Class      *Class
	Block      *Block
	Interface  *Interface
}

func (p *Parser) Parse() []error {
	p.Expression = &Expression{p}
	p.Function = &Function{}
	p.Function.parser = p
	p.Class = &Class{}
	p.Class.parser = p
	p.Interface = &Interface{}
	p.Interface.parser = p
	p.Block = &Block{}
	p.Block.parser = p
	p.errs = []error{}
	p.scanner = lex.New(p.bs, 1, 1)
	p.lines = bytes.Split(p.bs, []byte("\n"))
	p.Next()
	if p.token.Type == lex.TOKEN_EOF {
		return nil
	}
	p.parseImports() // next is called
	if p.token.Type == lex.TOKEN_EOF {
		return p.errs
	}
	if p.onlyimport { // only parse imports
		return p.errs
	}
	ispublic := false
	resetProperty := func() {
		ispublic = false
	}
	for p.token.Type != lex.TOKEN_EOF {
		if len(p.errs) > p.nerr {
			break
		}
		switch p.token.Type {
		case lex.TOKEN_SEMICOLON: // empty statement, no big deal
			p.Next()
			continue
		case lex.TOKEN_VAR:
			pos := p.mkPos()
			p.Next() // skip var key word
			vs, es, typ, err := p.parseConstDefinition(true)
			if err != nil {
				p.consume(untils_semicolon)
				p.Next()
				continue
			}
			if typ != nil && typ.Type != lex.TOKEN_ASSIGN {
				p.errs = append(p.errs,
					fmt.Errorf("%s use '=' to initialize value",
						p.errorMsgPrefix()))
			}
			d := &ast.ExpressionDeclareVariable{Variables: vs, Values: es}
			e := &ast.Expression{
				Typ:      ast.EXPRESSION_TYPE_VAR,
				Data:     d,
				Pos:      pos,
				IsPublic: ispublic,
			}
			*p.tops = append(*p.tops, &ast.Node{
				Data: e,
			})
			resetProperty()
		case lex.TOKEN_IDENTIFIER:
			e, err := p.Expression.parseExpression(true)
			if err != nil {
				p.consume(untils_semicolon)
				p.Next()
				continue
			}
			e.IsPublic = ispublic
			p.validStatementEnding(e.Pos)
			*p.tops = append(*p.tops, &ast.Node{
				Data: e,
			})
			resetProperty()
		case lex.TOKEN_ENUM:
			e, err := p.parseEnum(ispublic)
			if err != nil {
				p.consume(untils_rc)
				p.Next()
				resetProperty()
				continue
			}
			if e != nil {
				*p.tops = append(*p.tops, &ast.Node{
					Data: e,
				})
			}
			resetProperty()
		case lex.TOKEN_FUNCTION:
			f, err := p.Function.parse(true)
			if err != nil {
				p.consume(untils_rc)
				p.Next()
				continue
			}
			if ispublic {
				f.AccessFlags |= cg.ACC_METHOD_PUBLIC
			} else {
				f.AccessFlags |= cg.ACC_METHOD_PRIVATE
			}
			*p.tops = append(*p.tops, &ast.Node{
				Data: f,
			})
			resetProperty()
		case lex.TOKEN_LC:
			b := &ast.Block{}
			p.Next()                            // skip {
			p.Block.parseStatementList(b, true) // this function will lookup next
			if p.token.Type != lex.TOKEN_RC {
				p.errs = append(p.errs, fmt.Errorf("%s expect '}', but '%s'"))
				p.consume(untils_rc)
			}
			p.Next() // skip }
			*p.tops = append(*p.tops, &ast.Node{
				Data: b,
			})
			resetProperty()
		case lex.TOKEN_CLASS:
			c, err := p.Class.parse()
			if err != nil {
				p.errs = append(p.errs, err)
				p.consume(untils_rc)
				p.Next()
				resetProperty()
				continue
			}
			*p.tops = append(*p.tops, &ast.Node{
				Data: c,
			})
			if ispublic {
				c.AccessFlags |= cg.ACC_CLASS_PUBLIC
			}
			resetProperty()
		case lex.TOKEN_INTERFACE:
			c, err := p.Interface.parse()
			if err != nil {
				p.errs = append(p.errs, err)
				p.consume(untils_rc)
				p.Next()
				resetProperty()
				continue
			}
			*p.tops = append(*p.tops, &ast.Node{
				Data: c,
			})
			if ispublic {
				c.AccessFlags |= cg.ACC_CLASS_PUBLIC
			}
			resetProperty()
		case lex.TOKEN_PUBLIC:
			ispublic = true
			p.Next()
			p.validAfterPublic(ispublic)
			continue
		case lex.TOKEN_CONST:
			p.Next() // skip const key word
			vs, es, typ, err := p.parseConstDefinition(false)
			if err != nil {
				p.consume(untils_semicolon)
				p.Next()
				resetProperty()
				continue
			}
			if p.validStatementEnding() == false { //assume missing ; not big deal
				p.Next()
				p.consume(untils_semicolon)
				resetProperty()
				continue
			}
			// const a := 1 is wrong,
			if typ != nil && typ.Type != lex.TOKEN_ASSIGN {
				p.errs = append(p.errs, fmt.Errorf("%s use '=' instead of ':=' for const definition",
					p.errorMsgPrefix()))
				resetProperty()
				continue
			}
			if len(vs) != len(es) {
				p.errs = append(p.errs,
					fmt.Errorf("%s cannot assign %d values to %d destinations",
						p.errorMsgPrefix(p.mkPos()), len(es), len(vs)))
			}
			for k, v := range vs {
				if k < len(es) {
					c := &ast.Const{}
					c.VariableDefinition = *v
					c.Expression = es[k]
					if ispublic {
						c.AccessFlags |= cg.ACC_FIELD_PUBLIC
					} else {
						c.AccessFlags |= cg.ACC_FIELD_PRIVATE
					}
					*p.tops = append(*p.tops, &ast.Node{
						Data: c,
					})
				}
			}
			resetProperty()
			continue
		case lex.TOKEN_PRIVATE: //is a default attribute
			ispublic = false
			p.Next()
			p.validAfterPublic(ispublic)
			continue
		case lex.TOKEN_TYPE:
			a, err := p.parseTypeaAlias()
			if err != nil {
				p.consume(untils_semicolon)
				p.Next()
				resetProperty()
				continue
			}
			*p.tops = append(*p.tops, &ast.Node{
				Data: a,
			})

		case lex.TOKEN_EOF:
			break
		default:
			p.errs = append(p.errs, fmt.Errorf("%s token(%s) is not except",
				p.errorMsgPrefix(), p.token.Desp))
			p.consume(untils_semicolon)
			resetProperty()
		}
	}
	return p.errs
}

func (p *Parser) parseTypes() ([]*ast.VariableType, error) {
	ret := []*ast.VariableType{}
	for p.token.Type != lex.TOKEN_EOF {
		t, err := p.parseType()
		if err != nil {
			return ret, err
		}
		ret = append(ret, t)
		if p.token.Type != lex.TOKEN_COMMA {
			break
		}
		p.Next() // skip ,
	}
	return ret, nil
}

func (p *Parser) validAfterPublic(isPublic bool) {
	if p.token.Type == lex.TOKEN_FUNCTION ||
		p.token.Type == lex.TOKEN_CLASS ||
		p.token.Type == lex.TOKEN_ENUM ||
		p.token.Type == lex.TOKEN_IDENTIFIER ||
		p.token.Type == lex.TOKEN_INTERFACE ||
		p.token.Type == lex.TOKEN_CONST ||
		p.token.Type == lex.TOKEN_VAR {
		return
	}
	var err error
	token := "public"
	if isPublic == false {
		token = "private"
	}
	if p.token.Desp != "" {
		err = fmt.Errorf("%s cannot have token:%s after '%s'",
			p.errorMsgPrefix(), p.token.Desp, token)
	} else {
		err = fmt.Errorf("%s cannot have token:%s after '%s'",
			p.errorMsgPrefix(), p.token.Desp, token)
	}
	p.errs = append(p.errs, err)
}
func (p *Parser) validStatementEnding(pos ...*ast.Pos) bool {
	if p.token.Type == lex.TOKEN_SEMICOLON ||
		(p.lastToken != nil && p.lastToken.Type == lex.TOKEN_RC) {
		return true
	}
	if len(pos) > 0 {
		p.errs = append(p.errs, fmt.Errorf("%s missing semicolon", p.errorMsgPrefix(pos[0])))
	} else {
		p.errs = append(p.errs, fmt.Errorf("%s missing semicolon", p.errorMsgPrefix()))
	}
	return false
}

func (p *Parser) mkPos() *ast.Pos {
	return &ast.Pos{
		Filename:    p.filename,
		StartLine:   p.token.StartLine,
		StartColumn: p.token.StartColumn,
		Offset:      p.scanner.GetOffSet(),
	}
}

// str := "hello world"   a,b = 123 or a b ;
func (p *Parser) parseConstDefinition(needType bool) ([]*ast.VariableDefinition, []*ast.Expression, *lex.Token, error) {
	names, err := p.parseNameList()
	if err != nil {
		return nil, nil, nil, err
	}
	var variableType *ast.VariableType
	//trying to parse type
	if p.isValidTypeBegin() || needType {
		variableType, err = p.parseType()
		if err != nil {
			p.errs = append(p.errs, err)
			return nil, nil, nil, err
		}
	}
	f := func() []*ast.VariableDefinition {
		vs := make([]*ast.VariableDefinition, len(names))
		for k, v := range names {
			vd := &ast.VariableDefinition{}
			vd.Name = v.Name
			vd.Pos = v.Pos
			if variableType != nil {
				vd.Typ = variableType.Clone()
			}
			vs[k] = vd
		}
		return vs
	}
	if p.token.Type != lex.TOKEN_ASSIGN &&
		p.token.Type != lex.TOKEN_COLON_ASSIGN {
		return f(), nil, nil, err
	}
	typ := p.token
	p.Next() // skip = or :=
	es, err := p.Expression.parseExpressions()
	if err != nil {
		return nil, nil, typ, err
	}
	return f(), es, typ, nil
}

func (p *Parser) Next() {
	var err error
	var tok *lex.Token
	p.lastToken = p.token
	for {
		tok, err = p.scanner.Next()
		if err != nil {
			p.errs = append(p.errs, fmt.Errorf("%s %s", p.errorMsgPrefix(), err.Error()))
		}
		if tok == nil {
			continue
		}
		p.token = tok
		if tok.Type != lex.TOKEN_CRLF {
			if p.token.Desp != "" {
				//	fmt.Println("#########", p.token.Type, p.token.Desp)
			} else {
				//fmt.Println("#########", p.token.Type, p.token.Data)
			}
			break
		}
	}
	return
}

/*
	errorMsgPrefix(pos) only receive one argument
*/
func (p *Parser) errorMsgPrefix(pos ...*ast.Pos) string {
	if len(pos) > 0 {
		return fmt.Sprintf("%s:%d:%d", pos[0].Filename, pos[0].StartLine, pos[0].StartColumn)
	}
	line, column := p.scanner.Pos()
	return fmt.Sprintf("%s:%d:%d", p.filename, line, column)
}

func (p *Parser) consume(untils map[int]bool) {
	if len(untils) == 0 {
		panic("no token to consume")
	}
	var ok bool
	for p.token.Type != lex.TOKEN_EOF {
		if _, ok = untils[p.token.Type]; ok {
			return
		}
		p.Next()
	}
}

func (p *Parser) lexPos2AstPos(t *lex.Token, pos *ast.Pos) {
	pos.Filename = p.filename
	pos.StartLine = t.StartLine
	pos.StartColumn = t.StartColumn
}

func (p *Parser) parseTypeaAlias() (*ast.ExpressionTypeAlias, error) {
	p.Next() // skip type key word
	if p.token.Type != lex.TOKEN_IDENTIFIER {
		err := fmt.Errorf("%s expect identifer,but %s", p.errorMsgPrefix(), p.token.Desp)
		p.errs = append(p.errs, err)
		return nil, err
	}
	ret := &ast.ExpressionTypeAlias{}
	ret.Pos = p.mkPos()
	ret.Name = p.token.Data.(string)
	p.Next() // skip identifier
	if p.token.Type != lex.TOKEN_ASSIGN {
		err := fmt.Errorf("%s expect '=',but %s", p.errorMsgPrefix(), p.token.Desp)
		p.errs = append(p.errs, err)
		return nil, err
	}
	p.Next() // skip =
	var err error
	ret.Typ, err = p.parseType()
	return ret, err
}

func (p *Parser) parseTypedName() (vs []*ast.VariableDefinition, err error) {
	names, err := p.parseNameList()
	if err != nil {
		return nil, err
	}
	t, err := p.parseType()
	if err != nil {
		return nil, err
	}
	vs = make([]*ast.VariableDefinition, len(names))
	for k, v := range names {
		vd := &ast.VariableDefinition{}
		vs[k] = vd
		vd.Name = v.Name
		vd.Pos = v.Pos
		vd.Typ = t.Clone()
	}
	return vs, nil
}

// a,b int or int,bool  c xxx
func (p *Parser) parseTypedNames() (vs []*ast.VariableDefinition, err error) {
	vs = []*ast.VariableDefinition{}
	for p.token.Type != lex.TOKEN_EOF {
		ns, err := p.parseNameList()
		if err != nil {
			return vs, err
		}
		t, err := p.parseType()
		if err != nil {
			return vs, err
		}
		for _, v := range ns {
			vd := &ast.VariableDefinition{}
			vd.Name = v.Name
			vd.Pos = v.Pos
			vd.Typ = t.Clone()
			vs = append(vs, vd)
		}
		if p.token.Type != lex.TOKEN_COMMA { // not a commna
			break
		} else {
			p.Next()
		}
	}
	return vs, nil
}
