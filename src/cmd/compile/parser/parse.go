package parser

import (
	"bytes"
	"fmt"

	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/lex"
)

func Parse(tops *[]*ast.Top, filename string, bs []byte, onlyImport bool, nerr int) []error {
	p := &Parser{
		bs:           bs,
		tops:         tops,
		filename:     filename,
		onlyImport:   onlyImport,
		nErrors2Stop: nerr,
	}
	return p.Parse()
}

type Parser struct {
	onlyImport   bool
	bs           []byte
	lines        [][]byte
	tops         *[]*ast.Top
	scanner      *lex.Lexer
	filename     string
	lastToken    *lex.Token
	token        *lex.Token
	errs         []error
	imports      map[string]*ast.Import
	nErrors2Stop int
	// parsers
	ExpressionParser *ExpressionParser
	FunctionParser   *FunctionParser
	ClassParser      *ClassParser
	BlockParser      *BlockParser
	InterfaceParser  *InterfaceParser
}

func (parser *Parser) Parse() []error {
	parser.ExpressionParser = &ExpressionParser{parser}
	parser.FunctionParser = &FunctionParser{}
	parser.FunctionParser.parser = parser
	parser.ClassParser = &ClassParser{}
	parser.ClassParser.parser = parser
	parser.InterfaceParser = &InterfaceParser{}
	parser.InterfaceParser.parser = parser
	parser.BlockParser = &BlockParser{}
	parser.BlockParser.parser = parser
	parser.errs = []error{}
	parser.scanner = lex.New(parser.bs, 1, 1)
	parser.lines = bytes.Split(parser.bs, []byte("\n"))
	parser.Next()
	if parser.token.Type == lex.TOKEN_EOF {
		return nil
	}
	parser.parseImports() // next is called
	if parser.token.Type == lex.TOKEN_EOF {
		return parser.errs
	}
	if parser.onlyImport { // only parse imports
		return parser.errs
	}
	isPublic := false
	resetProperty := func() {
		isPublic = false
	}
	for parser.token.Type != lex.TOKEN_EOF {
		if len(parser.errs) > parser.nErrors2Stop {
			break
		}
		switch parser.token.Type {
		case lex.TOKEN_SEMICOLON: // empty statement, no big deal
			parser.Next()
			continue
		case lex.TOKEN_VAR:
			pos := parser.mkPos()
			parser.Next() // skip var key word
			vs, es, typ, err := parser.parseConstDefinition(true)
			if err != nil {
				parser.consume(untilSemicolon)
				parser.Next()
				continue
			}
			if typ != nil && typ.Type != lex.TOKEN_ASSIGN {
				parser.errs = append(parser.errs,
					fmt.Errorf("%s use '=' to initialize value",
						parser.errorMsgPrefix()))
			}
			d := &ast.ExpressionDeclareVariable{Variables: vs, InitValues: es}
			e := &ast.Expression{
				Type:     ast.EXPRESSION_TYPE_VAR,
				Data:     d,
				Pos:      pos,
				IsPublic: isPublic,
			}
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: e,
			})
			resetProperty()
		case lex.TOKEN_IDENTIFIER:
			e, err := parser.ExpressionParser.parseExpression(true)
			if err != nil {
				parser.consume(untilSemicolon)
				parser.Next()
				continue
			}
			e.IsPublic = isPublic
			parser.validStatementEnding(e.Pos)
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: e,
			})
			resetProperty()
		case lex.TOKEN_ENUM:
			e, err := parser.parseEnum(isPublic)
			if err != nil {
				parser.consume(untilRc)
				parser.Next()
				resetProperty()
				continue
			}
			if e != nil {
				*parser.tops = append(*parser.tops, &ast.Top{
					Data: e,
				})
			}
			resetProperty()
		case lex.TOKEN_FUNCTION:
			f, err := parser.FunctionParser.parse(true)
			if err != nil {
				parser.consume(untilRc)
				parser.Next()
				continue
			}
			if isPublic {
				f.AccessFlags |= cg.ACC_METHOD_PUBLIC
			} else {
				f.AccessFlags |= cg.ACC_METHOD_PRIVATE
			}
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: f,
			})
			resetProperty()
		case lex.TOKEN_LC:
			b := &ast.Block{}
			parser.Next()                                  // skip {
			parser.BlockParser.parseStatementList(b, true) // this function will lookup next
			if parser.token.Type != lex.TOKEN_RC {
				parser.errs = append(parser.errs, fmt.Errorf("%s expect '}', but '%s'",
					parser.errorMsgPrefix(), parser.token.Description))
				parser.consume(untilRc)
			}
			parser.Next() // skip }
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: b,
			})
			resetProperty()
		case lex.TOKEN_CLASS:
			c, err := parser.ClassParser.parse()
			if err != nil {
				parser.errs = append(parser.errs, err)
				parser.consume(untilRc)
				parser.Next()
				resetProperty()
				continue
			}
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: c,
			})
			if isPublic {
				c.AccessFlags |= cg.ACC_CLASS_PUBLIC
			}
			resetProperty()
		case lex.TOKEN_INTERFACE:
			c, err := parser.InterfaceParser.parse()
			if err != nil {
				parser.errs = append(parser.errs, err)
				parser.consume(untilRc)
				parser.Next()
				resetProperty()
				continue
			}
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: c,
			})
			if isPublic {
				c.AccessFlags |= cg.ACC_CLASS_PUBLIC
			}
			resetProperty()
		case lex.TOKEN_PUBLIC:
			isPublic = true
			parser.Next()
			parser.validAfterPublic(isPublic)
			continue
		case lex.TOKEN_CONST:
			parser.Next() // skip const key word
			vs, es, typ, err := parser.parseConstDefinition(false)
			if err != nil {
				parser.consume(untilSemicolon)
				parser.Next()
				resetProperty()
				continue
			}
			if parser.validStatementEnding() == false { //assume missing ; not big deal
				parser.Next()
				parser.consume(untilSemicolon)
				resetProperty()
				continue
			}
			// const a := 1 is wrong,
			if typ != nil && typ.Type != lex.TOKEN_ASSIGN {
				parser.errs = append(parser.errs, fmt.Errorf("%s use '=' instead of ':=' for const definition",
					parser.errorMsgPrefix()))
				resetProperty()
				continue
			}
			if len(vs) != len(es) {
				parser.errs = append(parser.errs,
					fmt.Errorf("%s cannot assign %d values to %d destinations",
						parser.errorMsgPrefix(parser.mkPos()), len(es), len(vs)))
			}
			for k, v := range vs {
				if k < len(es) {
					c := &ast.Constant{}
					c.Variable = *v
					c.Expression = es[k]
					if isPublic {
						c.AccessFlags |= cg.ACC_FIELD_PUBLIC
					} else {
						c.AccessFlags |= cg.ACC_FIELD_PRIVATE
					}
					*parser.tops = append(*parser.tops, &ast.Top{
						Data: c,
					})
				}
			}
			resetProperty()
			continue
		case lex.TOKEN_TYPE:
			a, err := parser.parseTypeAlias()
			if err != nil {
				parser.consume(untilSemicolon)
				parser.Next()
				resetProperty()
				continue
			}
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: a,
			})

		case lex.TOKEN_EOF:
			break
		default:
			parser.errs = append(parser.errs, fmt.Errorf("%s token(%s) is not except",
				parser.errorMsgPrefix(), parser.token.Description))
			parser.consume(untilSemicolon)
			resetProperty()
		}
	}
	return parser.errs
}

