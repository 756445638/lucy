package lex

import (
	"fmt"
	"math"
)

func New(bs []byte, startLine, startColumn int) *Lexer {
	lex := &Lexer{bs: bs}
	lex.end = len(bs)
	lex.line = startLine
	lex.column = startColumn
	lex.lastLine = startLine
	lex.lastColumn = startColumn
	return lex
}

type Lexer struct {
	bs                   []byte
	lastLine, lastColumn int
	line, column         int
	offset, end          int
}

func (lex *Lexer) GetLineAndColumn() (int, int) {
	return lex.line, lex.column
}

func (lex *Lexer) GetOffSet() int {
	return lex.offset
}

func (lex *Lexer) getChar() (c byte, eof bool) {
	if lex.offset == lex.end {
		eof = true
		return
	}
	offset := lex.offset
	lex.offset++
	c = lex.bs[offset]
	lex.lastLine = lex.line
	lex.lastColumn = lex.column
	if c == '\n' {
		lex.line++
		lex.column = 1
	} else {
		if c == '\t' {
			lex.column += 4 // TODO:: 4 OR 8
		} else {
			lex.column++
		}
	}
	return

}

func (lex *Lexer) unGetChar() {
	lex.offset--
	lex.line, lex.column = lex.lastLine, lex.lastColumn
}

func (lex *Lexer) isLetter(c byte) bool {
	return ('a' <= c && c <= 'z') ||
		('A' <= c && c <= 'Z')
}
func (lex *Lexer) isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
func (lex *Lexer) isOctal(c byte) bool {
	return '0' <= c && c <= '7'
}
func (lex *Lexer) isHex(c byte) bool {
	return '0' <= c && c <= '9' ||
		('a' <= c && c <= 'f') ||
		('A' <= c && c <= 'F')
}

func (lex *Lexer) hexByte2ByteValue(c byte) byte {
	if 'a' <= c && c <= 'f' {
		return c - 'a' + 10
	}
	if 'A' <= c && c <= 'F' {
		return c - 'A' + 10
	}
	return c - '0' //also valid for digit
}

func (lex *Lexer) parseInt(bs []byte) int64 {
	base := int64(10)
	if bs[0] == '0' {
		base = 8
	}
	if len(bs) >= 2 && bs[0] == '0' && (bs[1] == 'X' || bs[1] == 'x') { // correct base to hex
		base = 16
		bs = bs[2:]
	}
	var result int64 = 0
	for _, v := range bs {
		result = result*base + int64(lex.hexByte2ByteValue(v))
	}
	return result
}

