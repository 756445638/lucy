package parser

import (
	"bytes"
	"fmt"

	"github.com/756445638/lucy/src/cmd/compile/ast"
	"github.com/756445638/lucy/src/cmd/compile/jvm/cg"
	"github.com/756445638/lucy/src/cmd/compile/lex"
	"github.com/timtadh/lexmachine"
)

func Parse(tops *[]*ast.Node, filename string, bs []byte, onlyimport bool) []error {
	return (&Parser{bs: bs, tops: tops, filename: filename, onlyimport: onlyimport}).Parse()
}

type Parser struct {
	onlyimport       bool
	bs               []byte
	lines            [][]byte
	tops             *[]*ast.Node
	ExpressionParser *ExpressionParser
	Function         *Function
	Class            *Class
	Block            *Block
	scanner          *lexmachine.Scanner
	filename         string
	lastToken        *lex.Token
	token            *lex.Token
	eof              bool
	errs             []error
	imports          map[string]*ast.Imports
}

func (p *Parser) Parse() []error {
	p.ExpressionParser = &ExpressionParser{p}
	p.Function = &Function{}
	p.Function.parser = p
	p.Class = &Class{}
	p.Class.parser = p
	p.Block = &Block{}
	p.Block.parser = p
	p.errs = []error{}
	var err error
	p.scanner, err = lex.Lexer.Scanner(p.bs)
	p.lines = bytes.Split(p.bs, []byte("\n"))
	if err != nil {
		p.errs = append(p.errs, err)
		return p.errs
	}
	p.Next()
	//package name definition
	if p.eof {
		p.errs = append(p.errs, fmt.Errorf("%s no package name definition found", p.errorMsgPrefix()))
		return p.errs
	}
	if p.token.Type != lex.TOKEN_PACKAGE {
		p.errs = append(p.errs, fmt.Errorf("%s first token must be a  package name definition", p.errorMsgPrefix()))
		return p.errs
	}
	p.Next()
	if p.eof {
		p.errs = append(p.errs, fmt.Errorf("%s no package name definition found(no name after)", p.errorMsgPrefix()))
		return p.errs
	}
	if p.token.Type != lex.TOKEN_IDENTIFIER {
		p.errs = append(p.errs, fmt.Errorf("%s no package name definition found(no name after)", p.errorMsgPrefix()))
		return p.errs
	}
	pd := &ast.PackageNameDeclare{
		Name: p.token.Data.(string),
	}
	p.Next()
	if p.token.Type != lex.TOKEN_SEMICOLON {
		p.errs = append(p.errs, fmt.Errorf("%s no semicolon after package name definition", p.errorMsgPrefix()))
		p.consume(untils_semicolon)
		p.Next()
	} else {
		p.Next()
	}
	pd.Pos = &ast.Pos{}
	p.lexPos2AstPos(p.token, pd.Pos)
	*p.tops = append(*p.tops, &ast.Node{
		Data: pd,
	})

	p.parseImports() // next is called
	if p.eof {
		return p.errs
	}

	if p.onlyimport { // only parse imports
		return p.errs
	}
	ispublic := false
	isconst := false
	resetProperty := func() {
		ispublic = false
		isconst = false
	}
	for !p.eof {
		switch p.token.Type {
		case lex.TOKEN_SEMICOLON: // empty statement, no big deal
			p.Next()
			continue
		case lex.TOKEN_VAR:
			vs, err := p.parseVarDefinition(ispublic)
			if err != nil {
				p.consume(untils_semicolon)
				p.Next()
				continue
			}
			if vs != nil && len(vs) > 0 {
				for _, v := range vs {
					*p.tops = append(*p.tops, &ast.Node{
						Data: v,
					})
				}
			}
			resetProperty()
		case lex.TOKEN_IDENTIFIER:

			resetProperty()
		case lex.TOKEN_ENUM:
			if isconst {
				p.errs = append(p.errs, fmt.Errorf("%s cannot use const for a enum", p.errorMsgPrefix()))
			}
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
			if isconst {
				p.errs = append(p.errs, fmt.Errorf("%s cannot use const for a enum", p.errorMsgPrefix()))
			}
			f, err := p.Function.parse(ispublic)
			if err != nil {
				p.errs = append(p.errs, err)
				p.consume(untils_rc)
				p.Next()
				continue
			}
			f.MkVariableType()
			*p.tops = append(*p.tops, &ast.Node{
				Data: f,
			})
			resetProperty()
		case lex.TOKEN_LC:
			if isconst {
				p.errs = append(p.errs, fmt.Errorf("%s cannot use const for a block", p.errorMsgPrefix()))
			}
			if ispublic {
				p.errs = append(p.errs, fmt.Errorf("%s cannot use public for a block", p.errorMsgPrefix()))
			}
			b := &ast.Block{}
			err = p.Block.parse(b) // this function will lookup next
			if err != nil {
				p.consume(untils_rc)
				p.Next()
			}
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
				c.Access |= cg.ACC_FIELD_PUBLIC
			} else {
				c.Access |= cg.ACC_FIELD_PRIVATE
			}
			resetProperty()
		case lex.TOKEN_PUBLIC:
			ispublic = true
			p.Next()
			continue
		case lex.TOKEN_CONST:
			isconst = true
			p.Next()
			if p.token.Type == lex.TOKEN_ENUM || p.token.Type == lex.TOKEN_CLASS {
				p.errs = append(p.errs, fmt.Errorf("%s cannot use const for enum or class ", p.errorMsgPrefix()))
				resetProperty()
			}
			vs, typ, err := p.parseAssignedNames()
			if err != nil {
				p.consume(untils_semicolon)
				p.Next()
				resetProperty()
				continue
			}
			if p.token.Type != lex.TOKEN_SEMICOLON && (p.lastToken != nil && p.lastToken.Type != lex.TOKEN_RC) { //assume missing ; not big deal
				p.errs = append(p.errs, fmt.Errorf("%s not ; after variable or const definition,but %s", p.errorMsgPrefix(), p.token.Desp))
				p.Next()
				p.consume(untils_semicolon)
				resetProperty()
				continue
			}
			// const a := 1 is wrong,
			if typ == lex.TOKEN_COLON_ASSIGN && isconst == true {
				p.errs = append(p.errs, fmt.Errorf("%s use = instead of := for const definition", p.errorMsgPrefix()))
				resetProperty()
				continue
			}
			for _, v := range vs {
				c := &ast.Const{}
				c.VariableDefinition = *v
				if ispublic {
					c.AccessFlags |= cg.ACC_FIELD_PUBLIC
				} else {
					c.AccessFlags |= cg.ACC_FIELD_PRIVATE
				}
				*p.tops = append(*p.tops, &ast.Node{
					Data: c,
				})
			}
			continue
		case lex.TOKEN_PRIVATE: //is a default attribute
			ispublic = false
			p.Next()
			continue
		default:
			p.errs = append(p.errs, fmt.Errorf("%s token(%s) is not except", p.errorMsgPrefix(), p.token.Desp))
			p.consume(untils_semicolon)
			resetProperty()
		}
	}
	return p.errs
}

