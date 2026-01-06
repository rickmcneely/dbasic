package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zditech/dbasic/pkg/lexer"
)

// Operator precedence levels
const (
	LOWEST int = iota
	OR_PREC
	AND_PREC
	NOT_PREC
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	POWER
	PREFIX
	CALL
	INDEX
)

var precedences = map[lexer.TokenType]int{
	lexer.TOKEN_OR:        OR_PREC,
	lexer.TOKEN_XOR:       OR_PREC,
	lexer.TOKEN_AND:       AND_PREC,
	lexer.TOKEN_EQ:        EQUALS,
	lexer.TOKEN_ASSIGN:    EQUALS,
	lexer.TOKEN_NEQ:       EQUALS,
	lexer.TOKEN_LT:        LESSGREATER,
	lexer.TOKEN_GT:        LESSGREATER,
	lexer.TOKEN_LTE:       LESSGREATER,
	lexer.TOKEN_GTE:       LESSGREATER,
	lexer.TOKEN_PLUS:      SUM,
	lexer.TOKEN_MINUS:     SUM,
	lexer.TOKEN_AMPERSAND: SUM,
	lexer.TOKEN_ASTERISK:  PRODUCT,
	lexer.TOKEN_SLASH:     PRODUCT,
	lexer.TOKEN_BACKSLASH: PRODUCT,
	lexer.TOKEN_MOD:       PRODUCT,
	lexer.TOKEN_CARET:     POWER,
	lexer.TOKEN_LPAREN:    CALL,
	lexer.TOKEN_LBRACKET:  INDEX,
	lexer.TOKEN_DOT:       INDEX,
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

// Parser parses DBasic source code into an AST
type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  lexer.Token
	peekToken lexer.Token

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

// formatError creates a formatted error message with source context
func (p *Parser) formatError(line, column int, message string, hint string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("parse error at line %d", line))
	if column > 0 {
		sb.WriteString(fmt.Sprintf(", column %d", column))
	}
	sb.WriteString(": ")
	sb.WriteString(message)
	sb.WriteString("\n")

	// Get source line and show context
	sourceLine := p.l.GetSourceLine(line)
	if sourceLine != "" {
		sb.WriteString(fmt.Sprintf("  %d | %s\n", line, sourceLine))
		if column > 0 {
			lineNumWidth := len(fmt.Sprintf("%d", line))
			padding := lineNumWidth + 3 + column - 1
			sb.WriteString(strings.Repeat(" ", padding))
			sb.WriteString("^\n")
		}
	}

	if hint != "" {
		sb.WriteString(fmt.Sprintf("  hint: %s\n", hint))
	}

	return sb.String()
}

// New creates a new Parser
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.TOKEN_IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.TOKEN_INT, p.parseIntegerLiteral)
	p.registerPrefix(lexer.TOKEN_FLOAT, p.parseFloatLiteral)
	p.registerPrefix(lexer.TOKEN_STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TOKEN_TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.TOKEN_FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.TOKEN_NIL, p.parseNilLiteral)
	p.registerPrefix(lexer.TOKEN_LBRACE, p.parseJSONLiteral)
	p.registerPrefix(lexer.TOKEN_LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(lexer.TOKEN_LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer.TOKEN_MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.TOKEN_NOT, p.parsePrefixExpression)
	p.registerPrefix(lexer.TOKEN_AT, p.parseAddressOf)
	p.registerPrefix(lexer.TOKEN_CARET, p.parseDereference)
	p.registerPrefix(lexer.TOKEN_MAKE_CHAN, p.parseMakeChan)

	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.TOKEN_PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_BACKSLASH, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_CARET, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_AMPERSAND, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_MOD, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_ASSIGN, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_NEQ, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_LT, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_GT, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_LTE, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_GTE, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_AND, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_OR, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_XOR, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.TOKEN_LBRACKET, p.parseIndexExpression)
	p.registerInfix(lexer.TOKEN_DOT, p.parseMemberExpression)

	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	var hint string
	switch t {
	case lexer.TOKEN_RPAREN:
		hint = "check for unmatched parentheses"
	case lexer.TOKEN_THEN:
		hint = "IF statements require THEN after the condition"
	case lexer.TOKEN_AS:
		hint = "variable declarations require AS followed by a type"
	case lexer.TOKEN_ASSIGN:
		hint = "use = for assignment"
	case lexer.TOKEN_IDENT:
		hint = "expected an identifier (variable or function name)"
	}

	msg := p.formatError(
		p.peekToken.Line,
		p.peekToken.Column,
		fmt.Sprintf("expected %s, got %s instead", t, p.peekToken.Type),
		hint,
	)
	p.errors = append(p.errors, msg)
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

