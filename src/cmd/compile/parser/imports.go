package parser

import (
	"fmt"

	"github.com/756445638/lucy/src/cmd/compile/ast"
	"github.com/756445638/lucy/src/cmd/compile/lex"
)

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