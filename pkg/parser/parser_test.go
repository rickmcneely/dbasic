package parser

import (
	"testing"

	"github.com/zditech/dbasic/pkg/lexer"
)

func TestParseDimStatement(t *testing.T) {
	tests := []struct {
		input        string
		expectedName string
		expectedType string
	}{
		{"DIM x AS INTEGER", "x", "INTEGER"},
		{"DIM name AS STRING", "name", "STRING"},
		{"DIM flag AS BOOLEAN", "flag", "BOOLEAN"},
		{"DIM data AS JSON", "data", "JSON"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*DimStatement)
		if !ok {
			t.Fatalf("expected DimStatement, got %T", program.Statements[0])
		}

		if stmt.Name.Value != tt.expectedName {
			t.Errorf("expected name %s, got %s", tt.expectedName, stmt.Name.Value)
		}

		if stmt.Type.Name != tt.expectedType {
			t.Errorf("expected type %s, got %s", tt.expectedType, stmt.Type.Name)
		}
	}
}

func TestParseDimWithValue(t *testing.T) {
	input := `DIM x AS INTEGER = 42`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*DimStatement)
	if !ok {
		t.Fatalf("expected DimStatement, got %T", program.Statements[0])
	}

	if stmt.Value == nil {
		t.Fatal("expected initial value, got nil")
	}

	intLit, ok := stmt.Value.(*IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", stmt.Value)
	}

	if intLit.Value != 42 {
		t.Errorf("expected value 42, got %d", intLit.Value)
	}
}

func TestParseLetStatement(t *testing.T) {
	input := `LET x = 42`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*LetStatement)
	if !ok {
		t.Fatalf("expected LetStatement, got %T", program.Statements[0])
	}

	if stmt.Name.Value != "x" {
		t.Errorf("expected name 'x', got %s", stmt.Name.Value)
	}

	intLit, ok := stmt.Value.(*IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", stmt.Value)
	}

	if intLit.Value != 42 {
		t.Errorf("expected value 42, got %d", intLit.Value)
	}
}

func TestParseIfStatement(t *testing.T) {
	input := `IF x > 10 THEN
    PRINT "big"
ENDIF`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("expected IfStatement, got %T", program.Statements[0])
	}

	if stmt.Condition == nil {
		t.Fatal("expected condition, got nil")
	}

	if stmt.Consequence == nil {
		t.Fatal("expected consequence, got nil")
	}

	if len(stmt.Consequence.Statements) != 1 {
		t.Errorf("expected 1 statement in consequence, got %d", len(stmt.Consequence.Statements))
	}
}

func TestParseIfElseStatement(t *testing.T) {
	input := `IF x > 10 THEN
    PRINT "big"
ELSE
    PRINT "small"
ENDIF`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("expected IfStatement, got %T", program.Statements[0])
	}

	if stmt.Alternative == nil {
		t.Fatal("expected alternative, got nil")
	}
}

func TestParseIfElseIfStatement(t *testing.T) {
	input := `IF x > 10 THEN
    PRINT "big"
ELSEIF x > 5 THEN
    PRINT "medium"
ELSE
    PRINT "small"
ENDIF`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("expected IfStatement, got %T", program.Statements[0])
	}

	if len(stmt.ElseIfs) != 1 {
		t.Errorf("expected 1 ELSEIF clause, got %d", len(stmt.ElseIfs))
	}
}

func TestParseForStatement(t *testing.T) {
	input := `FOR i = 1 TO 10
    PRINT i
NEXT`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ForStatement)
	if !ok {
		t.Fatalf("expected ForStatement, got %T", program.Statements[0])
	}

	if stmt.Variable.Value != "i" {
		t.Errorf("expected variable 'i', got %s", stmt.Variable.Value)
	}

	if stmt.Step != nil {
		t.Error("expected nil step, got value")
	}
}

func TestParseForWithStep(t *testing.T) {
	input := `FOR i = 10 TO 0 STEP -1
    PRINT i
NEXT`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ForStatement)
	if stmt.Step == nil {
		t.Fatal("expected step, got nil")
	}
}