func (p *Parser) insertImports(im *ast.Imports) {
	if p.imports == nil {
		p.imports = make(map[string]*ast.Imports)
	}
	access, err := im.GetAccessName()
	if err != nil {
		p.errs = append(p.errs, fmt.Errorf("%s %v", p.errorMsgPrefix(im.Pos), err))
		return
	}
	if p.imports[access] != nil {
		p.errs = append(p.errs, fmt.Errorf("%s package %s reimported", p.errorMsgPrefix(im.Pos), access))
		return
	}
	p.imports[access] = im
}

func (p *Parser) mkPos() *ast.Pos {
	return &ast.Pos{
		Filename:    p.filename,
		StartLine:   p.token.Match.StartLine,
		StartColumn: p.token.Match.StartColumn,
	}
}

// str := "hello world"   a,b = 123 or a b ;
func (p *Parser) parseAssignedNames() ([]*ast.VariableDefinition, int, error) {
	names, err := p.parseNameList()
	if err != nil {
		return nil, 0, err
	}
	//trying to parse type
	var variableType *ast.VariableType
	if p.token.Type != lex.TOKEN_ASSIGN && p.token.Type != lex.TOKEN_COLON_ASSIGN {
		variableType, err = p.parseType()
		if err != nil {
			p.errs = append(p.errs, err)
		}
		return nil, 0, err
	}
	if p.token.Type != lex.TOKEN_ASSIGN && p.token.Type != lex.TOKEN_COLON_ASSIGN {
		err = fmt.Errorf("%s missing = or := after a name list", p.errorMsgPrefix())
		p.errs = append(p.errs, err)
		return nil, 0, err
	}
	typ := p.token.Type
	p.Next()
	if p.eof {
		err = p.mkUnexpectedEofErr()
		p.errs = append(p.errs, err)
		return nil, typ, err
	}
	es, err := p.ExpressionParser.parseExpressions()
	if err != nil {
		return nil, typ, err
	}
	if len(es) != len(names) {
		err = fmt.Errorf("%s mame and value not match", p.errorMsgPrefix())
		p.errs = append(p.errs, err)
		return nil, typ, err
	}
	vs := make([]*ast.VariableDefinition, len(names))
	for k, v := range names {
		vd := &ast.VariableDefinition{}
		vs[k] = vd
		vd.Name = v.Name
		vd.Typ = variableType
		vd.Pos = v.Pos
		vd.Expression = es[k]
	}
	return vs, typ, nil
}