func (lex *Lexer) lexNumber(token *Token, c byte) (eof bool, err error) {
	integerPart := []byte{c}
	isHex := false
	isOctal := false
	if c == '0' { // enter when first char is '0'
		c, eof = lex.getChar()
		if c == 'x' || c == 'X' {
			isHex = true
			integerPart = append(integerPart, 'X')
		} else {
			isOctal = true
			lex.unGetChar()
		}
	}
	c, eof = lex.getChar() //get next char
	for eof == false {
		ok := false
		if isHex {
			ok = lex.isHex(c)
		} else if isOctal {
			if lex.isDigit(c) == true && lex.isOctal(c) == false { // integer but not octal
				err = fmt.Errorf("octal number cannot be '8' and '9'")
			}
			ok = lex.isDigit(c)
		} else {
			ok = lex.isDigit(c)
		}
		if ok {
			integerPart = append(integerPart, c)
			c, eof = lex.getChar() // get next char
			continue
		} else { // something that I cannot handle
			lex.unGetChar()
			break
		}
	}
	c, eof = lex.getChar()
	floatPart := []byte{}
	isFloat := false // float or double
	if c == '.' {    // float numbers
		isFloat = true
		c, eof = lex.getChar()
		for eof == false {
			if lex.isDigit(c) {
				floatPart = append(floatPart, c)
				c, eof = lex.getChar()
			} else {
				lex.unGetChar()
				break
			}
		}
	} else {
		lex.unGetChar()
	}
	if isHex && isFloat {
		token.Type = TokenLiteralInt
		token.Data = 0
		err = fmt.Errorf("mix up float and hex")
		return
	}
	isDouble := false
	isLong := false
	isShort := false
	isByte := false
	c, eof = lex.getChar()
	if c == 'l' || c == 'L' {
		isLong = true
	} else if c == 'f' || c == 'F' {
		isFloat = true
	} else if c == 's' || c == 'S' {
		isShort = true
	} else if c == 'd' || c == 'D' {
		isDouble = true
	} else if c == 'b' || c == 'B' {
		isByte = true
	} else {
		lex.unGetChar()
	}
	isScientificNotation := false
	power := []byte{}
	powerPositive := true
	c, eof = lex.getChar()
	if (c == 'e' || c == 'E') && eof == false {
		isScientificNotation = true
		c, eof = lex.getChar()
		if eof {
			err = fmt.Errorf("unexpect EOF")
		}
		if c == '-' {
			powerPositive = false
			c, eof = lex.getChar()
		} else if lex.isDigit(c) {
			// nothing to do
		} else if c == '+' { // default is true
			c, eof = lex.getChar()
		} else {
			err = fmt.Errorf("wrong format scientific notation")
		}

		if lex.isDigit(c) == false {
			lex.unGetChar() //
			err = fmt.Errorf("wrong format scientific notation")
		} else {
			power = append(power, c)
			c, eof = lex.getChar()
			for eof == false && lex.isDigit(c) {
				power = append(power, c)
				c, eof = lex.getChar()
			}
			lex.unGetChar()
		}
	} else {
		lex.unGetChar()
	}
	if isHex && isScientificNotation {
		token.Type = TokenLiteralInt
		token.Data = 0
		token.Description = "0"
		err = fmt.Errorf("mix up hex and seientific notation")
		return
	}
	/*
		parse float part
	*/
	parseFloat := func(bs []byte) float64 {
		index := len(bs) - 1
		var fp float64
		for index >= 0 {
			fp = fp*0.1 + (float64(lex.hexByte2ByteValue(bs[index])) / 10.0)
			index--
		}
		return fp
	}
	token.EndLine = lex.line
	token.EndColumn = lex.column
	if isScientificNotation == false {
		if isFloat {
			value := parseFloat(floatPart) + float64(lex.parseInt(integerPart))
			if isDouble {
				token.Type = TokenLiteralDouble
				token.Data = value
			} else {
				token.Type = TokenLiteralFloat
				token.Data = float32(value)
			}
		} else {
			value := lex.parseInt(integerPart)
			if isLong {
				token.Type = TokenLiteralLong
				token.Data = value
			} else if isByte {
				token.Type = TokenLiteralByte
				token.Data = byte(value)
				if int32(value) > math.MaxUint8 {
					err = fmt.Errorf("max byte is %v", math.MaxUint8)
				}
			} else if isShort {
				token.Type = TokenLiteralShort
				token.Data = int32(value)
				if int32(value) > math.MaxInt16 {
					err = fmt.Errorf("max short is %v", math.MaxInt16)
				}
			} else {
				token.Type = TokenLiteralInt
				token.Data = int32(value)
			}
		}
		return
	}
	//scientific notation
	if t := lex.parseInt(integerPart); t > 10 || t < 1 {
		err = fmt.Errorf("wrong format scientific notation")
		token.Type = TokenLiteralInt
		token.Data = 0
		return
	}
	p := int(lex.parseInt(power))
	if powerPositive {
		if p >= len(floatPart) { // int
			integerPart = append(integerPart, floatPart...)
			b := make([]byte, p-len(floatPart))
			for k, _ := range b {
				b[k] = '0'
			}
			integerPart = append(integerPart, b...)
			value := lex.parseInt(integerPart)
			token.Type = TokenLiteralInt
			token.Data = int32(value)
		} else { // float
			integerPart = append(integerPart, floatPart[0:p]...)
			fmt.Println(floatPart[p:], parseFloat(floatPart[p:]))
			value := float64(lex.parseInt(integerPart)) + parseFloat(floatPart[p:])
			token.Type = TokenLiteralFloat
			token.Data = value
		}
	} else { // power is negative,must be float number
		b := make([]byte, p-len(integerPart))
		for k, _ := range b {
			b[k] = '0'
		}
		b = append(b, integerPart...)
		b = append(b, floatPart...)
		value := parseFloat(b)
		token.Type = TokenLiteralFloat
		token.Data = value
	}
	return
}
func (lex *Lexer) looksLikeT(bs []byte) bool {
	if len(bs) == 0 {
		return false
	}
	if bs[0] != 'T' {
		return false
	}
	bs = bs[1:]
	for _, v := range bs {
		if !(v >= '0' && v <= '9') {
			return false
		}
	}
	return true
}