// skipNewlines skips any newline tokens
func (p *Parser) skipNewlines() {
	for p.curTokenIs(lexer.TOKEN_NEWLINE) || p.curTokenIs(lexer.TOKEN_COMMENT) {
		p.nextToken()
	}
}

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *Program {
	program := &Program{}
	program.Statements = []Statement{}

	for !p.curTokenIs(lexer.TOKEN_EOF) {
		p.skipNewlines()
		if p.curTokenIs(lexer.TOKEN_EOF) {
			break
		}

		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case lexer.TOKEN_IMPORT:
		return p.parseImportStatement()
	case lexer.TOKEN_DIM:
		return p.parseDimStatement()
	case lexer.TOKEN_LET:
		return p.parseLetStatement()
	case lexer.TOKEN_CONST:
		return p.parseConstStatement()
	case lexer.TOKEN_PRINT:
		return p.parsePrintStatement()
	case lexer.TOKEN_INPUT:
		return p.parseInputStatement()
	case lexer.TOKEN_IF:
		return p.parseIfStatement()
	case lexer.TOKEN_FOR:
		return p.parseForStatement()
	case lexer.TOKEN_WHILE:
		return p.parseWhileStatement()
	case lexer.TOKEN_DO:
		return p.parseDoLoopStatement()
	case lexer.TOKEN_SELECT:
		return p.parseSelectStatement()
	case lexer.TOKEN_SUB:
		return p.parseSubStatement()
	case lexer.TOKEN_FUNCTION:
		return p.parseFunctionStatement()
	case lexer.TOKEN_RETURN:
		return p.parseReturnStatement()
	case lexer.TOKEN_EXIT:
		return p.parseExitStatement()
	case lexer.TOKEN_GOTO:
		return p.parseGotoStatement()
	case lexer.TOKEN_SPAWN:
		return p.parseSpawnStatement()
	case lexer.TOKEN_SEND:
		return p.parseSendStatement()
	case lexer.TOKEN_RECEIVE:
		return p.parseReceiveStatement()
	case lexer.TOKEN_IDENT:
		// Check if it's a label (identifier followed by colon)
		if p.peekTokenIs(lexer.TOKEN_COLON) {
			return p.parseLabelStatement()
		}
		// Otherwise it's an assignment or expression
		return p.parseAssignmentOrExpression()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseImportStatement() *ImportStatement {
	stmt := &ImportStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_STRING) {
		return nil
	}

	stmt.Package = p.curToken.Literal

	// Check for optional AS alias
	if p.peekTokenIs(lexer.TOKEN_AS) {
		p.nextToken()
		if !p.expectPeek(lexer.TOKEN_IDENT) {
			return nil
		}
		stmt.Alias = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseDimStatement() *DimStatement {
	stmt := &DimStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for array declaration
	if p.peekTokenIs(lexer.TOKEN_LPAREN) {
		p.nextToken()
		p.nextToken()
		stmt.ArraySize = p.parseExpression(LOWEST)
		if !p.expectPeek(lexer.TOKEN_RPAREN) {
			return nil
		}
	}

	if !p.expectPeek(lexer.TOKEN_AS) {
		return nil
	}

	p.nextToken()
	stmt.Type = p.parseTypeSpec()

	// Check for initial value
	if p.peekTokenIs(lexer.TOKEN_ASSIGN) {
		p.nextToken()
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)
	}

	return stmt
}

func (p *Parser) parseLetStatement() *LetStatement {
	stmt := &LetStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseConstStatement() *ConstStatement {
	stmt := &ConstStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_AS) {
		return nil
	}

	p.nextToken()
	stmt.Type = p.parseTypeSpec()

	if !p.expectPeek(lexer.TOKEN_ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseTypeSpec() *TypeSpec {
	spec := &TypeSpec{Token: p.curToken}

	switch p.curToken.Type {
	case lexer.TOKEN_POINTER:
		spec.IsPointer = true
		if !p.expectPeek(lexer.TOKEN_TO) {
			return nil
		}
		p.nextToken()
		spec.ElementType = p.parseTypeSpec()
	case lexer.TOKEN_CHAN:
		spec.IsChannel = true
		if !p.expectPeek(lexer.TOKEN_OF) {
			return nil
		}
		p.nextToken()
		spec.ElementType = p.parseTypeSpec()
	default:
		spec.Name = strings.ToUpper(p.curToken.Literal)
	}

	return spec
}

func (p *Parser) parsePrintStatement() *PrintStatement {
	stmt := &PrintStatement{Token: p.curToken}
	stmt.Values = []Expression{}
	stmt.Separators = []string{}

	p.nextToken()

	// Empty PRINT
	if p.curTokenIs(lexer.TOKEN_NEWLINE) || p.curTokenIs(lexer.TOKEN_EOF) {
		return stmt
	}

	stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.TOKEN_SEMICOLON) || p.peekTokenIs(lexer.TOKEN_COMMA) {
		sep := p.peekToken.Literal
		stmt.Separators = append(stmt.Separators, sep)
		p.nextToken()

		// Check for trailing separator (no more values)
		if p.peekTokenIs(lexer.TOKEN_NEWLINE) || p.peekTokenIs(lexer.TOKEN_EOF) {
			break
		}

		p.nextToken()
		stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
	}

	return stmt
}

func (p *Parser) parseInputStatement() *InputStatement {
	stmt := &InputStatement{Token: p.curToken}

	p.nextToken()

	// Check for optional prompt
	if p.curTokenIs(lexer.TOKEN_STRING) {
		stmt.Prompt = &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		if !p.expectPeek(lexer.TOKEN_SEMICOLON) && !p.expectPeek(lexer.TOKEN_COMMA) {
			return nil
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.TOKEN_IDENT) {
		msg := p.formatError(p.curToken.Line, p.curToken.Column,
			"expected identifier in INPUT statement",
			"INPUT requires a variable to store the user's input")
		p.errors = append(p.errors, msg)
		return nil
	}

	stmt.Variable = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return stmt
}

func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{Token: p.curToken}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_THEN) {
		return nil
	}

	p.nextToken()
	stmt.Consequence = p.parseBlockStatement(lexer.TOKEN_ENDIF, lexer.TOKEN_ELSE, lexer.TOKEN_ELSEIF)

	// Handle ELSEIF clauses
	for p.curTokenIs(lexer.TOKEN_ELSEIF) {
		elseif := &ElseIfClause{Token: p.curToken}
		p.nextToken()
		elseif.Condition = p.parseExpression(LOWEST)
		if !p.expectPeek(lexer.TOKEN_THEN) {
			return nil
		}
		p.nextToken()
		elseif.Consequence = p.parseBlockStatement(lexer.TOKEN_ENDIF, lexer.TOKEN_ELSE, lexer.TOKEN_ELSEIF)
		stmt.ElseIfs = append(stmt.ElseIfs, elseif)
	}

	// Handle ELSE
	if p.curTokenIs(lexer.TOKEN_ELSE) {
		p.nextToken()
		stmt.Alternative = p.parseBlockStatement(lexer.TOKEN_ENDIF)
	}

	return stmt
}

func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}

	stmt.Variable = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Start = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_TO) {
		return nil
	}

	p.nextToken()
	stmt.End = p.parseExpression(LOWEST)

	// Optional STEP
	if p.peekTokenIs(lexer.TOKEN_STEP) {
		p.nextToken()
		p.nextToken()
		stmt.Step = p.parseExpression(LOWEST)
	}

	p.nextToken()
	stmt.Body = p.parseBlockStatement(lexer.TOKEN_NEXT)

	return stmt
}

