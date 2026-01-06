package lexer

import "fmt"

// TokenType represents the type of a lexical token
type TokenType int

const (
	// Special tokens
	TOKEN_EOF TokenType = iota
	TOKEN_ILLEGAL
	TOKEN_NEWLINE
	TOKEN_COMMENT

	// Literals
	TOKEN_IDENT       // identifier
	TOKEN_INT         // integer literal
	TOKEN_FLOAT       // float literal
	TOKEN_STRING      // string literal
	TOKEN_BYTE_STRING // byte string literal B"..."
	TOKEN_LABEL       // label (identifier followed by :)

	// Operators
	TOKEN_PLUS       // +
	TOKEN_MINUS      // -
	TOKEN_ASTERISK   // *
	TOKEN_SLASH      // /
	TOKEN_BACKSLASH  // \ (integer division)
	TOKEN_CARET      // ^ (pointer dereference / exponentiation)
	TOKEN_AT         // @ (address-of)
	TOKEN_AMPERSAND  // & (string concatenation)
	TOKEN_ASSIGN     // =
	TOKEN_EQ         // = (comparison)
	TOKEN_NEQ        // <>
	TOKEN_LT         // <
	TOKEN_GT         // >
	TOKEN_LTE        // <=
	TOKEN_GTE        // >=
	TOKEN_ARROW      // ->

	// Delimiters
	TOKEN_LPAREN     // (
	TOKEN_RPAREN     // )
	TOKEN_LBRACKET   // [
	TOKEN_RBRACKET   // ]
	TOKEN_LBRACE     // {
	TOKEN_RBRACE     // }
	TOKEN_COMMA      // ,
	TOKEN_COLON      // :
	TOKEN_SEMICOLON  // ;
	TOKEN_DOT        // .

	// Keywords - Declarations
	TOKEN_DIM
	TOKEN_AS
	TOKEN_LET
	TOKEN_CONST

	// Keywords - Types
	TOKEN_INTEGER
	TOKEN_LONG
	TOKEN_SINGLE
	TOKEN_DOUBLE
	TOKEN_STRING_TYPE
	TOKEN_BOOLEAN
	TOKEN_JSON
	TOKEN_BYTES
	TOKEN_BSTRING
	TOKEN_POINTER
	TOKEN_CHAN
	TOKEN_TO
	TOKEN_OF

	// Keywords - Control Flow
	TOKEN_IF
	TOKEN_THEN
	TOKEN_ELSE
	TOKEN_ELSEIF
	TOKEN_ENDIF
	TOKEN_FOR
	TOKEN_STEP
	TOKEN_NEXT
	TOKEN_WHILE
	TOKEN_WEND
	TOKEN_DO
	TOKEN_LOOP
	TOKEN_UNTIL
	TOKEN_EXIT
	TOKEN_SELECT
	TOKEN_CASE
	TOKEN_END
	TOKEN_GOTO
	TOKEN_GOSUB
	TOKEN_RETURN

	// Keywords - Functions/Subs
	TOKEN_SUB
	TOKEN_FUNCTION
	TOKEN_BYREF
	TOKEN_BYVAL

	// Keywords - Logical
	TOKEN_AND
	TOKEN_OR
	TOKEN_NOT
	TOKEN_XOR
	TOKEN_MOD

	// Keywords - Boolean/Nil
	TOKEN_TRUE
	TOKEN_FALSE
	TOKEN_NIL

	// Keywords - I/O
	TOKEN_PRINT
	TOKEN_INPUT

	// Keywords - Go Integration
	TOKEN_IMPORT
	TOKEN_SPAWN

	// Keywords - Channels
	TOKEN_CHANNEL
	TOKEN_SEND
	TOKEN_RECEIVE
	TOKEN_FROM
	TOKEN_MAKE_CHAN
)