func TestParseWhileStatement(t *testing.T) {
	input := `WHILE x > 0
    x = x - 1
WEND`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*WhileStatement)
	if !ok {
		t.Fatalf("expected WhileStatement, got %T", program.Statements[0])
	}

	if stmt.Condition == nil {
		t.Fatal("expected condition")
	}
}

func TestParseDoLoopStatement(t *testing.T) {
	tests := []struct {
		input          string
		isPreCondition bool
		isWhile        bool
	}{
		{"DO WHILE x > 0\n    x = x - 1\nLOOP", true, true},
		{"DO UNTIL x = 0\n    x = x - 1\nLOOP", true, false},
		{"DO\n    x = x - 1\nLOOP WHILE x > 0", false, true},
		{"DO\n    x = x - 1\nLOOP UNTIL x = 0", false, false},
	}

	for i, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt, ok := program.Statements[0].(*DoLoopStatement)
		if !ok {
			t.Fatalf("test[%d]: expected DoLoopStatement, got %T", i, program.Statements[0])
		}

		if stmt.IsPreCondition != tt.isPreCondition {
			t.Errorf("test[%d]: expected IsPreCondition=%v, got %v", i, tt.isPreCondition, stmt.IsPreCondition)
		}

		if stmt.IsWhile != tt.isWhile {
			t.Errorf("test[%d]: expected IsWhile=%v, got %v", i, tt.isWhile, stmt.IsWhile)
		}
	}
}

func TestParseSubStatement(t *testing.T) {
	input := `SUB MyProc()
    PRINT "Hello"
END SUB`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*SubStatement)
	if !ok {
		t.Fatalf("expected SubStatement, got %T", program.Statements[0])
	}

	if stmt.Name.Value != "MyProc" {
		t.Errorf("expected name 'MyProc', got %s", stmt.Name.Value)
	}

	if len(stmt.Params) != 0 {
		t.Errorf("expected 0 params, got %d", len(stmt.Params))
	}
}

func TestParseSubWithParams(t *testing.T) {
	input := `SUB Greet(name AS STRING, times AS INTEGER)
    FOR i = 1 TO times
        PRINT "Hello, "; name
    NEXT
END SUB`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*SubStatement)
	if len(stmt.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(stmt.Params))
	}

	if stmt.Params[0].Name.Value != "name" {
		t.Errorf("expected param name 'name', got %s", stmt.Params[0].Name.Value)
	}

	if stmt.Params[1].Type.Name != "INTEGER" {
		t.Errorf("expected param type INTEGER, got %s", stmt.Params[1].Type.Name)
	}
}

func TestParseFunctionStatement(t *testing.T) {
	input := `FUNCTION Add(a AS INTEGER, b AS INTEGER) AS INTEGER
    RETURN a + b
END FUNCTION`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*FunctionStatement)
	if !ok {
		t.Fatalf("expected FunctionStatement, got %T", program.Statements[0])
	}

	if stmt.Name.Value != "Add" {
		t.Errorf("expected name 'Add', got %s", stmt.Name.Value)
	}

	if len(stmt.ReturnTypes) != 1 {
		t.Fatalf("expected 1 return type, got %d", len(stmt.ReturnTypes))
	}

	if stmt.ReturnTypes[0].Name != "INTEGER" {
		t.Errorf("expected return type INTEGER, got %s", stmt.ReturnTypes[0].Name)
	}
}

func TestParseFunctionMultipleReturns(t *testing.T) {
	input := `FUNCTION Divide(a AS INTEGER, b AS INTEGER) AS (INTEGER, BOOLEAN)
    IF b = 0 THEN
        RETURN 0, FALSE
    ENDIF
    RETURN a / b, TRUE
END FUNCTION`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*FunctionStatement)
	if len(stmt.ReturnTypes) != 2 {
		t.Fatalf("expected 2 return types, got %d", len(stmt.ReturnTypes))
	}
}

func TestParseImportStatement(t *testing.T) {
	input := `IMPORT "fmt"`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ImportStatement)
	if !ok {
		t.Fatalf("expected ImportStatement, got %T", program.Statements[0])
	}

	if stmt.Package != "fmt" {
		t.Errorf("expected package 'fmt', got %s", stmt.Package)
	}
}