func (p *Parser) Next() {
	var err error
	var tok interface{}
	p.lastToken = p.token
	for !p.eof {
		tok, err, p.eof = p.scanner.Next()
		if err != nil {
			p.eof = true
			return
		}
		if tok != nil && tok.(*lex.Token).Type != lex.TOKEN_CRLF {
			p.token = tok.(*lex.Token)
			fmt.Println("#########", p.token.Desp)
			break
		}
	}
	return
}

func (p *Parser) unexpectedErr() {
	p.errs = append(p.errs, p.mkUnexpectedEofErr())
}
func (p *Parser) mkUnexpectedEofErr() error {
	return fmt.Errorf("%s unexpected EOF", p.errorMsgPrefix())
}

/*
	errorMsgPrefix(pos) only receive one argument
*/
func (p *Parser) errorMsgPrefix(pos ...*ast.Pos) string {
	if len(pos) > 0 {
		return fmt.Sprintf("%s:%d:%d '%s'", pos[0].Filename, pos[0].StartLine, pos[0].StartColumn, string(p.lines[pos[0].StartLine-1]))
	}
	return fmt.Sprintf("%s:%d:%d '%s'", p.filename, p.token.Match.StartLine, p.token.Match.StartColumn, string(p.lines[p.token.Match.StartLine-1]))
}

//var a,b,c int,char,bool  | var a,b,c int = 123;
func (p *Parser) parseVarDefinition(ispublic ...bool) (vs []*ast.VariableDefinition, err error) {
	p.Next()
	if p.eof {
		err = p.mkUnexpectedEofErr()
		p.errs = append(p.errs, err)
		return
	}
	names, err := p.parseNameList()
	if err != nil {
		return nil, err
	}
	if p.eof {
		err = p.mkUnexpectedEofErr()
		p.errs = append(p.errs, err)
		return
	}
	t, err := p.parseType()
	if t == nil {
		err = fmt.Errorf("%s no variable type found or defined wrong", p.errorMsgPrefix())
		p.errs = append(p.errs, err)
		return nil, err
	}
	var expressions []*ast.Expression
	//value , no default value definition
	if lex.TOKEN_ASSIGN == p.token.Type {
		//assign
		p.Next() // skip =
		expressions, err = p.ExpressionParser.parseExpressions()
		panic(expressions)
		if err != nil {
			p.errs = append(p.errs, err)
		}
	}
	if p.token.Type != lex.TOKEN_SEMICOLON {
		err = fmt.Errorf("%s not a \";\" after a variable declare ", p.errorMsgPrefix())
		p.errs = append(p.errs, err)
		return
	}
	p.Next() // look next
	if len(expressions) > 0 && len(names) != len(expressions) {
		err = fmt.Errorf("%s name list and value list has no same length", p.errorMsgPrefix())
		p.errs = append(p.errs, err)
		return
	}
	fmt.Println(expressions)
	vs = make([]*ast.VariableDefinition, len(names))
	for k, v := range names {
		vd := &ast.VariableDefinition{}
		vd.Name = v.Name
		vd.Typ = t.Clone()
		if len(ispublic) > 0 && ispublic[0] {
			vd.AccessFlags |= cg.ACC_FIELD_PUBLIC
		} else {
			vd.AccessFlags |= cg.ACC_FIELD_PRIVATE
		}
		if k >= 0 && k < len(expressions) {
			vd.Expression = expressions[k]
		}
		vd.Pos = v.Pos
		vs[k] = vd
	}
	return vs, nil
}