func (p *Parser) parseWhileStatement() *WhileStatement {
	stmt := &WhileStatement{Token: p.curToken}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	p.nextToken()
	stmt.Body = p.parseBlockStatement(lexer.TOKEN_WEND)

	return stmt
}

func (p *Parser) parseDoLoopStatement() *DoLoopStatement {
	stmt := &DoLoopStatement{Token: p.curToken}

	// Check for DO WHILE or DO UNTIL at the start
	if p.peekTokenIs(lexer.TOKEN_WHILE) {
		stmt.IsPreCondition = true
		stmt.IsWhile = true
		p.nextToken()
		p.nextToken()
		stmt.Condition = p.parseExpression(LOWEST)
	} else if p.peekTokenIs(lexer.TOKEN_UNTIL) {
		stmt.IsPreCondition = true
		stmt.IsWhile = false
		p.nextToken()
		p.nextToken()
		stmt.Condition = p.parseExpression(LOWEST)
	}

	p.nextToken()
	stmt.Body = p.parseBlockStatement(lexer.TOKEN_LOOP)

	// Check for LOOP WHILE or LOOP UNTIL
	if p.peekTokenIs(lexer.TOKEN_WHILE) {
		stmt.IsWhile = true
		p.nextToken()
		p.nextToken()
		stmt.Condition = p.parseExpression(LOWEST)
	} else if p.peekTokenIs(lexer.TOKEN_UNTIL) {
		stmt.IsWhile = false
		p.nextToken()
		p.nextToken()
		stmt.Condition = p.parseExpression(LOWEST)
	}

	return stmt
}

