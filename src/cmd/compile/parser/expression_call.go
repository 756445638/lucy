package parser

import (
	"fmt"

	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/lex"
)

func (expressionParser *ExpressionParser) parseCallExpression(on *ast.Expression) (*ast.Expression, error) {
	var err error
	pos := expressionParser.parser.mkPos()
	expressionParser.Next(lfNotToken) // skip (
	args := []*ast.Expression{}
	if expressionParser.parser.token.Type != lex.TokenRp { //a(123)
		args, err = expressionParser.parseExpressions(lex.TokenRp)
		if err != nil {
			return nil, err
		}
	}
	expressionParser.parser.unExpectNewLineAndSkip()
	if expressionParser.parser.token.Type != lex.TokenRp {
		return nil, fmt.Errorf("%s except ')' ,but '%s'",
			expressionParser.parser.errorMsgPrefix(),
			expressionParser.parser.token.Description)
	}
	expressionParser.Next(lfIsToken) // skip )
	result := &ast.Expression{}
	if on.Type == ast.ExpressionTypeSelection {
		/*
			x.x()
		*/
		result.Type = ast.ExpressionTypeMethodCall
		call := &ast.ExpressionMethodCall{}
		index := on.Data.(*ast.ExpressionSelection)
		call.Expression = index.Expression
		call.Args = args
		call.Name = index.Name
		result.Data = call
		result.Pos = expressionParser.parser.mkPos()
	} else {
		/*
			x()
		*/
		result.Type = ast.ExpressionTypeFunctionCall
		call := &ast.ExpressionFunctionCall{}
		call.Expression = on
		call.Args = args
		result.Data = call
		result.Pos = expressionParser.parser.mkPos()
	}

	if expressionParser.parser.token.Type == lex.TokenLt { // <
		/*
			template function call return type binds
			fn a ()->(r T) {

			}
			a<int , ... >
		*/
		expressionParser.Next(lfNotToken) // skip <
		ts, err := expressionParser.parser.parseTypes()
		if err != nil {

			return result, err
		}
		if expressionParser.parser.token.Type != lex.TokenGt {
			expressionParser.parser.errs = append(expressionParser.parser.errs, fmt.Errorf("%s '<' and '>' not match",
				expressionParser.parser.errorMsgPrefix()))
			expressionParser.parser.consume(untilGt)
		}
		expressionParser.Next(lfIsToken)
		if result.Type == ast.ExpressionTypeFunctionCall {
			result.Data.(*ast.ExpressionFunctionCall).ParameterTypes = ts
		} else {
			result.Data.(*ast.ExpressionMethodCall).ParameterTypes = ts
		}
	}
	result.Pos = pos
	return result, nil
}
