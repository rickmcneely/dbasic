package lexer

import (
	"strings"
	"unicode"
)

// Lexer tokenizes DBasic source code
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
	column       int  // current column number
	lines        []string // source lines for error reporting
}

// New creates a new Lexer for the given input
func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
		lines:  strings.Split(input, "\n"),
	}
	l.readChar()
	return l
}

// GetSourceLine returns the source line at the given line number (1-indexed)
func (l *Lexer) GetSourceLine(lineNum int) string {
	if lineNum < 1 || lineNum > len(l.lines) {
		return ""
	}
	return l.lines[lineNum-1]
}

// GetSource returns the full source code
func (l *Lexer) GetSource() string {
	return l.input
}

// readChar reads the next character and advances the position
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing the position
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '=':
		tok = l.newToken(TOKEN_ASSIGN, l.ch)
	case '+':
		tok = l.newToken(TOKEN_PLUS, l.ch)
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_ARROW, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = l.newToken(TOKEN_MINUS, l.ch)
		}
	case '*':
		tok = l.newToken(TOKEN_ASTERISK, l.ch)
	case '/':
		tok = l.newToken(TOKEN_SLASH, l.ch)
	case '\\':
		tok = l.newToken(TOKEN_BACKSLASH, l.ch)
	case '^':
		tok = l.newToken(TOKEN_CARET, l.ch)
	case '@':
		tok = l.newToken(TOKEN_AT, l.ch)
	case '&':
		tok = l.newToken(TOKEN_AMPERSAND, l.ch)
	case '<':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_NEQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_LTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = l.newToken(TOKEN_LT, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_GTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = l.newToken(TOKEN_GT, l.ch)
		}
	case '(':
		tok = l.newToken(TOKEN_LPAREN, l.ch)
	case ')':
		tok = l.newToken(TOKEN_RPAREN, l.ch)
	case '[':
		tok = l.newToken(TOKEN_LBRACKET, l.ch)
	case ']':
		tok = l.newToken(TOKEN_RBRACKET, l.ch)
	case '{':
		tok = l.newToken(TOKEN_LBRACE, l.ch)
	case '}':
		tok = l.newToken(TOKEN_RBRACE, l.ch)
	case ',':
		tok = l.newToken(TOKEN_COMMA, l.ch)
	case ':':
		tok = l.newToken(TOKEN_COLON, l.ch)
	case ';':
		tok = l.newToken(TOKEN_SEMICOLON, l.ch)
	case '.':
		if isDigit(l.peekChar()) {
			tok.Type = TOKEN_FLOAT
			tok.Literal = l.readNumber()
			return tok
		}
		tok = l.newToken(TOKEN_DOT, l.ch)
	case '"':
		tok.Type = TOKEN_STRING
		tok.Literal = l.readString()
		tok.Line = l.line
		tok.Column = l.column
		return tok
	case '\'':
		// Comment - read until end of line
		tok.Type = TOKEN_COMMENT
		tok.Literal = l.readComment()
		return tok
	case '\n':
		tok = l.newToken(TOKEN_NEWLINE, l.ch)
	case 0:
		tok.Literal = ""
		tok.Type = TOKEN_EOF
	default:
		if isLetter(l.ch) {
			// Check for B"..." byte string literal
			if (l.ch == 'B' || l.ch == 'b') && l.peekChar() == '"' {
				l.readChar() // skip B
				tok.Type = TOKEN_BYTE_STRING
				tok.Literal = l.readString()
				tok.Line = l.line
				tok.Column = l.column
				return tok
			}
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(strings.ToUpper(tok.Literal))
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else if isDigit(l.ch) {
			tok.Literal = l.readNumber()
			if strings.Contains(tok.Literal, ".") {
				tok.Type = TOKEN_FLOAT
			} else {
				tok.Type = TOKEN_INT
			}
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else {
			tok = l.newToken(TOKEN_ILLEGAL, l.ch)
		}
	}

	l.readChar()
	return tok
}

// newToken creates a new token with the given type and character
func (l *Lexer) newToken(tokenType TokenType, ch byte) Token {
	return Token{
		Type:    tokenType,
		Literal: string(ch),
		Line:    l.line,
		Column:  l.column,
	}
}

// skipWhitespace skips whitespace characters (but not newlines)
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier reads an identifier
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number (integer or float)
func (l *Lexer) readNumber() string {
	position := l.position
	hasDecimal := false

	// Handle leading decimal point
	if l.ch == '.' {
		hasDecimal = true
		l.readChar()
	}

	for isDigit(l.ch) {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && !hasDecimal {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	// Check for exponent
	if l.ch == 'e' || l.ch == 'E' {
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position]
}

// readString reads a string literal
func (l *Lexer) readString() string {
	var sb strings.Builder
	l.readChar() // skip opening quote

	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(l.ch)
			}
		} else {
			sb.WriteByte(l.ch)
		}
		l.readChar()
	}

	l.readChar() // skip closing quote
	return sb.String()
}

// readComment reads a comment until end of line
func (l *Lexer) readComment() string {
	position := l.position + 1 // skip the '
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return strings.TrimSpace(l.input[position:l.position])
}

// isLetter checks if a character is a letter
func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

// isDigit checks if a character is a digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// Tokenize returns all tokens from the input
func (l *Lexer) Tokenize() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TOKEN_EOF {
			break
		}
	}
	return tokens
}