func (p *Parser) parseType() (*ast.VariableType, error) {
	var err error
	switch p.token.Type {
	case lex.TOKEN_LB:
		p.Next()
		if p.token.Type != lex.TOKEN_RB {
			// [ and ] not match
			err = fmt.Errorf("%s [ and ] not match", p.errorMsgPrefix())
			p.errs = append(p.errs, err)
			return nil, err
		}
		//lookahead
		p.Next() //skip ]
		t, err := p.parseType()
		if err != nil {
			return nil, err
		}
		tt := &ast.VariableType{}
		tt.CombinationType = &ast.VariableType{}
		tt.CombinationType.Typ = ast.VARIABLE_TYPE_ARRAY
		tt.CombinationType = t
		return tt, nil
	case lex.TOKEN_BOOL:
		p.Next()
		return &ast.VariableType{
			Typ: ast.VARIABLE_TYPE_BOOL,
			Pos: p.mkPos(),
		}, nil
	case lex.TOKEN_BYTE:
		p.Next()
		return &ast.VariableType{
			Typ: ast.VARIABLE_TYPE_BYTE,
			Pos: p.mkPos(),
		}, nil
	case lex.TOKEN_SHORT:
		p.Next()
		return &ast.VariableType{
			Typ: ast.VARIABLE_TYPE_SHORT,
			Pos: p.mkPos(),
		}, nil
	case lex.TOKEN_INT:
		p.Next()
		return &ast.VariableType{
			Typ: ast.VARIABLE_TYPE_INT,
			Pos: p.mkPos(),
		}, nil
	case lex.TOKEN_FLOAT:
		p.Next()
		return &ast.VariableType{
			Typ: ast.VARIABLE_TYPE_FLOAT,
			Pos: p.mkPos(),
		}, nil

	case lex.TOKEN_DOUBLE:
		p.Next()
		return &ast.VariableType{
			Typ: ast.VARIABLE_TYPE_DOUBLE,
			Pos: p.mkPos(),
		}, nil
	case lex.TOKEN_LONG:
		p.Next()
		return &ast.VariableType{
			Typ: ast.VARIABLE_TYPE_LONG,
			Pos: p.mkPos(),
		}, nil
	case lex.TOKEN_STRING:
		p.Next()
		return &ast.VariableType{
			Typ: ast.VARIABLE_TYPE_STRING,
			Pos: p.mkPos(),
		}, nil
	case lex.TOKEN_IDENTIFIER:
		return p.parseIdentifierType()
	case lex.TOKEN_FUNCTION:
		p.Next()
		t, err := p.parseFunctionType()
		if err != nil {
			return nil, err
		}
		return &ast.VariableType{
			Typ:          ast.VARIABLE_TYPE_FUNCTION,
			FunctionType: t,
			Pos:          p.mkPos(),
		}, nil
	}
	err = fmt.Errorf("%s unkown type,first token:", p.errorMsgPrefix(), p.token.Desp)
	p.errs = append(p.errs, err)
	return nil, err
}

//(a,b int)->(total int)
func (p *Parser) parseFunctionType() (t *ast.FunctionType, err error) {
	t = &ast.FunctionType{}
	if p.token.Type != lex.TOKEN_LP {
		err = fmt.Errorf("%s fn declared wrong,missing (,but %s", p.errorMsgPrefix(), p.token.Desp)
		p.errs = append(p.errs, err)
		return
	}
	p.Next()                          // skip (
	if p.token.Type != lex.TOKEN_RP { // not (
		t.Parameters, err = p.parseTypedNames()
		if err != nil {
			return nil, err
		}
	}
	if p.token.Type != lex.TOKEN_RP { // not )
		err = fmt.Errorf("%s fn declared wrong,missing ),but %s", p.errorMsgPrefix(), p.token.Desp)
		p.errs = append(p.errs, err)
		return
	}
	p.Next()
	if p.token.Type == lex.TOKEN_ARROW { // ->
		p.Next() // skip ->
		if p.token.Type != lex.TOKEN_LP {
			err = fmt.Errorf("%s fn declared wrong, not ( after ->", p.errorMsgPrefix())
			p.errs = append(p.errs, err)
			return
		}
		p.Next()
		t.Returns, err = p.parseTypedNames()
		if err != nil {
			return
		}
		if p.token.Type != lex.TOKEN_RP {
			err = fmt.Errorf("%s fn declared wrong, ( and ) not match", p.errorMsgPrefix())
			p.errs = append(p.errs, err)
			return
		}
		p.Next()
	}
	return t, err
}
func (p *Parser) parseIdentifierType() (*ast.VariableType, error) {
	name := p.token.Data.(string)
	ret := &ast.VariableType{
		Pos: p.mkPos(),
		Typ: ast.VARIABLE_TYPE_NAME,
	}
	p.Next() // skip name identifier
	for p.token.Type == lex.TOKEN_DOT && !p.eof {
		p.Next() // skip .
		if p.token.Type != lex.TOKEN_IDENTIFIER {
			return nil, fmt.Errorf("%s not a identifier after dot", p.errorMsgPrefix())
		}
		name += "." + p.token.Data.(string)
		p.Next() // if
	}
	ret.Name = name
	return ret, nil
}