var tokenNames = map[TokenType]string{
	TOKEN_EOF:         "EOF",
	TOKEN_ILLEGAL:     "ILLEGAL",
	TOKEN_NEWLINE:     "NEWLINE",
	TOKEN_COMMENT:     "COMMENT",
	TOKEN_IDENT:       "IDENT",
	TOKEN_INT:         "INT",
	TOKEN_FLOAT:       "FLOAT",
	TOKEN_STRING:      "STRING",
	TOKEN_BYTE_STRING: "BYTE_STRING",
	TOKEN_LABEL:       "LABEL",
	TOKEN_PLUS:        "+",
	TOKEN_MINUS:       "-",
	TOKEN_ASTERISK:    "*",
	TOKEN_SLASH:       "/",
	TOKEN_BACKSLASH:   "\\",
	TOKEN_CARET:       "^",
	TOKEN_AT:          "@",
	TOKEN_AMPERSAND:   "&",
	TOKEN_ASSIGN:      "=",
	TOKEN_EQ:          "==",
	TOKEN_NEQ:         "<>",
	TOKEN_LT:          "<",
	TOKEN_GT:          ">",
	TOKEN_LTE:         "<=",
	TOKEN_GTE:         ">=",
	TOKEN_ARROW:       "->",
	TOKEN_LPAREN:      "(",
	TOKEN_RPAREN:      ")",
	TOKEN_LBRACKET:    "[",
	TOKEN_RBRACKET:    "]",
	TOKEN_LBRACE:      "{",
	TOKEN_RBRACE:      "}",
	TOKEN_COMMA:       ",",
	TOKEN_COLON:       ":",
	TOKEN_SEMICOLON:   ";",
	TOKEN_DOT:         ".",
	TOKEN_DIM:         "DIM",
	TOKEN_AS:          "AS",
	TOKEN_LET:         "LET",
	TOKEN_CONST:       "CONST",
	TOKEN_INTEGER:     "INTEGER",
	TOKEN_LONG:        "LONG",
	TOKEN_SINGLE:      "SINGLE",
	TOKEN_DOUBLE:      "DOUBLE",
	TOKEN_STRING_TYPE: "STRING",
	TOKEN_BOOLEAN:     "BOOLEAN",
	TOKEN_JSON:        "JSON",
	TOKEN_BYTES:       "BYTES",
	TOKEN_BSTRING:     "BSTRING",
	TOKEN_POINTER:     "POINTER",
	TOKEN_CHAN:        "CHAN",
	TOKEN_TO:          "TO",
	TOKEN_OF:          "OF",
	TOKEN_IF:          "IF",
	TOKEN_THEN:        "THEN",
	TOKEN_ELSE:        "ELSE",
	TOKEN_ELSEIF:      "ELSEIF",
	TOKEN_ENDIF:       "ENDIF",
	TOKEN_FOR:         "FOR",
	TOKEN_STEP:        "STEP",
	TOKEN_NEXT:        "NEXT",
	TOKEN_WHILE:       "WHILE",
	TOKEN_WEND:        "WEND",
	TOKEN_DO:          "DO",
	TOKEN_LOOP:        "LOOP",
	TOKEN_UNTIL:       "UNTIL",
	TOKEN_EXIT:        "EXIT",
	TOKEN_SELECT:      "SELECT",
	TOKEN_CASE:        "CASE",
	TOKEN_END:         "END",
	TOKEN_GOTO:        "GOTO",
	TOKEN_GOSUB:       "GOSUB",
	TOKEN_RETURN:      "RETURN",
	TOKEN_SUB:         "SUB",
	TOKEN_FUNCTION:    "FUNCTION",
	TOKEN_BYREF:       "BYREF",
	TOKEN_BYVAL:       "BYVAL",
	TOKEN_AND:         "AND",
	TOKEN_OR:          "OR",
	TOKEN_NOT:         "NOT",
	TOKEN_XOR:         "XOR",
	TOKEN_MOD:         "MOD",
	TOKEN_TRUE:        "TRUE",
	TOKEN_FALSE:       "FALSE",
	TOKEN_NIL:         "NIL",
	TOKEN_PRINT:       "PRINT",
	TOKEN_INPUT:       "INPUT",
	TOKEN_IMPORT:      "IMPORT",
	TOKEN_SPAWN:       "SPAWN",
	TOKEN_CHANNEL:     "CHANNEL",
	TOKEN_SEND:        "SEND",
	TOKEN_RECEIVE:     "RECEIVE",
	TOKEN_FROM:        "FROM",
	TOKEN_MAKE_CHAN:   "MAKE_CHAN",
}