func TestParseImportWithAlias(t *testing.T) {
	input := `IMPORT "net/http" AS http`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ImportStatement)
	if stmt.Alias != "http" {
		t.Errorf("expected alias 'http', got %s", stmt.Alias)
	}
}

func TestParsePrintStatement(t *testing.T) {
	tests := []struct {
		input         string
		valueCount    int
		separatorCount int
	}{
		{`PRINT "Hello"`, 1, 0},
		{`PRINT "Hello"; " World"`, 2, 1},
		{`PRINT a, b, c`, 3, 2},
		{`PRINT "Value:"; x;`, 2, 2},
	}

	for i, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt, ok := program.Statements[0].(*PrintStatement)
		if !ok {
			t.Fatalf("test[%d]: expected PrintStatement, got %T", i, program.Statements[0])
		}

		if len(stmt.Values) != tt.valueCount {
			t.Errorf("test[%d]: expected %d values, got %d", i, tt.valueCount, len(stmt.Values))
		}

		if len(stmt.Separators) != tt.separatorCount {
			t.Errorf("test[%d]: expected %d separators, got %d", i, tt.separatorCount, len(stmt.Separators))
		}
	}
}

func TestParseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5 + 3", "(5 + 3)"},
		{"5 - 3 * 2", "(5 - (3 * 2))"},
		{"(5 - 3) * 2", "((5 - 3) * 2)"},
		{"2 ^ 3", "(2 ^ 3)"},
		{"a AND b OR c", "((a AND b) OR c)"},
		{"NOT flag", "(NOTflag)"},
		{"-5", "(-5)"},
	}

	for i, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("test[%d]: expected 1 statement, got %d", i, len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ExpressionStatement)
		if !ok {
			t.Fatalf("test[%d]: expected ExpressionStatement, got %T", i, program.Statements[0])
		}

		if stmt.Expression.String() != tt.expected {
			t.Errorf("test[%d]: expected %s, got %s", i, tt.expected, stmt.Expression.String())
		}
	}
}

func TestParseCallExpression(t *testing.T) {
	input := `MyFunc(1, 2, "hello")`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ExpressionStatement)
	call, ok := stmt.Expression.(*CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", stmt.Expression)
	}

	if len(call.Arguments) != 3 {
		t.Errorf("expected 3 arguments, got %d", len(call.Arguments))
	}
}

func TestParseIndexExpression(t *testing.T) {
	input := `arr[0]`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ExpressionStatement)
	idx, ok := stmt.Expression.(*IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", stmt.Expression)
	}

	ident, ok := idx.Left.(*Identifier)
	if !ok {
		t.Fatalf("expected Identifier, got %T", idx.Left)
	}

	if ident.Value != "arr" {
		t.Errorf("expected 'arr', got %s", ident.Value)
	}
}

func TestParseMemberExpression(t *testing.T) {
	input := `obj.property`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ExpressionStatement)
	member, ok := stmt.Expression.(*MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", stmt.Expression)
	}

	if member.Member.Value != "property" {
		t.Errorf("expected 'property', got %s", member.Member.Value)
	}
}

