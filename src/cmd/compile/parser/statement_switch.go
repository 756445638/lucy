package parser

import (
	"fmt"

	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/lex"
)

//TODO:: missing format
func (blockParser *BlockParser) parseSwitchTemplate(pos *ast.Position) (*ast.StatementSwitchTemplate, error) {
	condition, err := blockParser.parser.parseType()
	if err != nil {
		blockParser.parser.errs = append(blockParser.parser.errs, err)
		blockParser.consume(untilLc)
	}
	if blockParser.parser.token.Type != lex.TokenLc {
		err = fmt.Errorf("%s expect '{',but '%s'",
			blockParser.parser.errorMsgPrefix(), blockParser.parser.token.Description)
		blockParser.parser.errs = append(blockParser.parser.errs, err)
		blockParser.consume(untilLc)
	}
	blockParser.Next(lfNotToken) // skip {  , must be case
	if blockParser.parser.token.Type != lex.TokenCase {
		err = fmt.Errorf("%s expect 'case',but '%s'",
			blockParser.parser.errorMsgPrefix(), blockParser.parser.token.Description)
		blockParser.parser.errs = append(blockParser.parser.errs, err)
		return nil, err
	}
	statementSwitchTemplate := &ast.StatementSwitchTemplate{}
	statementSwitchTemplate.Pos = pos
	statementSwitchTemplate.Condition = condition
	for blockParser.parser.token.Type == lex.TokenCase {
		blockParser.Next(lfNotToken) // skip case
		ts, err := blockParser.parser.parseTypes()
		if err != nil {
			blockParser.parser.errs = append(blockParser.parser.errs, err)
			return statementSwitchTemplate, err
		}
		if blockParser.parser.token.Type != lex.TokenColon {
			err = fmt.Errorf("%s expect ':',but '%s'",
				blockParser.parser.errorMsgPrefix(), blockParser.parser.token.Description)
			blockParser.parser.errs = append(blockParser.parser.errs, err)
			return statementSwitchTemplate, err
		}
		blockParser.Next(lfNotToken) // skip :
		var block *ast.Block
		if blockParser.parser.token.Type != lex.TokenCase &&
			blockParser.parser.token.Type != lex.TokenDefault &&
			blockParser.parser.token.Type != lex.TokenRc {
			block = &ast.Block{}
			block.IsSwitchBlock = true
			blockParser.parseStatementList(block, false)
		}
		statementSwitchTemplate.StatementSwitchCases = append(statementSwitchTemplate.StatementSwitchCases, &ast.StatementSwitchTemplateCase{
			Matches: ts,
			Block:   block,
		})
	}
	//default value
	if blockParser.parser.token.Type == lex.TokenDefault {
		blockParser.Next(lfIsToken) // skip default key word
		if blockParser.parser.token.Type != lex.TokenColon {
			err = fmt.Errorf("%s missing colon after default",
				blockParser.parser.errorMsgPrefix())
			blockParser.parser.errs = append(blockParser.parser.errs, err)
		} else {
			blockParser.Next(lfNotToken)
		}
		if blockParser.parser.token.Type != lex.TokenRc {
			block := ast.Block{}
			block.IsSwitchBlock = true
			blockParser.parseStatementList(&block, false)
			statementSwitchTemplate.Default = &block
		}
	}
	if blockParser.parser.token.Type != lex.TokenRc {
		err = fmt.Errorf("%s expect '}',but '%s'",
			blockParser.parser.errorMsgPrefix(), blockParser.parser.token.Description)
		blockParser.parser.errs = append(blockParser.parser.errs, err)
		return statementSwitchTemplate, err
	}
	blockParser.Next(lfNotToken) //  skip }
	return statementSwitchTemplate, nil
}

