package parser

import (
	"fmt"

	"github.com/756445638/lucy/src/cmd/compile/ast"
	"github.com/756445638/lucy/src/cmd/compile/lex"
)

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
	}
	err = fmt.Errorf("%s unkown type,first token:%s", p.errorMsgPrefix(), p.token.Desp)
	p.errs = append(p.errs, err)
	return nil, err
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