package parser

import (
	"fmt"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/lex"
)

func Parse(tops *[]*ast.Top, filename string, bs []byte, onlyParseImport bool, nErrors2Stop int) []error {
	p := &Parser{
		bs:              bs,
		tops:            tops,
		filename:        filename,
		onlyParseImport: onlyParseImport,
		nErrors2Stop:    nErrors2Stop,
	}
	return p.Parse()
}

type Parser struct {
	onlyParseImport             bool
	bs                          []byte
	tops                        *[]*ast.Top
	lexer                       *lex.Lexer
	filename                    string
	lastToken                   *lex.Token
	token                       *lex.Token
	errs                        []error
	importsByAccessName         map[string]*ast.Import
	importsByResourceName       map[string]*ast.Import
	nErrors2Stop                int
	consumeFoundValidStartToken bool
	// parsers
	ExpressionParser *ExpressionParser
	FunctionParser   *FunctionParser
	ClassParser      *ClassParser
	BlockParser      *BlockParser
	InterfaceParser  *InterfaceParser
}

/*
	call before parse source file
*/
func (parser *Parser) initParser() {
	parser.ExpressionParser = &ExpressionParser{parser}
	parser.FunctionParser = &FunctionParser{}
	parser.FunctionParser.parser = parser
	parser.ClassParser = &ClassParser{}
	parser.ClassParser.parser = parser
	parser.InterfaceParser = &InterfaceParser{}
	parser.InterfaceParser.parser = parser
	parser.BlockParser = &BlockParser{}
	parser.BlockParser.parser = parser
}