func (blockParser *BlockParser) parseSwitch() (interface{}, error) {
	pos := blockParser.parser.mkPos()
	blockParser.Next(lfIsToken) // skip switch key word
	blockParser.parser.unExpectNewLineAndSkip()
	if blockParser.parser.token.Type == lex.TokenTemplate {
		return blockParser.parseSwitchTemplate(pos)
	}
	condition, err := blockParser.parser.ExpressionParser.parseExpression(false)
	if err != nil {
		blockParser.parser.errs = append(blockParser.parser.errs, err)
		blockParser.consume(untilLc)
	}
	blockParser.parser.ifTokenIsLfSkip()
	if blockParser.parser.token.Type != lex.TokenLc {
		err = fmt.Errorf("%s expect '{',but '%s'",
			blockParser.parser.errorMsgPrefix(), blockParser.parser.token.Description)
		blockParser.parser.errs = append(blockParser.parser.errs, err)
		blockParser.consume(untilLc)
	}
	blockParser.Next(lfIsToken) // skip {  , must be case
	blockParser.parser.expectNewLineAndSkip()
	if blockParser.parser.token.Type != lex.TokenCase {
		err = fmt.Errorf("%s expect 'case',but '%s'",
			blockParser.parser.errorMsgPrefix(), blockParser.parser.token.Description)
		blockParser.parser.errs = append(blockParser.parser.errs, err)
		return nil, err
	}
	statementSwitch := &ast.StatementSwitch{}
	statementSwitch.Pos = pos
	statementSwitch.Condition = condition
	for blockParser.parser.token.Type == lex.TokenCase {
		blockParser.Next(lfIsToken) // skip case
		blockParser.parser.unExpectNewLineAndSkip()
		es, err := blockParser.parser.ExpressionParser.parseExpressions()
		if err != nil {
			blockParser.parser.errs = append(blockParser.parser.errs, err)
			return statementSwitch, err
		}
		if blockParser.parser.token.Type != lex.TokenColon {
			err = fmt.Errorf("%s expect ':',but '%s'",
				blockParser.parser.errorMsgPrefix(), blockParser.parser.token.Description)
			blockParser.parser.errs = append(blockParser.parser.errs, err)
			return statementSwitch, err
		}
		blockParser.Next(lfIsToken) // skip :
		blockParser.parser.expectNewLineAndSkip()
		var block *ast.Block
		if blockParser.parser.token.Type != lex.TokenCase &&
			blockParser.parser.token.Type != lex.TokenDefault &&
			blockParser.parser.token.Type != lex.TokenRc {
			block = &ast.Block{}
			block.IsSwitchBlock = true
			blockParser.parseStatementList(block, false)
		}
		statementSwitch.StatementSwitchCases = append(statementSwitch.StatementSwitchCases, &ast.StatementSwitchCase{
			Matches: es,
			Block:   block,
		})
	}
	//default value
	if blockParser.parser.token.Type == lex.TokenDefault {
		blockParser.Next(lfIsToken) // skip default key word
		blockParser.parser.unExpectNewLineAndSkip()
		if blockParser.parser.token.Type != lex.TokenColon {
			err = fmt.Errorf("%s missing colon after 'default'",
				blockParser.parser.errorMsgPrefix())
			blockParser.parser.errs = append(blockParser.parser.errs, err)
		} else {
			blockParser.Next(lfIsToken)
		}
		blockParser.parser.expectNewLineAndSkip()
		if blockParser.parser.token.Type != lex.TokenRc {
			block := ast.Block{}
			block.IsSwitchBlock = true
			blockParser.parseStatementList(&block, false)
			statementSwitch.Default = &block
		}
	}
	if blockParser.parser.token.Type != lex.TokenRc {
		err = fmt.Errorf("%s expect '}',but '%s'",
			blockParser.parser.errorMsgPrefix(), blockParser.parser.token.Description)
		blockParser.parser.errs = append(blockParser.parser.errs, err)
		return statementSwitch, err
	}
	blockParser.Next(lfNotToken) //  skip }
	return statementSwitch, nil
}