func (lex *Lexer) lexIdentifier(c byte) (token *Token, err error) {
	token = &Token{}
	token.StartLine = lex.line
	token.StartColumn = lex.column - 1 // c is read
	bs := []byte{c}
	token.Offset = lex.offset - 1 // read
	c, eof := lex.getChar()
	for eof == false {
		if lex.isLetter(c) || c == '_' || lex.isDigit(c) || c == '$' {
			bs = append(bs, c)
			c, eof = lex.getChar()
		} else {
			lex.unGetChar()
			break
		}
	}
	token.EndLine = lex.line
	token.EndColumn = lex.column
	identifier := string(bs)
	if t, ok := keywordsMap[identifier]; ok {
		token.Type = t
		token.Description = identifier
		if token.Type == TokenElse {
			is := lex.tryLexElseIf()
			if is {
				token.Type = TokenElseif
				token.Description = "else if"
			}
		}
	} else {
		if lex.looksLikeT(bs) {
			token.Type = TokenTemplate
			token.Data = identifier
			token.Description = identifier
		} else {
			token.Type = TokenIdentifier
			token.Data = identifier
			token.Description = "identifier_" + identifier
		}
	}
	token.EndLine = lex.line
	token.EndColumn = lex.column
	return
}

func (lex *Lexer) tryLexElseIf() (is bool) {
	c, eof := lex.getChar()
	for (c == ' ' || c == '\t' || c == '\r') && eof == false {
		c, eof = lex.getChar()
	}
	if eof {
		return
	}
	if c != 'i' {
		lex.unGetChar()
		return
	}
	c, eof = lex.getChar()
	if c != 'f' {
		lex.unGetChar()
		lex.unGetChar()
		return
	}
	c, eof = lex.getChar()
	if c != ' ' && c != '\t' && c != '\r' { // white list
		lex.unGetChar()
		lex.unGetChar()
		lex.unGetChar()
		return
	}
	is = true
	return
}

func (lex *Lexer) lexString(endChar byte) (token *Token, err error) {
	token = &Token{}
	token.StartLine = lex.line
	token.StartColumn = lex.column
	token.Type = TokenLiteralString
	bs := []byte{}
	var c byte
	c, eof := lex.getChar()
	for c != endChar && c != '\n' && eof == false {
		if c != '\\' {
			bs = append(bs, c)
			c, eof = lex.getChar()
			continue
		}
		c, eof = lex.getChar() // get next char
		if eof {
			err = fmt.Errorf("unexpected EOF")
			break
		}
		switch c {
		case 'a':
			bs = append(bs, '\a')
			c, eof = lex.getChar()
		case 'b':
			bs = append(bs, '\b')
			c, eof = lex.getChar()
		case 'f':
			bs = append(bs, '\f')
			c, eof = lex.getChar()
		case 'n':
			bs = append(bs, '\n')
			c, eof = lex.getChar()
		case 'r':
			bs = append(bs, '\r')
			c, eof = lex.getChar()
		case 't':
			bs = append(bs, '\t')
			c, eof = lex.getChar()
		case 'v':
			bs = append(bs, '\v')
			c, eof = lex.getChar()
		case '\\':
			bs = append(bs, '\\')
			c, eof = lex.getChar()
		case '\'':
			bs = append(bs, '\'')
			c, eof = lex.getChar()
		case '"':
			bs = append(bs, '"')
			c, eof = lex.getChar()
		case 'x':
			var c1, c2 byte
			c1, eof = lex.getChar() //skip 'x'
			if eof {
				err = fmt.Errorf("unexpect EOF")
				continue
			}
			if !lex.isHex(c) {
				err = fmt.Errorf("unknown escape sequence")
				continue
			}
			b := lex.hexByte2ByteValue(c1)
			c2, eof = lex.getChar()
			if lex.isHex(c2) {
				if t := b*16 + lex.hexByte2ByteValue(c2); t < 127 { // only support standard ascii
					b = t
				} else {
					lex.unGetChar()
				}
			} else { //not hex
				lex.unGetChar()
			}
			bs = append(bs, b)
			c, eof = lex.getChar()
		case '0', '1', '2', '3', '4', '5', '7':
			// first char must be octal
			b := byte(0)
			for i := 0; i < 3; i++ {
				if eof {
					break
				}
				if lex.isOctal(c) == false {
					lex.unGetChar()
					break
				}
				if t := b*8 + lex.hexByte2ByteValue(c); t > 127 { // only support standard ascii
					lex.unGetChar()
					break
				} else {
					b = t
				}
				c, eof = lex.getChar()
			}
			bs = append(bs, b)
			c, eof = lex.getChar()
		default:
			err = fmt.Errorf("unknown escape sequence")
		}
	}
	if c == '\n' {
		err = fmt.Errorf("string literal start new line")
	}
	token.EndLine = lex.line
	token.EndColumn = lex.column
	token.Data = string(bs)
	token.Description = string(bs)
	return
}

