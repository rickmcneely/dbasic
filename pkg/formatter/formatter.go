// Package formatter provides a comment-preserving source formatter for DBasic.
// It re-emits the token stream with consistent indentation and spacing while
// keeping comments and blank-line groupings intact.
package formatter

import (
	"strings"

	"github.com/zditech/dbasic/pkg/lexer"
)

const indentUnit = "    "

// Format takes DBasic source and returns the formatted source.
func Format(src string) (string, error) {
	l := lexer.New(src)
	return formatTokens(l.Tokenize()), nil
}

// blockKind tracks open scopes for indentation. We push on opener lines and
// pop on closer lines. The single-byte tag identifies the kind so closers
// only pop matching scopes.
const (
	bkSub    byte = 's'
	bkType   byte = 't'
	bkIf     byte = 'i'
	bkFor    byte = 'f'
	bkWhile  byte = 'w'
	bkDo     byte = 'd'
	bkSelect byte = 'S'
	bkCase   byte = 'c'
)

func formatTokens(tokens []lexer.Token) string {
	var out strings.Builder
	var stack []byte
	pos := 0
	pendingBlank := false
	haveOutput := false

	popKind := func(b byte) {
		if len(stack) > 0 && stack[len(stack)-1] == b {
			stack = stack[:len(stack)-1]
		}
	}

	for pos < len(tokens) && tokens[pos].Type != lexer.TOKEN_EOF {
		// Blank lines (collapse runs of NEWLINE into at most one blank).
		if tokens[pos].Type == lexer.TOKEN_NEWLINE {
			if haveOutput {
				pendingBlank = true
			}
			pos++
			continue
		}

		// Collect a logical line (until NEWLINE or EOF).
		var line []lexer.Token
		for pos < len(tokens) {
			t := tokens[pos]
			if t.Type == lexer.TOKEN_NEWLINE {
				pos++
				break
			}
			if t.Type == lexer.TOKEN_EOF {
				break
			}
			line = append(line, t)
			pos++
		}
		if len(line) == 0 {
			continue
		}

		// Find first significant (non-comment) token.
		leadIdx := -1
		for i, t := range line {
			if t.Type != lexer.TOKEN_COMMENT {
				leadIdx = i
				break
			}
		}

		if leadIdx == -1 {
			// Pure comment line: render at current indent.
			if pendingBlank && haveOutput {
				out.WriteString("\n")
				pendingBlank = false
			}
			writeIndent(&out, len(stack))
			out.WriteString(renderTokens(line))
			out.WriteString("\n")
			haveOutput = true
			continue
		}

		lead := line[leadIdx]

		// Adjust indent for closer / mid-block lines BEFORE rendering.
		switch lead.Type {
		case lexer.TOKEN_END:
			second := secondSig(line, leadIdx)
			switch second {
			case lexer.TOKEN_SUB, lexer.TOKEN_FUNCTION:
				popKind(bkSub)
			case lexer.TOKEN_TYPE:
				popKind(bkType)
			case lexer.TOKEN_IF:
				popKind(bkIf)
			case lexer.TOKEN_SELECT:
				if len(stack) > 0 && stack[len(stack)-1] == bkCase {
					stack = stack[:len(stack)-1]
				}
				popKind(bkSelect)
			default:
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
		case lexer.TOKEN_ENDIF:
			popKind(bkIf)
		case lexer.TOKEN_NEXT:
			popKind(bkFor)
		case lexer.TOKEN_WEND:
			popKind(bkWhile)
		case lexer.TOKEN_LOOP:
			popKind(bkDo)
		case lexer.TOKEN_CASE:
			if len(stack) > 0 && stack[len(stack)-1] == bkCase {
				stack = stack[:len(stack)-1]
			}
		}

		renderIndent := len(stack)
		switch lead.Type {
		case lexer.TOKEN_ELSE, lexer.TOKEN_ELSEIF:
			if renderIndent > 0 {
				renderIndent--
			}
		}

		if pendingBlank && haveOutput {
			out.WriteString("\n")
			pendingBlank = false
		}
		writeIndent(&out, renderIndent)
		out.WriteString(renderTokens(line))
		out.WriteString("\n")
		haveOutput = true

		// Apply post-line opener pushes.
		switch lead.Type {
		case lexer.TOKEN_SUB, lexer.TOKEN_FUNCTION:
			stack = append(stack, bkSub)
		case lexer.TOKEN_TYPE:
			// Only treat as opener if followed by an identifier (TYPE Foo)
			if leadIdx+1 < len(line) && line[leadIdx+1].Type == lexer.TOKEN_IDENT {
				stack = append(stack, bkType)
			}
		case lexer.TOKEN_IF:
			if endsWithThen(line) {
				stack = append(stack, bkIf)
			}
		case lexer.TOKEN_FOR:
			stack = append(stack, bkFor)
		case lexer.TOKEN_WHILE:
			stack = append(stack, bkWhile)
		case lexer.TOKEN_DO:
			stack = append(stack, bkDo)
		case lexer.TOKEN_SELECT:
			stack = append(stack, bkSelect)
		case lexer.TOKEN_CASE:
			stack = append(stack, bkCase)
		}
	}

	return out.String()
}

func secondSig(line []lexer.Token, after int) lexer.TokenType {
	for j := after + 1; j < len(line); j++ {
		if line[j].Type != lexer.TOKEN_COMMENT {
			return line[j].Type
		}
	}
	return 0
}

func endsWithThen(line []lexer.Token) bool {
	for i := len(line) - 1; i >= 0; i-- {
		if line[i].Type == lexer.TOKEN_COMMENT {
			continue
		}
		return line[i].Type == lexer.TOKEN_THEN
	}
	return false
}

func writeIndent(sb *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		sb.WriteString(indentUnit)
	}
}