func (parser *Parser) Parse() []error {
	parser.initParser()
	parser.lexer = lex.New(parser.bs, 1, 1)
	parser.Next(lfNotToken) //
	if parser.token.Type == lex.TokenEof {
		//TODO::empty source file , should forbidden???
		return nil
	}
	for _, t := range parser.parseImports() {
		parser.insertImports(t)
	}
	if parser.onlyParseImport { // only parse imports
		return parser.errs
	}
	var accessControlToken *lex.Token
	isFinal := false
	resetProperty := func() {
		accessControlToken = nil
		isFinal = false
	}
	isPublic := func() bool {
		return accessControlToken != nil && accessControlToken.Type == lex.TokenPublic
	}
	for parser.token.Type != lex.TokenEof {
		if len(parser.errs) > parser.nErrors2Stop {
			break
		}

		switch parser.token.Type {
		case lex.TokenSemicolon, lex.TokenLf: // empty statement, no big deal
			parser.Next(lfNotToken)
			continue
		case lex.TokenPublic:
			accessControlToken = parser.token
			parser.Next(lfIsToken)
			if err := parser.validAfterPublic(); err != nil {
				accessControlToken = nil
			}
			continue
		case lex.TokenFinal:
			isFinal = true
			parser.Next(lfIsToken)
			if err := parser.validAfterFinal(); err != nil {
				isFinal = false
			}
			continue
		case lex.TokenVar:
			pos := parser.mkPos()
			parser.Next(lfIsToken) // skip var key word
			vs, es, err := parser.parseConstDefinition(true)
			if err != nil {
				parser.consume(untilSemicolonOrLf)
				parser.Next(lfNotToken)
				continue
			}
			d := &ast.ExpressionDeclareVariable{Variables: vs, InitValues: es}
			isPublic := isPublic()
			e := &ast.Expression{
				Type:     ast.ExpressionTypeVar,
				Data:     d,
				Pos:      pos,
				IsPublic: isPublic,
			}
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: e,
			})
			resetProperty()
		case lex.TokenIdentifier:
			e, err := parser.ExpressionParser.parseExpression(true)
			if err != nil {
				parser.errs = append(parser.errs, err)
				parser.consume(untilSemicolonOrLf)
				parser.Next(lfNotToken)
				continue
			}
			e.IsPublic = isPublic()
			parser.validStatementEnding()
			if e.Type == ast.ExpressionTypeVarAssign {
				*parser.tops = append(*parser.tops, &ast.Top{
					Data: e,
				})
			} else {
				parser.errs = append(parser.errs, fmt.Errorf("%s cannot have expression '%s' in top",
					parser.errorMsgPrefix(e.Pos), e.OpName()))
			}
			resetProperty()
		case lex.TokenEnum:
			e, err := parser.parseEnum()
			if err != nil {
				resetProperty()
				continue
			}
			isPublic := isPublic()
			if isPublic {
				e.AccessFlags |= cg.ACC_CLASS_PUBLIC
			}
			if e != nil {
				*parser.tops = append(*parser.tops, &ast.Top{
					Data: e,
				})
			}
			resetProperty()
		case lex.TokenFn:
			f, err := parser.FunctionParser.parse(true)
			if err != nil {
				parser.Next(lfNotToken)
				continue
			}
			isPublic := isPublic()
			if isPublic {
				f.AccessFlags |= cg.ACC_METHOD_PUBLIC
			}
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: f,
			})
			resetProperty()
		case lex.TokenLc:
			b := &ast.Block{}
			parser.Next(lfNotToken) // skip {
			parser.BlockParser.parseStatementList(b, true)
			if parser.token.Type != lex.TokenRc {
				parser.errs = append(parser.errs, fmt.Errorf("%s expect '}', but '%s'",
					parser.errorMsgPrefix(), parser.token.Description))
				parser.consume(untilRc)
			}
			parser.Next(lfNotToken) // skip }
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: b,
			})
		case lex.TokenClass, lex.TokenInterface:
			var c *ast.Class
			var err error
			if parser.token.Type == lex.TokenClass {
				c, err = parser.ClassParser.parse()
			} else {
				c, err = parser.InterfaceParser.parse()
			}
			if err != nil {
				resetProperty()
				continue
			}
			if c == nil && err == nil {
				panic(1)
			}
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: c,
			})
			if isPublic() {
				c.AccessFlags |= cg.ACC_CLASS_PUBLIC
			}
			if isFinal {
				c.AccessFlags |= cg.ACC_CLASS_FINAL
			}
			resetProperty()
		case lex.TokenConst:
			parser.Next(lfIsToken) // skip const key word
			vs, es, err := parser.parseConstDefinition(false)
			if err != nil {
				parser.consume(untilSemicolonOrLf)
				parser.Next(lfNotToken)
				resetProperty()
				continue
			}
			if len(vs) != len(es) {
				parser.errs = append(parser.errs,
					fmt.Errorf("%s cannot assign %d values to %d destinations",
						parser.errorMsgPrefix(parser.mkPos()), len(es), len(vs)))
			}
			isPublic := isPublic()
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
		case lex.TokenType:
			a, err := parser.parseTypeAlias()
			if err != nil {
				parser.consume(untilSemicolonOrLf)
				parser.Next(lfNotToken)
				resetProperty()
				continue
			}
			*parser.tops = append(*parser.tops, &ast.Top{
				Data: a,
			})
		case lex.TokenImport:
			pos := parser.mkPos()
			parser.parseImports()
			parser.errs = append(parser.errs, fmt.Errorf("%s cannot have import at this scope",
				parser.errorMsgPrefix(pos)))
		case lex.TokenEof:
			break
		default:
			if parser.ExpressionParser.looksLikeExpression() {
				e, err := parser.ExpressionParser.parseExpression(true)
				if err != nil {
					parser.errs = append(parser.errs, err)
					continue
				}
				if e.Type == ast.ExpressionTypeVarAssign {
					*parser.tops = append(*parser.tops, &ast.Top{
						Data: e,
					})
				} else {
					parser.errs = append(parser.errs, fmt.Errorf("%s cannot have expression '%s' in top",
						parser.errorMsgPrefix(e.Pos), e.OpName()))
				}
				continue
			}
			parser.errs = append(parser.errs, fmt.Errorf("%s token '%s' is not except",
				parser.errorMsgPrefix(), parser.token.Description))
			parser.consume(untilSemicolonOrLf)
			resetProperty()
		}
	}
	return parser.errs
}