func (parser *Parser) parseTypes() ([]*ast.Type, error) {
	ret := []*ast.Type{}
	for parser.token.Type != lex.TOKEN_EOF {
		t, err := parser.parseType()
		if err != nil {
			return ret, err
		}
		ret = append(ret, t)
		if parser.token.Type != lex.TOKEN_COMMA {
			break
		}
		parser.Next() // skip ,
	}
	return ret, nil
}

func (parser *Parser) validAfterPublic(isPublic bool) {
	if parser.token.Type == lex.TOKEN_FUNCTION ||
		parser.token.Type == lex.TOKEN_CLASS ||
		parser.token.Type == lex.TOKEN_ENUM ||
		parser.token.Type == lex.TOKEN_IDENTIFIER ||
		parser.token.Type == lex.TOKEN_INTERFACE ||
		parser.token.Type == lex.TOKEN_CONST ||
		parser.token.Type == lex.TOKEN_VAR {
		return
	}
	var err error
	token := "public"
	if isPublic == false {
		token = "private"
	}
	if parser.token.Description != "" {
		err = fmt.Errorf("%s cannot have token:%s after '%s'",
			parser.errorMsgPrefix(), parser.token.Description, token)
	} else {
		err = fmt.Errorf("%s cannot have token:%s after '%s'",
			parser.errorMsgPrefix(), parser.token.Description, token)
	}
	parser.errs = append(parser.errs, err)
}
func (parser *Parser) validStatementEnding(pos ...*ast.Position) bool {
	if parser.token.Type == lex.TOKEN_SEMICOLON ||
		(parser.lastToken != nil && parser.lastToken.Type == lex.TOKEN_RC) {
		return true
	}
	if len(pos) > 0 {
		parser.errs = append(parser.errs, fmt.Errorf("%s missing semicolon", parser.errorMsgPrefix(pos[0])))
	} else {
		parser.errs = append(parser.errs, fmt.Errorf("%s missing semicolon", parser.errorMsgPrefix()))
	}
	return false
}

func (parser *Parser) mkPos() *ast.Position {
	return &ast.Position{
		Filename:    parser.filename,
		StartLine:   parser.token.StartLine,
		StartColumn: parser.token.StartColumn,
		Offset:      parser.scanner.GetOffSet(),
	}
}