// Keywords maps keyword strings to token types
var Keywords = map[string]TokenType{
	"DIM":       TOKEN_DIM,
	"AS":        TOKEN_AS,
	"LET":       TOKEN_LET,
	"CONST":     TOKEN_CONST,
	"INTEGER":   TOKEN_INTEGER,
	"LONG":      TOKEN_LONG,
	"SINGLE":    TOKEN_SINGLE,
	"DOUBLE":    TOKEN_DOUBLE,
	"STRING":    TOKEN_STRING_TYPE,
	"BOOLEAN":   TOKEN_BOOLEAN,
	"JSON":      TOKEN_JSON,
	"BYTES":     TOKEN_BYTES,
	"BSTRING":   TOKEN_BSTRING,
	"POINTER":   TOKEN_POINTER,
	"CHAN":      TOKEN_CHAN,
	"TO":        TOKEN_TO,
	"OF":        TOKEN_OF,
	"IF":        TOKEN_IF,
	"THEN":      TOKEN_THEN,
	"ELSE":      TOKEN_ELSE,
	"ELSEIF":    TOKEN_ELSEIF,
	"ENDIF":     TOKEN_ENDIF,
	"END":       TOKEN_END,
	"FOR":       TOKEN_FOR,
	"STEP":      TOKEN_STEP,
	"NEXT":      TOKEN_NEXT,
	"WHILE":     TOKEN_WHILE,
	"WEND":      TOKEN_WEND,
	"DO":        TOKEN_DO,
	"LOOP":      TOKEN_LOOP,
	"UNTIL":     TOKEN_UNTIL,
	"EXIT":      TOKEN_EXIT,
	"SELECT":    TOKEN_SELECT,
	"CASE":      TOKEN_CASE,
	"GOTO":      TOKEN_GOTO,
	"GOSUB":     TOKEN_GOSUB,
	"RETURN":    TOKEN_RETURN,
	"SUB":       TOKEN_SUB,
	"FUNCTION":  TOKEN_FUNCTION,
	"BYREF":     TOKEN_BYREF,
	"BYVAL":     TOKEN_BYVAL,
	"AND":       TOKEN_AND,
	"OR":        TOKEN_OR,
	"NOT":       TOKEN_NOT,
	"XOR":       TOKEN_XOR,
	"MOD":       TOKEN_MOD,
	"TRUE":      TOKEN_TRUE,
	"FALSE":     TOKEN_FALSE,
	"NIL":       TOKEN_NIL,
	"PRINT":     TOKEN_PRINT,
	"INPUT":     TOKEN_INPUT,
	"IMPORT":    TOKEN_IMPORT,
	"SPAWN":     TOKEN_SPAWN,
	"CHANNEL":   TOKEN_CHANNEL,
	"SEND":      TOKEN_SEND,
	"RECEIVE":   TOKEN_RECEIVE,
	"FROM":      TOKEN_FROM,
	"MAKE_CHAN": TOKEN_MAKE_CHAN,
}

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TOKEN(%d)", t)
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("Token{Type: %s, Literal: %q, Line: %d, Col: %d}",
		t.Type, t.Literal, t.Line, t.Column)
}

// LookupIdent checks if an identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := Keywords[ident]; ok {
		return tok
	}
	return TOKEN_IDENT
}