func (parser *Parser) validAfterPublic() error {
	if parser.token.Type == lex.TokenFn ||
		parser.token.Type == lex.TokenClass ||
		parser.token.Type == lex.TokenEnum ||
		parser.token.Type == lex.TokenIdentifier ||
		parser.token.Type == lex.TokenInterface ||
		parser.token.Type == lex.TokenConst ||
		parser.token.Type == lex.TokenVar ||
		parser.token.Type == lex.TokenFinal {
		return nil
	}
	err := fmt.Errorf("%s cannot have token '%s' after 'public'",
		parser.errorMsgPrefix(), parser.token.Description)
	parser.errs = append(parser.errs, err)
	return err
}

func (parser *Parser) validAfterFinal() error {
	if parser.token.Type == lex.TokenClass ||
		parser.token.Type == lex.TokenInterface {
		return nil
	}
	err := fmt.Errorf("%s cannot have token '%s' after 'final'",
		parser.errorMsgPrefix(), parser.token.Description)
	parser.errs = append(parser.errs, err)
	return err
}

func (parser *Parser) shouldBeSemicolonOrLf() error {
	if parser.token.Type == lex.TokenSemicolon ||
		parser.token.Type == lex.TokenLf {
		return nil
	}
	token := parser.token
	err := fmt.Errorf("%s expect semicolon or new line", parser.errorMsgPrefix(&ast.Pos{
		Filename:    parser.filename,
		StartLine:   token.StartLine,
		StartColumn: token.StartColumn,
	}))
	parser.errs = append(parser.errs, err)
	return nil

}
func (parser *Parser) validStatementEnding() error {
	return parser.shouldBeSemicolonOrLf()
}

func (parser *Parser) mkPos() *ast.Pos {
	return &ast.Pos{
		Filename:    parser.filename,
		StartLine:   parser.token.StartLine,
		StartColumn: parser.token.StartColumn,
		Offset:      parser.lexer.GetOffSet(),
	}
}

func (parser *Parser) assignExpressionForConstants(vs []*ast.Variable, es []*ast.Expression) []*ast.Constant {
	if len(vs) != len(es) {
		parser.errs = append(parser.errs,
			fmt.Errorf("%s cannot assign %d values to %d destination",
				parser.errorMsgPrefix(vs[0].Pos), len(es), len(vs)))
	}
	cs := make([]*ast.Constant, len(vs))
	for k, v := range vs {
		c := &ast.Constant{}
		c.Variable = *v
		cs[k] = c
		if k < len(es) {
			cs[k].Expression = es[k] // assignment
		}
	}
	return cs
}