// str := "hello world"   a,b = 123 or a b ;
func (parser *Parser) parseConstDefinition(needType bool) ([]*ast.Variable, []*ast.Expression, *lex.Token, error) {
	names, err := parser.parseNameList()
	if err != nil {
		return nil, nil, nil, err
	}
	var variableType *ast.Type
	//trying to parse type
	if parser.isValidTypeBegin() || needType {
		variableType, err = parser.parseType()
		if err != nil {
			parser.errs = append(parser.errs, err)
			return nil, nil, nil, err
		}
	}
	f := func() []*ast.Variable {
		vs := make([]*ast.Variable, len(names))
		for k, v := range names {
			vd := &ast.Variable{}
			vd.Name = v.Name
			vd.Pos = v.Pos
			if variableType != nil {
				vd.Type = variableType.Clone()
			}
			vs[k] = vd
		}
		return vs
	}
	if parser.token.Type != lex.TOKEN_ASSIGN &&
		parser.token.Type != lex.TOKEN_COLON_ASSIGN {
		return f(), nil, nil, err
	}
	typ := parser.token
	parser.Next() // skip = or :=
	es, err := parser.ExpressionParser.parseExpressions()
	if err != nil {
		return nil, nil, typ, err
	}
	return f(), es, typ, nil
}

func (parser *Parser) Next() {
	var err error
	var tok *lex.Token
	parser.lastToken = parser.token
	for {
		tok, err = parser.scanner.Next()
		if err != nil {
			parser.errs = append(parser.errs, fmt.Errorf("%s %s", parser.errorMsgPrefix(), err.Error()))
		}
		if tok == nil {
			continue
		}
		parser.token = tok
		if tok.Type != lex.TOKEN_LF {
			if parser.token.Description != "" {
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
func (parser *Parser) errorMsgPrefix(pos ...*ast.Position) string {
	if len(pos) > 0 {
		return fmt.Sprintf("%s:%d:%d", pos[0].Filename, pos[0].StartLine, pos[0].StartColumn)
	}
	line, column := parser.scanner.GetPos()
	return fmt.Sprintf("%s:%d:%d", parser.filename, line, column)
}

func (parser *Parser) consume(until map[int]bool) {
	if len(until) == 0 {
		panic("no token to consume")
	}
	var ok bool
	for parser.token.Type != lex.TOKEN_EOF {
		if _, ok = until[parser.token.Type]; ok {
			return
		}
		parser.Next()
	}
}

func (parser *Parser) lexPos2AstPos(t *lex.Token, pos *ast.Position) {
	pos.Filename = parser.filename
	pos.StartLine = t.StartLine
	pos.StartColumn = t.StartColumn
}

func (parser *Parser) parseTypeAlias() (*ast.ExpressionTypeAlias, error) {
	parser.Next() // skip type key word
	if parser.token.Type != lex.TOKEN_IDENTIFIER {
		err := fmt.Errorf("%s expect identifer,but %s", parser.errorMsgPrefix(), parser.token.Description)
		parser.errs = append(parser.errs, err)
		return nil, err
	}
	ret := &ast.ExpressionTypeAlias{}
	ret.Pos = parser.mkPos()
	ret.Name = parser.token.Data.(string)
	parser.Next() // skip identifier
	if parser.token.Type != lex.TOKEN_ASSIGN {
		err := fmt.Errorf("%s expect '=',but %s", parser.errorMsgPrefix(), parser.token.Description)
		parser.errs = append(parser.errs, err)
		return nil, err
	}
	parser.Next() // skip =
	var err error
	ret.Type, err = parser.parseType()
	return ret, err
}

func (parser *Parser) parseTypedName() (vs []*ast.Variable, err error) {
	names, err := parser.parseNameList()
	if err != nil {
		return nil, err
	}
	t, err := parser.parseType()
	if err != nil {
		return nil, err
	}
	vs = make([]*ast.Variable, len(names))
	for k, v := range names {
		vd := &ast.Variable{}
		vs[k] = vd
		vd.Name = v.Name
		vd.Pos = v.Pos
		vd.Type = t.Clone()
	}
	return vs, nil
}

// a,b int or int,bool  c xxx
func (parser *Parser) parseTypedNames() (vs []*ast.Variable, err error) {
	vs = []*ast.Variable{}
	for parser.token.Type != lex.TOKEN_EOF {
		ns, err := parser.parseNameList()
		if err != nil {
			return vs, err
		}
		t, err := parser.parseType()
		if err != nil {
			return vs, err
		}
		for _, v := range ns {
			vd := &ast.Variable{}
			vd.Name = v.Name
			vd.Pos = v.Pos
			vd.Type = t.Clone()
			vs = append(vs, vd)
		}
		if parser.token.Type != lex.TOKEN_COMMA { // not a comma
			break
		} else {
			parser.Next()
		}
	}
	return vs, nil
}
