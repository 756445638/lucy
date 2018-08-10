package parser

import (
	"fmt"

	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/lex"
)

type InterfaceParser struct {
	parser             *Parser
	ret                *ast.Class
	accessControlToken *lex.Token
}

func (interfaceParser *InterfaceParser) Next(lfIsToken bool) {
	interfaceParser.parser.Next(lfIsToken)
}

func (interfaceParser *InterfaceParser) consume(m map[lex.TokenKind]bool) {
	interfaceParser.parser.consume(m)
}

func (interfaceParser *InterfaceParser) validStatementEnding() error {
	return interfaceParser.parser.validStatementEnding()
}

func (interfaceParser *InterfaceParser) parse() (classDefinition *ast.Class, err error) {
	interfaceParser.Next(lfIsToken) // skip interface key word
	interfaceParser.parser.unExpectNewLineAndSkip()
	interfaceParser.ret = &ast.Class{}
	interfaceParser.ret.Pos = interfaceParser.parser.mkPos()
	interfaceParser.ret.Block.IsClassBlock = true
	interfaceParser.ret.AccessFlags |= cg.ACC_CLASS_INTERFACE // interface
	interfaceParser.ret.AccessFlags |= cg.ACC_CLASS_ABSTRACT
	interfaceParser.ret.Name, err = interfaceParser.parser.ClassParser.parseClassName()
	classDefinition = interfaceParser.ret
	if err != nil {
		interfaceParser.parser.errs = append(interfaceParser.parser.errs, err)
		interfaceParser.consume(untilLc) //
		interfaceParser.ret.Name = compileAutoName()
	}
	if interfaceParser.parser.token.Type == lex.TokenExtends { // parse father expression
		interfaceParser.Next(lfNotToken) // skip extends
		interfaceParser.ret.Pos = interfaceParser.parser.mkPos()
		if interfaceParser.parser.token.Type != lex.TokenIdentifier {
			err = fmt.Errorf("%s class`s father must be a identifier", interfaceParser.parser.errorMsgPrefix())
			interfaceParser.parser.errs = append(interfaceParser.parser.errs, err)
			interfaceParser.consume(untilLc) //
		} else {
			t, err := interfaceParser.parser.ClassParser.parseClassName()
			interfaceParser.ret.SuperClassName = t
			if err != nil {
				interfaceParser.parser.errs = append(interfaceParser.parser.errs, err)
				return nil, err
			}
		}
	}
	if interfaceParser.parser.token.Type == lex.TokenImplements {
		interfaceParser.Next(lfNotToken) // skip key word
		interfaceParser.ret.InterfaceNames, err = interfaceParser.parser.ClassParser.parseImplementsInterfaces()
		if err != nil {
			interfaceParser.consume(untilLc)
		}
	}
	interfaceParser.parser.ifTokenIsLfThenSkip()
	if interfaceParser.parser.token.Type != lex.TokenLc {
		err = fmt.Errorf("%s expect '{' but '%s'", interfaceParser.parser.errorMsgPrefix(),
			interfaceParser.parser.token.Description)
		interfaceParser.parser.errs = append(interfaceParser.parser.errs, err)
		interfaceParser.consume(untilLc)
	}
	interfaceParser.Next(lfNotToken)
	for interfaceParser.parser.token.Type != lex.TokenEof {
		if len(interfaceParser.parser.errs) > interfaceParser.parser.nErrors2Stop {
			break
		}
		switch interfaceParser.parser.token.Type {
		case lex.TokenRc:
			interfaceParser.Next(lfNotToken)
			return
		case lex.TokenSemicolon:
			interfaceParser.Next(lfNotToken)
			continue
		case lex.TokenFn:
			interfaceParser.Next(lfIsToken) /// skip key word
			interfaceParser.parser.unExpectNewLineAndSkip()
			var name string
			if interfaceParser.parser.token.Type != lex.TokenIdentifier {
				interfaceParser.parser.errs = append(interfaceParser.parser.errs, fmt.Errorf("%s expect function name,but '%s'",
					interfaceParser.parser.errorMsgPrefix(), interfaceParser.parser.token.Description))
				interfaceParser.consume(untilSemicolonOrLf)
				interfaceParser.Next(lfNotToken)
				continue
			}
			name = interfaceParser.parser.token.Data.(string)
			interfaceParser.Next(lfNotToken) // skip name
			functionType, err := interfaceParser.parser.parseFunctionType()
			if err != nil {
				interfaceParser.consume(untilSemicolonOrLf)
				interfaceParser.Next(lfNotToken)
				continue
			}
			interfaceParser.validStatementEnding()
			if interfaceParser.ret.Methods == nil {
				interfaceParser.ret.Methods = make(map[string][]*ast.ClassMethod)
			}
			m := &ast.ClassMethod{}
			m.Function = &ast.Function{}
			m.Function.Name = name
			m.Function.Type = functionType
			m.Function.AccessFlags |= cg.ACC_METHOD_PUBLIC
			m.Function.AccessFlags |= cg.ACC_METHOD_ABSTRACT
			if interfaceParser.ret.Methods == nil {
				interfaceParser.ret.Methods = make(map[string][]*ast.ClassMethod)
			}
			interfaceParser.ret.Methods[m.Function.Name] = append(interfaceParser.ret.Methods[m.Function.Name], m)
		case lex.TokenImport:
			pos := interfaceParser.parser.mkPos()
			interfaceParser.parser.parseImports()
			interfaceParser.parser.errs = append(interfaceParser.parser.errs, fmt.Errorf("%s cannot have import at this scope",
				interfaceParser.parser.errorMsgPrefix(pos)))
		default:
			interfaceParser.parser.errs = append(interfaceParser.parser.errs, fmt.Errorf("%s unexpect token:%s", interfaceParser.parser.errorMsgPrefix(),
				interfaceParser.parser.token.Description))
			interfaceParser.Next(lfNotToken)
		}
	}
	return
}
