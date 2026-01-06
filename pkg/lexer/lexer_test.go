package lexer

import (
	"testing"
)

func TestNextToken_Operators(t *testing.T) {
	input := `+ - * / \ ^ @ & = <> < > <= >= ->`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_PLUS, "+"},
		{TOKEN_MINUS, "-"},
		{TOKEN_ASTERISK, "*"},
		{TOKEN_SLASH, "/"},
		{TOKEN_BACKSLASH, "\\"},
		{TOKEN_CARET, "^"},
		{TOKEN_AT, "@"},
		{TOKEN_AMPERSAND, "&"},
		{TOKEN_ASSIGN, "="},
		{TOKEN_NEQ, "<>"},
		{TOKEN_LT, "<"},
		{TOKEN_GT, ">"},
		{TOKEN_LTE, "<="},
		{TOKEN_GTE, ">="},
		{TOKEN_ARROW, "->"},
		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_Delimiters(t *testing.T) {
	input := `( ) [ ] { } , : ; .`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_LPAREN, "("},
		{TOKEN_RPAREN, ")"},
		{TOKEN_LBRACKET, "["},
		{TOKEN_RBRACKET, "]"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_COMMA, ","},
		{TOKEN_COLON, ":"},
		{TOKEN_SEMICOLON, ";"},
		{TOKEN_DOT, "."},
		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_Keywords(t *testing.T) {
	input := `DIM AS SUB FUNCTION END IF THEN ELSE ELSEIF ENDIF
FOR TO STEP NEXT WHILE WEND DO LOOP UNTIL
RETURN IMPORT SPAWN CHANNEL SEND RECEIVE SELECT CASE
PRINT INPUT LET GOTO AND OR NOT MOD XOR
TRUE FALSE NIL CONST EXIT BYREF BYVAL
INTEGER LONG SINGLE DOUBLE STRING BOOLEAN JSON POINTER CHAN OF`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_DIM, "DIM"},
		{TOKEN_AS, "AS"},
		{TOKEN_SUB, "SUB"},
		{TOKEN_FUNCTION, "FUNCTION"},
		{TOKEN_END, "END"},
		{TOKEN_IF, "IF"},
		{TOKEN_THEN, "THEN"},
		{TOKEN_ELSE, "ELSE"},
		{TOKEN_ELSEIF, "ELSEIF"},
		{TOKEN_ENDIF, "ENDIF"},
		{TOKEN_NEWLINE, "\n"},
		{TOKEN_FOR, "FOR"},
		{TOKEN_TO, "TO"},
		{TOKEN_STEP, "STEP"},
		{TOKEN_NEXT, "NEXT"},
		{TOKEN_WHILE, "WHILE"},
		{TOKEN_WEND, "WEND"},
		{TOKEN_DO, "DO"},
		{TOKEN_LOOP, "LOOP"},
		{TOKEN_UNTIL, "UNTIL"},
		{TOKEN_NEWLINE, "\n"},
		{TOKEN_RETURN, "RETURN"},
		{TOKEN_IMPORT, "IMPORT"},
		{TOKEN_SPAWN, "SPAWN"},
		{TOKEN_CHANNEL, "CHANNEL"},
		{TOKEN_SEND, "SEND"},
		{TOKEN_RECEIVE, "RECEIVE"},
		{TOKEN_SELECT, "SELECT"},
		{TOKEN_CASE, "CASE"},
		{TOKEN_NEWLINE, "\n"},
		{TOKEN_PRINT, "PRINT"},
		{TOKEN_INPUT, "INPUT"},
		{TOKEN_LET, "LET"},
		{TOKEN_GOTO, "GOTO"},
		{TOKEN_AND, "AND"},
		{TOKEN_OR, "OR"},
		{TOKEN_NOT, "NOT"},
		{TOKEN_MOD, "MOD"},
		{TOKEN_XOR, "XOR"},
		{TOKEN_NEWLINE, "\n"},
		{TOKEN_TRUE, "TRUE"},
		{TOKEN_FALSE, "FALSE"},
		{TOKEN_NIL, "NIL"},
		{TOKEN_CONST, "CONST"},
		{TOKEN_EXIT, "EXIT"},
		{TOKEN_BYREF, "BYREF"},
		{TOKEN_BYVAL, "BYVAL"},
		{TOKEN_NEWLINE, "\n"},
		{TOKEN_INTEGER, "INTEGER"},
		{TOKEN_LONG, "LONG"},
		{TOKEN_SINGLE, "SINGLE"},
		{TOKEN_DOUBLE, "DOUBLE"},
		{TOKEN_STRING_TYPE, "STRING"},
		{TOKEN_BOOLEAN, "BOOLEAN"},
		{TOKEN_JSON, "JSON"},
		{TOKEN_POINTER, "POINTER"},
		{TOKEN_CHAN, "CHAN"},
		{TOKEN_OF, "OF"},
		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
	}
}