func (p *Parser) parseSelectStatement() *SelectStatement {
	stmt := &SelectStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_CASE) {
		return nil
	}

	p.nextToken()
	stmt.TestExpr = p.parseExpression(LOWEST)

	p.nextToken()
	p.skipNewlines()

	// Parse CASE clauses
	for p.curTokenIs(lexer.TOKEN_CASE) {
		if p.peekTokenIs(lexer.TOKEN_ELSE) {
			// CASE ELSE
			p.nextToken()
			p.nextToken()
			stmt.Default = p.parseBlockStatement(lexer.TOKEN_END, lexer.TOKEN_CASE)
		} else {
			caseClause := &CaseClause{Token: p.curToken}
			p.nextToken()

			// Parse case values
			caseClause.Values = append(caseClause.Values, p.parseExpression(LOWEST))
			for p.peekTokenIs(lexer.TOKEN_COMMA) {
				p.nextToken()
				p.nextToken()
				caseClause.Values = append(caseClause.Values, p.parseExpression(LOWEST))
			}

			p.nextToken()
			caseClause.Body = p.parseBlockStatement(lexer.TOKEN_CASE, lexer.TOKEN_END)
			stmt.Cases = append(stmt.Cases, caseClause)
		}
	}

	// Expect END SELECT
	if p.curTokenIs(lexer.TOKEN_END) {
		p.nextToken() // skip SELECT after END
	}

	return stmt
}

func (p *Parser) parseSubStatement() *SubStatement {
	stmt := &SubStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}

	stmt.Params = p.parseParameters()

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Body = p.parseBlockStatementUntilEnd("SUB")

	return stmt
}

func (p *Parser) parseFunctionStatement() *FunctionStatement {
	stmt := &FunctionStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}

	stmt.Params = p.parseParameters()

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}

	if !p.expectPeek(lexer.TOKEN_AS) {
		return nil
	}

	p.nextToken()

	// Check for multiple return types
	if p.curTokenIs(lexer.TOKEN_LPAREN) {
		p.nextToken()
		for !p.curTokenIs(lexer.TOKEN_RPAREN) {
			stmt.ReturnTypes = append(stmt.ReturnTypes, p.parseTypeSpec())
			if p.peekTokenIs(lexer.TOKEN_COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
	} else {
		stmt.ReturnTypes = append(stmt.ReturnTypes, p.parseTypeSpec())
	}

	p.nextToken()
	stmt.Body = p.parseBlockStatementUntilEnd("FUNCTION")

	return stmt
}

func (p *Parser) parseParameters() []*Parameter {
	params := []*Parameter{}

	// Check for empty parameter list
	if p.peekTokenIs(lexer.TOKEN_RPAREN) {
		return params
	}

	p.nextToken()

	for {
		param := &Parameter{}

		// Check for BYREF/BYVAL
		if p.curTokenIs(lexer.TOKEN_BYREF) {
			param.ByRef = true
			p.nextToken()
		} else if p.curTokenIs(lexer.TOKEN_BYVAL) {
			param.ByRef = false
			p.nextToken()
		}

		if !p.curTokenIs(lexer.TOKEN_IDENT) {
			msg := p.formatError(p.curToken.Line, p.curToken.Column,
				"expected parameter name",
				"parameters should be: name AS TYPE")
			p.errors = append(p.errors, msg)
			return nil
		}

		param.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

		if !p.expectPeek(lexer.TOKEN_AS) {
			return nil
		}

		p.nextToken()
		param.Type = p.parseTypeSpec()
		params = append(params, param)

		if !p.peekTokenIs(lexer.TOKEN_COMMA) {
			break
		}
		p.nextToken()
		p.nextToken()
	}

	return params
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.curToken}

	p.nextToken()

	if p.curTokenIs(lexer.TOKEN_NEWLINE) || p.curTokenIs(lexer.TOKEN_EOF) {
		return stmt
	}

	stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken()
		p.nextToken()
		stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
	}

	return stmt
}