//at least one name
func (p *Parser) parseNameList() (names []*ast.NameWithPos, err error) {
	if p.token.Type != lex.TOKEN_IDENTIFIER {
		err = fmt.Errorf("%s is not identifer,but %s", p.errorMsgPrefix(), p.token.Desp)
		p.errs = append(p.errs, err)
		return nil, err
	}
	names = []*ast.NameWithPos{}
	for p.token.Type == lex.TOKEN_IDENTIFIER && !p.eof {
		names = append(names, &ast.NameWithPos{
			Name: p.token.Data.(string),
			Pos:  p.mkPos(),
		})
		p.Next()
		if p.token.Type != lex.TOKEN_COMMA {
			// not a ,
			break
		}
		pos := p.mkPos() // more
		p.Next()
		if p.token.Type != lex.TOKEN_IDENTIFIER {
			err = fmt.Errorf("%s not identifier after a comma,but %s ", p.errorMsgPrefix(pos), p.token.Desp)
			p.errs = append(p.errs, err)
			return names, err
		}
	}
	return
}

func (p *Parser) consume(untils map[int]bool) {
	if len(untils) == 0 {
		panic("no token to consume")
	}
	var ok bool
	for !p.eof {
		if _, ok = untils[p.token.Type]; ok {
			return
		}
		p.Next()
	}
}

//imports,alway call next
func (p *Parser) parseImports() {
	if p.token.Type != lex.TOKEN_IMPORT {
		// not a import
		return
	}
	// p.token.Type == lex.TOKEN_IMPORT
	p.Next()
	if p.token.Type != lex.TOKEN_LITERAL_STRING {
		p.consume(untils_semicolon)
		p.errs = append(p.errs, fmt.Errorf("%s expect string literal after import", p.errorMsgPrefix()))
		p.parseImports()
		return
	}
	packagename := p.token.Data.(string)
	p.Next()
	if p.token.Type == lex.TOKEN_AS {
		i := &ast.Imports{}
		i.Pos = &ast.Pos{}
		p.lexPos2AstPos(p.token, i.Pos)
		i.Name = packagename
		p.Next()
		if p.token.Type != lex.TOKEN_IDENTIFIER {
			p.consume(untils_semicolon)
			p.Next()
			p.errs = append(p.errs, fmt.Errorf("%s expect identifier after as", p.errorMsgPrefix()))
			p.parseImports()
			return
		}
		i.AccessName = p.token.Data.(string)
		p.Next()
		if p.token.Type != lex.TOKEN_SEMICOLON {
			p.consume(untils_semicolon)
			p.Next()
			p.errs = append(p.errs, fmt.Errorf("%s  semicolon after import statement", p.errorMsgPrefix()))
			p.parseImports()
			return
		}
		p.Next()
		*p.tops = append(*p.tops, &ast.Node{
			Data: i,
		})
		p.insertImports(i)
		p.parseImports()
		return
	} else if p.token.Type == lex.TOKEN_SEMICOLON {
		i := &ast.Imports{}
		i.Name = packagename
		i.Pos = &ast.Pos{}
		p.lexPos2AstPos(p.token, i.Pos)
		*p.tops = append(*p.tops, &ast.Node{
			Data: i,
		})
		p.Next()
		p.insertImports(i)
		p.parseImports()
		return
	} else {
		p.consume(untils_semicolon)
		p.Next()
		p.errs = append(p.errs, fmt.Errorf("%s expect semicolon after", p.errorMsgPrefix()))
		p.parseImports()
		return
	}
}

func (p *Parser) lexPos2AstPos(t *lex.Token, pos *ast.Pos) {
	pos.Filename = p.filename
	pos.StartLine = t.Match.StartLine
	pos.StartColumn = t.Match.StartColumn
}

//func (p *Parser) parseTypeOrTypedNames() (names []*ast.VariableDefinition, err error) {
//
//	return nil, nil
//}

// a,b int or int,bool  c xxx
func (p *Parser) parseTypedNames() (vs []*ast.VariableDefinition, err error) {
	vs = []*ast.VariableDefinition{}
	for !p.eof {
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
		}
	}
	return vs, nil
}