func (lex *Lexer) lexMultiLineComment() {
redo:
	c, eof := lex.getChar()
	if eof {
		return
	}
	for c != '*' && eof == false {
		c, eof = lex.getChar()
	}
	if eof {
		return
	}
	c, eof = lex.getChar()
	if eof || c == '/' {
		return
	}
	goto redo
}

func (lex *Lexer) Next() (token *Token, err error) {
redo:
	token = &Token{}
	var c byte
	token.StartLine = lex.line
	token.StartColumn = lex.column
	c, eof := lex.getChar()
	if eof {
		token.Type = TokenEof
		token.Description = "EOF"
		return
	}
	for c == ' ' || c == '\t' || c == '\r' {
		token.StartLine = lex.line
		token.StartColumn = lex.column
		c, eof = lex.getChar()
	}
	if eof {
		token.Type = TokenEof
		token.Description = "EOF"
		return
	}
	if lex.isLetter(c) || c == '_' || c == '$' {
		return lex.lexIdentifier(c)
	}
	if lex.isDigit(c) {
		eof, err = lex.lexNumber(token, c)
		return
	}
	token.Offset = lex.offset
	switch c {
	case '?':
		token.Type = TokenQuestion
		token.Description = "?"
	case '(':
		token.Type = TokenLp
		token.Description = "("
	case ')':
		token.Type = TokenRp
		token.Description = ")"
	case '{':
		token.Type = TokenLc
		token.Description = "{"
	case '}':
		token.Type = TokenRc
		token.Description = "}"
	case '[':
		token.Type = TokenLb
		token.Description = "["
	case ']':
		token.Type = TokenRb
		token.Description = "]"
	case ';':
		token.Type = TokenSemicolon
		token.Description = ";"
	case ',':
		token.Type = TokenComma
		token.Description = ","
	case '&':
		c, eof = lex.getChar()
		if c == '&' {
			token.Type = TokenLogicalAnd
			token.Description = "&&"
		} else if c == '=' {
			token.Type = TokenAndAssign
			token.Description = "&="
		} else {
			lex.unGetChar()
			token.Type = TokenAnd
			token.Description = "&"
		}
	case '|':
		c, eof = lex.getChar()
		if c == '|' {
			token.Type = TokenLogicalOr
			token.Description = "||"
		} else if c == '=' {
			token.Type = TokenOrAssign
			token.Description = "|="
		} else {
			lex.unGetChar()
			token.Type = TokenOr
			token.Description = "|"
		}
	case '=':
		c, eof = lex.getChar()
		if c == '=' {
			token.Type = TokenEqual
			token.Description = "=="
		} else {
			lex.unGetChar()
			token.Type = TokenAssign
			token.Description = "="
		}
	case '!':
		c, eof = lex.getChar()
		if c == '=' {
			token.Type = TokenNe
			token.Description = "!="
		} else {
			lex.unGetChar()
			token.Type = TokenNot
			token.Description = "!"
		}
	case '>':
		c, eof = lex.getChar()
		if c == '=' {
			token.Type = TokenGe
			token.Description = ">="
		} else if c == '>' {
			c, eof = lex.getChar()
			if c == '=' {
				token.Type = TokenRshAssign
				token.Description = ">>="
			} else {
				lex.unGetChar()
				token.Type = TokenRsh
				token.Description = ">>"
			}
		} else {
			lex.unGetChar()
			token.Type = TokenGt
			token.Description = ">"
		}
	case '<':
		c, eof = lex.getChar()
		if c == '=' {
			token.Type = TokenLe
			token.Description = "<="
		} else if c == '<' {
			c, eof = lex.getChar()
			if c == '=' {
				token.Type = TokenLshAssign
				token.Description = "<<="
			} else {
				lex.unGetChar()
				token.Type = TokenLsh
				token.Description = "<<"
			}
		} else {
			lex.unGetChar()
			token.Type = TokenLt
			token.Description = "<"
		}
	case '^':
		c, eof = lex.getChar()
		if c == '=' {
			token.Type = TokenXorAssign
			token.Description = "^="
		} else {
			lex.unGetChar()
			token.Type = TokenXor
			token.Description = "^"
		}
	case '~':
		token.Type = TokenBitNot
		token.Description = "~"
	case '+':
		c, eof = lex.getChar()
		if c == '+' {
			token.Type = TokenIncrement
			token.Description = "++"
		} else if c == '=' {
			token.Type = TokenAddAssign
			token.Description = "+="
		} else {
			lex.unGetChar()
			token.Type = TokenAdd
			token.Description = "+"
		}
	case '-':
		c, eof = lex.getChar()
		if c == '-' {
			token.Type = TokenDecrement
			token.Description = "--"
		} else if c == '=' {
			token.Type = TokenSubAssign
			token.Description = "-="
		} else if c == '>' {
			token.Type = TokenArrow
			token.Description = "->"
		} else {
			lex.unGetChar()
			token.Type = TokenSub
			token.Description = "-"
		}
	case '*':
		c, eof = lex.getChar()
		if c == '=' {
			token.Type = TokenMulAssign
			token.Description = "*="
		} else {
			lex.unGetChar()
			token.Type = TokenMul
			token.Description = "*"
		}
	case '%':
		c, eof = lex.getChar()
		if c == '=' {
			token.Type = TokenModAssign
			token.Description = "%="
		} else {
			lex.unGetChar()
			token.Type = TokenMod
			token.Description = "%"
		}
	case '/':
		c, eof = lex.getChar()
		if c == '=' {
			token.Type = TokenDivAssign
			token.Description = "/="
		} else if c == '/' {
			for c != '\n' && eof == false {
				c, eof = lex.getChar()
			}
			goto redo
		} else if c == '*' {
			lex.lexMultiLineComment()
			goto redo
		} else {
			lex.unGetChar()
			token.Type = TokenDiv
			token.Description = "/"
		}
	case '\n':
		token.Type = TokenLf
		token.Description = "\\n"
	case '.':
		token.Type = TokenSelection
		token.Description = "."
	case '`':
		bs := []byte{}
		c, eof = lex.getChar()
		for c != '`' && eof == false {
			bs = append(bs, c)
			c, eof = lex.getChar()
		}
		token.Type = TokenLiteralString
		token.Data = string(bs)
		token.Description = string(bs)
	case '"':
		return lex.lexString('"')
	case '\'':
		token, err = lex.lexString('\'')
		if err == nil {
			if t := []byte(token.Data.(string)); len(t) != 1 {
				err = fmt.Errorf("expect one char")
			} else { // correct token
				token.Type = TokenLiteralByte
				token.Data = byte([]byte(t)[0])
			}
		}
		return
	case ':':
		c, eof = lex.getChar()
		if c == '=' {
			token.Type = TokenColonAssign
			token.Description = ":= "
		} else {
			token.Type = TokenColon
			token.Description = ":"
			lex.unGetChar()
		}
	default:
		err = fmt.Errorf("unkown beginning of token:%d", c)
		return
	}
	token.EndLine = lex.line
	token.EndColumn = lex.column
	return
}