func (p *Parser) parseExitStatement() *ExitStatement {
	stmt := &ExitStatement{Token: p.curToken}

	p.nextToken()
	stmt.ExitType = strings.ToUpper(p.curToken.Literal)

	return stmt
}

func (p *Parser) parseGotoStatement() *GotoStatement {
	stmt := &GotoStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}

	stmt.Label = p.curToken.Literal

	return stmt
}

func (p *Parser) parseLabelStatement() *LabelStatement {
	stmt := &LabelStatement{
		Token: p.curToken,
		Name:  p.curToken.Literal,
	}
	p.nextToken() // skip colon
	return stmt
}

func (p *Parser) parseSpawnStatement() *SpawnStatement {
	stmt := &SpawnStatement{Token: p.curToken}

	p.nextToken()

	expr := p.parseExpression(LOWEST)
	call, ok := expr.(*CallExpression)
	if !ok {
		msg := p.formatError(p.curToken.Line, p.curToken.Column,
			"SPAWN requires a function call",
			"use: SPAWN SubName(args)")
		p.errors = append(p.errors, msg)
		return nil
	}

	stmt.Call = call
	return stmt
}

func (p *Parser) parseSendStatement() *SendStatement {
	stmt := &SendStatement{Token: p.curToken}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_TO) {
		return nil
	}

	p.nextToken()
	stmt.Channel = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseReceiveStatement() *ReceiveStatement {
	stmt := &ReceiveStatement{Token: p.curToken}

	p.nextToken()
	stmt.Variable = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_FROM) {
		return nil
	}

	p.nextToken()
	stmt.Channel = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseAssignmentOrExpression() Statement {
	// Parse left-hand side at a higher precedence to avoid consuming '='
	// This allows person.age = 31 to be parsed correctly
	left := p.parseExpression(LESSGREATER)

	// Check for assignment
	if p.peekTokenIs(lexer.TOKEN_ASSIGN) {
		p.nextToken()
		tok := p.curToken
		p.nextToken()
		value := p.parseExpression(LOWEST)
		return &AssignmentStatement{Token: tok, Left: left, Value: value}
	}

	// If no assignment, we need to continue parsing any comparison/logical operators
	// that may follow the left-hand side
	for p.peekPrecedence() > LOWEST {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			break
		}
		p.nextToken()
		left = infix(left)
	}

	// Check for multiple assignment: a, b = func()
	if p.peekTokenIs(lexer.TOKEN_COMMA) {
		targets := []Expression{left}
		for p.peekTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next target

			// Parse just the identifier, not a full expression
			// to avoid consuming the = sign
			if p.curTokenIs(lexer.TOKEN_IDENT) {
				targets = append(targets, &Identifier{Token: p.curToken, Value: p.curToken.Literal})
			} else {
				msg := p.formatError(p.curToken.Line, p.curToken.Column,
					"expected identifier in multiple assignment",
					"use: a, b = FunctionCall()")
				p.errors = append(p.errors, msg)
				return nil
			}
		}
		if p.peekTokenIs(lexer.TOKEN_ASSIGN) {
			p.nextToken()
			tok := p.curToken
			p.nextToken()
			value := p.parseExpression(LOWEST)
			return &MultiAssignmentStatement{Token: tok, Targets: targets, Value: value}
		}
	}

	return &ExpressionStatement{Token: p.curToken, Expression: left}
}