func TestNextToken_Identifiers(t *testing.T) {
	input := `myVar another_var var123 _private`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_IDENT, "myVar"},
		{TOKEN_IDENT, "another_var"},
		{TOKEN_IDENT, "var123"},
		{TOKEN_IDENT, "_private"},
		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_Numbers(t *testing.T) {
	input := `42 123 3.14 0.5 .25 2.5e-3`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_INT, "42"},
		{TOKEN_INT, "123"},
		{TOKEN_FLOAT, "3.14"},
		{TOKEN_FLOAT, "0.5"},
		{TOKEN_FLOAT, ".25"},
		{TOKEN_FLOAT, "2.5e-3"},
		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_Strings(t *testing.T) {
	input := `"hello" "world" "with spaces" "escape\nnewline" "escape\ttab"`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_STRING, "hello"},
		{TOKEN_STRING, "world"},
		{TOKEN_STRING, "with spaces"},
		{TOKEN_STRING, "escape\nnewline"},
		{TOKEN_STRING, "escape\ttab"},
		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_Comments(t *testing.T) {
	input := `DIM x AS INTEGER ' This is a comment
PRINT x`

	tests := []struct {
		expectedType TokenType
	}{
		{TOKEN_DIM},
		{TOKEN_IDENT},
		{TOKEN_AS},
		{TOKEN_INTEGER},
		{TOKEN_COMMENT},
		{TOKEN_NEWLINE},
		{TOKEN_PRINT},
		{TOKEN_IDENT},
		{TOKEN_EOF},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
	}
}

func TestNextToken_LineNumbers(t *testing.T) {
	input := `DIM x AS INTEGER
DIM y AS STRING
PRINT x`

	l := New(input)

	// Line 1
	tok := l.NextToken() // DIM
	if tok.Line != 1 {
		t.Errorf("expected line 1, got %d", tok.Line)
	}

	l.NextToken() // x
	l.NextToken() // AS
	l.NextToken() // INTEGER
	l.NextToken() // NEWLINE

	// Line 2
	tok = l.NextToken() // DIM
	if tok.Line != 2 {
		t.Errorf("expected line 2, got %d", tok.Line)
	}

	l.NextToken() // y
	l.NextToken() // AS
	l.NextToken() // STRING
	l.NextToken() // NEWLINE

	// Line 3
	tok = l.NextToken() // PRINT
	if tok.Line != 3 {
		t.Errorf("expected line 3, got %d", tok.Line)
	}
}

func TestNextToken_CaseInsensitiveKeywords(t *testing.T) {
	input := `dim Dim DIM DiM`

	l := New(input)

	for i := 0; i < 4; i++ {
		tok := l.NextToken()
		if tok.Type != TOKEN_DIM {
			t.Errorf("tests[%d] - expected DIM keyword, got %q", i, tok.Type)
		}
	}
}

func TestNextToken_CompleteProgram(t *testing.T) {
	input := `' Hello World in DBasic
SUB Main()
    PRINT "Hello, World!"
END SUB`

	expected := []TokenType{
		TOKEN_COMMENT,
		TOKEN_NEWLINE,
		TOKEN_SUB,
		TOKEN_IDENT,
		TOKEN_LPAREN,
		TOKEN_RPAREN,
		TOKEN_NEWLINE,
		TOKEN_PRINT,
		TOKEN_STRING,
		TOKEN_NEWLINE,
		TOKEN_END,
		TOKEN_SUB,
		TOKEN_EOF,
	}

	l := New(input)

	for i, expectedType := range expected {
		tok := l.NextToken()
		if tok.Type != expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, expectedType, tok.Type)
		}
	}
}

func TestGetSourceLine(t *testing.T) {
	input := `line one
line two
line three`

	l := New(input)

	if l.GetSourceLine(1) != "line one" {
		t.Errorf("expected 'line one', got '%s'", l.GetSourceLine(1))
	}
	if l.GetSourceLine(2) != "line two" {
		t.Errorf("expected 'line two', got '%s'", l.GetSourceLine(2))
	}
	if l.GetSourceLine(3) != "line three" {
		t.Errorf("expected 'line three', got '%s'", l.GetSourceLine(3))
	}
	if l.GetSourceLine(0) != "" {
		t.Errorf("expected empty string for line 0")
	}
	if l.GetSourceLine(4) != "" {
		t.Errorf("expected empty string for line 4")
	}
}