func TestParseJSONLiteral(t *testing.T) {
	input := `{"name": "John", "age": 30}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ExpressionStatement)
	json, ok := stmt.Expression.(*JSONLiteral)
	if !ok {
		t.Fatalf("expected JSONLiteral, got %T", stmt.Expression)
	}

	if len(json.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(json.Pairs))
	}
}

func TestParseArrayLiteral(t *testing.T) {
	input := `[1, 2, 3, 4, 5]`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ExpressionStatement)
	arr, ok := stmt.Expression.(*ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral, got %T", stmt.Expression)
	}

	if len(arr.Elements) != 5 {
		t.Errorf("expected 5 elements, got %d", len(arr.Elements))
	}
}

func TestParsePointerOperations(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@x", "@x"},
		{"^ptr", "^ptr"},
	}

	for i, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ExpressionStatement)
		if stmt.Expression.String() != tt.expected {
			t.Errorf("test[%d]: expected %s, got %s", i, tt.expected, stmt.Expression.String())
		}
	}
}

func TestParseSpawnStatement(t *testing.T) {
	input := `SPAWN Worker(1, "task")`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*SpawnStatement)
	if !ok {
		t.Fatalf("expected SpawnStatement, got %T", program.Statements[0])
	}

	if stmt.Call == nil {
		t.Fatal("expected call expression")
	}
}

func TestParseSendReceive(t *testing.T) {
	input := `SEND 42 TO ch
RECEIVE x FROM ch`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}

	send, ok := program.Statements[0].(*SendStatement)
	if !ok {
		t.Fatalf("expected SendStatement, got %T", program.Statements[0])
	}
	if send.Value == nil || send.Channel == nil {
		t.Error("expected value and channel in SEND")
	}

	recv, ok := program.Statements[1].(*ReceiveStatement)
	if !ok {
		t.Fatalf("expected ReceiveStatement, got %T", program.Statements[1])
	}
	if recv.Variable == nil || recv.Channel == nil {
		t.Error("expected variable and channel in RECEIVE")
	}
}

func TestParseLabelAndGoto(t *testing.T) {
	input := `start:
    PRINT "Hello"
    GOTO start`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	label, ok := program.Statements[0].(*LabelStatement)
	if !ok {
		t.Fatalf("expected LabelStatement, got %T", program.Statements[0])
	}
	if label.Name != "start" {
		t.Errorf("expected label 'start', got %s", label.Name)
	}

	gotoStmt, ok := program.Statements[2].(*GotoStatement)
	if !ok {
		t.Fatalf("expected GotoStatement, got %T", program.Statements[2])
	}
	if gotoStmt.Label != "start" {
		t.Errorf("expected goto 'start', got %s", gotoStmt.Label)
	}
}

func TestParseSelectCase(t *testing.T) {
	input := `SELECT CASE x
CASE 1
    PRINT "one"
CASE 2, 3
    PRINT "two or three"
CASE ELSE
    PRINT "other"
END SELECT`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*SelectStatement)
	if !ok {
		t.Fatalf("expected SelectStatement, got %T", program.Statements[0])
	}

	if len(stmt.Cases) != 2 {
		t.Errorf("expected 2 cases, got %d", len(stmt.Cases))
	}

	if stmt.Default == nil {
		t.Error("expected default case")
	}

	// Check that case 2 has two values
	if len(stmt.Cases[1].Values) != 2 {
		t.Errorf("expected 2 values in case, got %d", len(stmt.Cases[1].Values))
	}
}

func TestParseMultiAssignment(t *testing.T) {
	input := `result, ok = Divide(10, 2)`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*MultiAssignmentStatement)
	if !ok {
		t.Fatalf("expected MultiAssignmentStatement, got %T", program.Statements[0])
	}

	if len(stmt.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(stmt.Targets))
	}
}

func TestParserErrors(t *testing.T) {
	tests := []struct {
		input       string
		errorSubstr string
	}{
		{"IF x > 5", "expected THEN"},
		{"DIM x INTEGER", "expected AS"},
		{"FUNCTION foo(", "expected"},
	}

	for i, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		p.ParseProgram()

		if len(p.Errors()) == 0 {
			t.Errorf("test[%d]: expected errors, got none", i)
			continue
		}
	}
}

func TestCompleteProgram(t *testing.T) {
	input := `' A complete DBasic program
IMPORT "fmt"

DIM counter AS INTEGER = 0

FUNCTION Increment(n AS INTEGER) AS INTEGER
    RETURN n + 1
END FUNCTION

SUB Main()
    FOR i = 1 TO 5
        counter = Increment(counter)
        PRINT counter
    NEXT
END SUB`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	// Should have: IMPORT, DIM, FUNCTION, SUB
	if len(program.Statements) < 4 {
		t.Errorf("expected at least 4 statements, got %d", len(program.Statements))
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors:", len(errors))
	for _, msg := range errors {
		t.Errorf("  parser error: %s", msg)
	}
	t.FailNow()
}