func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	stmt := &ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)
	return stmt
}

func (p *Parser) parseBlockStatement(endTokens ...lexer.TokenType) *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}

	p.skipNewlines()

	for !p.isEndToken(endTokens...) && !p.curTokenIs(lexer.TOKEN_EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
		p.skipNewlines()
	}

	return block
}

func (p *Parser) parseBlockStatementUntilEnd(blockType string) *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}

	p.skipNewlines()

	for !p.curTokenIs(lexer.TOKEN_EOF) {
		if p.curTokenIs(lexer.TOKEN_END) {
			// Check if it's END SUB or END FUNCTION
			if p.peekTokenIs(lexer.TOKEN_SUB) || p.peekTokenIs(lexer.TOKEN_FUNCTION) {
				p.nextToken() // consume SUB or FUNCTION
				break
			}
		}
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
		p.skipNewlines()
	}

	return block
}

func (p *Parser) isEndToken(tokens ...lexer.TokenType) bool {
	for _, t := range tokens {
		if p.curTokenIs(t) {
			return true
		}
	}
	return false
}

// Expression parsing

func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		msg := p.formatError(p.curToken.Line, p.curToken.Column,
			fmt.Sprintf("unexpected token: %s", p.curToken.Type),
			"expected an expression (variable, literal, or function call)")
		p.errors = append(p.errors, msg)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(lexer.TOKEN_NEWLINE) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() Expression {
	lit := &IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := p.formatError(p.curToken.Line, p.curToken.Column,
			fmt.Sprintf("could not parse %q as integer", p.curToken.Literal),
			"integer values should be whole numbers like 42 or -17")
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() Expression {
	lit := &FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := p.formatError(p.curToken.Line, p.curToken.Column,
			fmt.Sprintf("could not parse %q as float", p.curToken.Literal),
			"float values should be like 3.14 or 2.5e10")
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() Expression {
	return &BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(lexer.TOKEN_TRUE)}
}

func (p *Parser) parseNilLiteral() Expression {
	return &NilLiteral{Token: p.curToken}
}

func (p *Parser) parseJSONLiteral() Expression {
	lit := &JSONLiteral{Token: p.curToken, Pairs: make(map[string]Expression)}

	p.nextToken()

	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		// Parse key
		if !p.curTokenIs(lexer.TOKEN_STRING) && !p.curTokenIs(lexer.TOKEN_IDENT) {
			msg := p.formatError(p.curToken.Line, p.curToken.Column,
				"expected string key in JSON object",
				"JSON keys should be strings like \"name\" or identifiers")
			p.errors = append(p.errors, msg)
			return nil
		}
		key := p.curToken.Literal

		if !p.expectPeek(lexer.TOKEN_COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)
		lit.Pairs[key] = value

		if p.peekTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
		}
		p.nextToken()
	}

	return lit
}

func (p *Parser) parseArrayLiteral() Expression {
	lit := &ArrayLiteral{Token: p.curToken}

	lit.Elements = p.parseExpressionList(lexer.TOKEN_RBRACKET)

	return lit
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []Expression {
	list := []Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parsePrefixExpression() Expression {
	expression := &PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseAddressOf() Expression {
	expr := &AddressOfExpression{Token: p.curToken}
	p.nextToken()
	expr.Value = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseDereference() Expression {
	expr := &DereferenceExpression{Token: p.curToken}
	p.nextToken()
	expr.Value = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseMakeChan() Expression {
	expr := &MakeChanExpression{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}

	p.nextToken()
	expr.ChannelType = p.parseTypeSpec()

	if p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken()
		p.nextToken()
		expr.Size = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}

	return expr
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expression := &InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseCallExpression(function Expression) Expression {
	exp := &CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(lexer.TOKEN_RPAREN)
	return exp
}

func (p *Parser) parseIndexExpression(left Expression) Expression {
	exp := &IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseMemberExpression(left Expression) Expression {
	exp := &MemberExpression{Token: p.curToken, Object: left}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}

	exp.Member = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}