// str := "hello world"   a,b = 123 or a b ;
func (parser *Parser) parseConstDefinition(needType bool) ([]*ast.Variable, []*ast.Expression, error) {
	names, err := parser.parseNameList()
	if err != nil {
		return nil, nil, err
	}
	var variableType *ast.Type
	//trying to parse type
	if parser.isValidTypeBegin() || needType {
		variableType, err = parser.parseType()
		if err != nil {
			parser.errs = append(parser.errs, err)
			return nil, nil, err
		}
	}
	mkResult := func() []*ast.Variable {
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
	if parser.token.Type != lex.TokenAssign &&
		parser.token.Type != lex.TokenVarAssign {
		return mkResult(), nil, err
	}
	if parser.token.Type != lex.TokenAssign {
		parser.errs = append(parser.errs, fmt.Errorf("%s use '=' instead of ':='", parser.errorMsgPrefix()))
	}
	parser.Next(lfNotToken) // skip = or :=
	es, err := parser.ExpressionParser.parseExpressions(lex.TokenSemicolon)
	if err != nil {
		return nil, nil, err
	}
	return mkResult(), es, nil
}

func (parser *Parser) Next(lfIsToken bool) {
	if parser.consumeFoundValidStartToken {
		parser.consumeFoundValidStartToken = false
		return
	}
	var err error
	var tok *lex.Token
	parser.lastToken = parser.token
	for {
		tok, err = parser.lexer.Next()
		if err != nil {
			parser.errs = append(parser.errs, fmt.Errorf("%s %s", parser.errorMsgPrefix(), err.Error()))
		}
		if tok == nil {
			continue
		}
		parser.token = tok
		if lfIsToken {
			break
		}
		if tok.Type != lex.TokenLf {
			break
		}
	}
	return
}

/*
	errorMsgPrefix(pos) only receive one argument
*/
func (parser *Parser) errorMsgPrefix(pos ...*ast.Pos) string {
	var line, column int
	if len(pos) > 0 {
		line = pos[0].StartLine
		column = pos[0].StartColumn
	} else {
		line, column = parser.token.StartLine, parser.token.StartColumn
	}
	return fmt.Sprintf("%s:%d:%d", parser.filename, line, column)
}

func (parser *Parser) consume(until map[lex.TokenKind]bool) {
	if len(until) == 0 {
		panic("no token to consume")
	}
	for parser.token.Type != lex.TokenEof {
		if parser.token.Type == lex.TokenPublic ||
			parser.token.Type == lex.TokenProtected ||
			parser.token.Type == lex.TokenPrivate ||
			parser.token.Type == lex.TokenClass ||
			parser.token.Type == lex.TokenInterface ||
			parser.token.Type == lex.TokenFn ||
			parser.token.Type == lex.TokenFor ||
			parser.token.Type == lex.TokenIf ||
			parser.token.Type == lex.TokenSwitch ||
			parser.token.Type == lex.TokenEnum ||
			parser.token.Type == lex.TokenConst ||
			parser.token.Type == lex.TokenVar ||
			parser.token.Type == lex.TokenImport {
			parser.consumeFoundValidStartToken = true
			return
		}
		if parser.token.Type == lex.TokenLc {
			if _, ok := until[lex.TokenLc]; ok == false {
				parser.consumeFoundValidStartToken = true
				return
			}
		}
		if _, ok := until[parser.token.Type]; ok {
			return
		}
		parser.Next(lfIsToken)
	}
}

func (parser *Parser) ifTokenIsLfThenSkip() {
	if parser.token.Type == lex.TokenLf {
		parser.Next(lfNotToken)
	}
}

func (parser *Parser) unExpectNewLineAndSkip() {
	if err := parser.unExpectNewLine(); err != nil {
		parser.Next(lfNotToken)
	}
}
func (parser *Parser) unExpectNewLine() error {
	var err error
	if parser.token.Type == lex.TokenLf {
		err = fmt.Errorf("%s unexpect new line",
			parser.errorMsgPrefix(parser.mkPos()))
		parser.errs = append(parser.errs, err)
	}
	return err
}
func (parser *Parser) expectNewLineAndSkip() {
	if err := parser.expectNewLine(); err == nil {
		parser.Next(lfNotToken)
	}
}
func (parser *Parser) expectNewLine() error {
	var err error
	if parser.token.Type != lex.TokenLf {
		err = fmt.Errorf("%s expect new line , but '%s'",
			parser.errorMsgPrefix(), parser.token.Description)
		parser.errs = append(parser.errs, err)
	}
	return err
}

func (parser *Parser) parseTypeAlias() (*ast.TypeAlias, error) {
	parser.Next(lfIsToken) // skip type key word
	parser.unExpectNewLineAndSkip()
	if parser.token.Type != lex.TokenIdentifier {
		err := fmt.Errorf("%s expect identifer,but '%s'", parser.errorMsgPrefix(), parser.token.Description)
		parser.errs = append(parser.errs, err)
		return nil, err
	}
	ret := &ast.TypeAlias{}
	ret.Pos = parser.mkPos()
	ret.Name = parser.token.Data.(string)
	parser.Next(lfIsToken) // skip identifier
	if parser.token.Type != lex.TokenAssign {
		err := fmt.Errorf("%s expect '=',but '%s'", parser.errorMsgPrefix(), parser.token.Description)
		parser.errs = append(parser.errs, err)
		return nil, err
	}
	parser.Next(lfNotToken) // skip =
	var err error
	ret.Type, err = parser.parseType()
	if err != nil {
		return nil, err
	}
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
		vd.Type.Pos = v.Pos // override pos
	}
	return vs, nil
}

// a,b int or int,bool  c xxx
func (parser *Parser) parseTypedNames() (vs []*ast.Variable, err error) {
	vs = []*ast.Variable{}
	for parser.token.Type != lex.TokenEof {
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
		if parser.token.Type != lex.TokenComma { // not a comma
			break
		} else {
			parser.Next(lfNotToken)
		}
	}
	return vs, nil
}