func renderTokens(line []lexer.Token) string {
	var sb strings.Builder
	var prev, prev2 lexer.Token
	havePrev := false
	havePrev2 := false
	for _, t := range line {
		if t.Type == lexer.TOKEN_COMMENT {
			if havePrev {
				sb.WriteString("  ' ")
			} else {
				sb.WriteString("' ")
			}
			sb.WriteString(t.Literal)
			continue
		}
		sep := separator(prev, t, havePrev)
		// Suppress the space after a unary minus: when the previous emitted
		// token is MINUS and what came before MINUS is an operator-like
		// context (`=`, `(`, `,`, return, etc.), the MINUS is unary and
		// should glue to the operand.
		if prev.Type == lexer.TOKEN_MINUS && havePrev2 && isUnaryContext(prev2.Type) {
			sep = ""
		}
		sb.WriteString(sep)
		sb.WriteString(literal(t))
		prev2 = prev
		prev = t
		havePrev2 = havePrev
		havePrev = true
	}
	return sb.String()
}

func isUnaryContext(t lexer.TokenType) bool {
	switch t {
	case lexer.TOKEN_ASSIGN, lexer.TOKEN_EQ, lexer.TOKEN_NEQ,
		lexer.TOKEN_LT, lexer.TOKEN_GT, lexer.TOKEN_LTE, lexer.TOKEN_GTE,
		lexer.TOKEN_PLUS, lexer.TOKEN_MINUS, lexer.TOKEN_ASTERISK,
		lexer.TOKEN_SLASH, lexer.TOKEN_BACKSLASH, lexer.TOKEN_AMPERSAND,
		lexer.TOKEN_LPAREN, lexer.TOKEN_LBRACKET, lexer.TOKEN_LBRACE,
		lexer.TOKEN_COMMA, lexer.TOKEN_COLON, lexer.TOKEN_SEMICOLON,
		lexer.TOKEN_AND, lexer.TOKEN_OR, lexer.TOKEN_NOT, lexer.TOKEN_XOR, lexer.TOKEN_MOD,
		lexer.TOKEN_RETURN, lexer.TOKEN_THEN, lexer.TOKEN_ELSE,
		lexer.TOKEN_TO, lexer.TOKEN_STEP, lexer.TOKEN_OF, lexer.TOKEN_AS,
		lexer.TOKEN_PRINT, lexer.TOKEN_DIM, lexer.TOKEN_LET, lexer.TOKEN_CONST:
		return true
	}
	return false
}

func separator(prev, curr lexer.Token, havePrev bool) string {
	if !havePrev {
		return ""
	}

	// No space before these tokens.
	switch curr.Type {
	case lexer.TOKEN_RPAREN, lexer.TOKEN_RBRACKET, lexer.TOKEN_RBRACE,
		lexer.TOKEN_COMMA, lexer.TOKEN_SEMICOLON, lexer.TOKEN_DOT, lexer.TOKEN_COLON:
		return ""
	}

	// No space after these tokens.
	switch prev.Type {
	case lexer.TOKEN_LPAREN, lexer.TOKEN_LBRACKET, lexer.TOKEN_LBRACE,
		lexer.TOKEN_DOT, lexer.TOKEN_AT:
		return ""
	}

	// Function call / subscript: IDENT/RPAREN/RBRACKET/CARET followed by ( or [
	if curr.Type == lexer.TOKEN_LPAREN || curr.Type == lexer.TOKEN_LBRACKET {
		switch prev.Type {
		case lexer.TOKEN_IDENT, lexer.TOKEN_RPAREN, lexer.TOKEN_RBRACKET, lexer.TOKEN_CARET:
			return ""
		}
	}

	// Postfix CARET (pointer deref): IDENT^ stays glued.
	if curr.Type == lexer.TOKEN_CARET {
		switch prev.Type {
		case lexer.TOKEN_IDENT, lexer.TOKEN_RPAREN, lexer.TOKEN_RBRACKET:
			return ""
		}
	}

	return " "
}

func literal(t lexer.Token) string {
	switch t.Type {
	case lexer.TOKEN_STRING:
		return "\"" + escapeString(t.Literal) + "\""
	case lexer.TOKEN_BYTE_STRING:
		return "B\"" + escapeString(t.Literal) + "\""
	}
	if isKeywordType(t.Type) {
		return strings.ToUpper(t.Literal)
	}
	return t.Literal
}

func isKeywordType(t lexer.TokenType) bool {
	return t >= lexer.TOKEN_DIM && t <= lexer.TOKEN_MAKE_CHAN
}

func escapeString(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			sb.WriteString("\\\\")
		case '"':
			sb.WriteString("\\\"")
		case '\n':
			sb.WriteString("\\n")
		case '\t':
			sb.WriteString("\\t")
		case '\r':
			sb.WriteString("\\r")
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}
