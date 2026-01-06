package analyzer

import (
	"strings"
	"testing"

	"github.com/zditech/dbasic/pkg/lexer"
	"github.com/zditech/dbasic/pkg/parser"
)

func parse(input string) *parser.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func TestAnalyzeVariableDeclaration(t *testing.T) {
	input := `DIM x AS INTEGER
DIM name AS STRING
DIM flag AS BOOLEAN`

	program := parse(input)
	a := New()
	symbols, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	// Check that variables are defined
	if sym := symbols.Resolve("x"); sym == nil {
		t.Error("expected 'x' to be defined")
	} else if sym.Type.Kind != TypeInteger {
		t.Errorf("expected 'x' to be INTEGER, got %s", sym.Type.String())
	}

	if sym := symbols.Resolve("name"); sym == nil {
		t.Error("expected 'name' to be defined")
	}

	if sym := symbols.Resolve("flag"); sym == nil {
		t.Error("expected 'flag' to be defined")
	}
}

func TestAnalyzeLetStatement(t *testing.T) {
	input := `LET x = 42
LET name = "hello"
LET flag = TRUE`

	program := parse(input)
	a := New()
	symbols, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	// Check type inference
	if sym := symbols.Resolve("x"); sym == nil {
		t.Error("expected 'x' to be defined")
	} else if !sym.Type.IsInteger() {
		t.Errorf("expected 'x' to be integer type, got %s", sym.Type.String())
	}

	if sym := symbols.Resolve("name"); sym == nil {
		t.Error("expected 'name' to be defined")
	} else if sym.Type.Kind != TypeString {
		t.Errorf("expected 'name' to be STRING, got %s", sym.Type.String())
	}

	if sym := symbols.Resolve("flag"); sym == nil {
		t.Error("expected 'flag' to be defined")
	} else if sym.Type.Kind != TypeBoolean {
		t.Errorf("expected 'flag' to be BOOLEAN, got %s", sym.Type.String())
	}
}

func TestAnalyzeDuplicateVariable(t *testing.T) {
	input := `DIM x AS INTEGER
DIM x AS STRING`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) == 0 {
		t.Error("expected duplicate variable error")
	}

	found := false
	for _, e := range errors {
		if strings.Contains(e, "already defined") || strings.Contains(e, "duplicate") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected duplicate definition error, got: %v", errors)
	}
}

func TestAnalyzeUndefinedVariable(t *testing.T) {
	input := `SUB Main()
    PRINT undefinedVar
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) == 0 {
		t.Error("expected undefined variable error")
	}

	found := false
	for _, e := range errors {
		if strings.Contains(e, "undefined") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected undefined error, got: %v", errors)
	}
}

func TestAnalyzeTypeMismatch(t *testing.T) {
	input := `DIM x AS INTEGER = "hello"`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) == 0 {
		t.Error("expected type mismatch error")
	}

	found := false
	for _, e := range errors {
		if strings.Contains(e, "type mismatch") || strings.Contains(e, "cannot assign") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected type mismatch error, got: %v", errors)
	}
}

func TestAnalyzeFunctionDeclaration(t *testing.T) {
	input := `FUNCTION Add(a AS INTEGER, b AS INTEGER) AS INTEGER
    RETURN a + b
END FUNCTION`

	program := parse(input)
	a := New()
	symbols, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	sym := symbols.Resolve("ADD")
	if sym == nil {
		t.Fatal("expected 'Add' function to be defined")
	}

	if sym.Kind != SymFunction {
		t.Errorf("expected function, got %v", sym.Kind)
	}

	if len(sym.Type.ParamTypes) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(sym.Type.ParamTypes))
	}

	if len(sym.Type.ReturnTypes) != 1 {
		t.Errorf("expected 1 return type, got %d", len(sym.Type.ReturnTypes))
	}
}

func TestAnalyzeSubDeclaration(t *testing.T) {
	input := `SUB PrintMessage(msg AS STRING)
    PRINT msg
END SUB`

	program := parse(input)
	a := New()
	symbols, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	sym := symbols.Resolve("PRINTMESSAGE")
	if sym == nil {
		t.Fatal("expected 'PrintMessage' sub to be defined")
	}

	if sym.Kind != SymSub {
		t.Errorf("expected sub, got %v", sym.Kind)
	}
}

func TestAnalyzeFunctionCall(t *testing.T) {
	input := `FUNCTION Add(a AS INTEGER, b AS INTEGER) AS INTEGER
    RETURN a + b
END FUNCTION

SUB Main()
    DIM result AS INTEGER
    result = Add(1, 2)
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeWrongArgumentCount(t *testing.T) {
	input := `FUNCTION Add(a AS INTEGER, b AS INTEGER) AS INTEGER
    RETURN a + b
END FUNCTION

SUB Main()
    DIM result AS INTEGER
    result = Add(1)
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) == 0 {
		t.Error("expected wrong argument count error")
	}
}

func TestAnalyzeIfCondition(t *testing.T) {
	input := `SUB Main()
    DIM x AS INTEGER = 5
    IF x THEN
        PRINT "yes"
    ENDIF
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) == 0 {
		t.Error("expected condition must be boolean error")
	}
}

func TestAnalyzeForLoop(t *testing.T) {
	input := `SUB Main()
    FOR i = 1 TO 10
        PRINT i
    NEXT
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeForLoopScope(t *testing.T) {
	input := `SUB Main()
    FOR i = 1 TO 10
        PRINT i
    NEXT
    ' i should not be accessible here, but DBasic keeps it in scope
    PRINT i
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	// FOR loop variable should be accessible after the loop in BASIC
	// This is different from languages like C
	// Note: Current implementation may or may not allow this
	_ = errors
}

func TestAnalyzeWhileLoop(t *testing.T) {
	input := `SUB Main()
    DIM x AS INTEGER = 10
    WHILE x > 0
        x = x - 1
    WEND
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeChannelOperations(t *testing.T) {
	input := `DIM ch AS CHAN OF INTEGER
SEND 42 TO ch
DIM x AS INTEGER
RECEIVE x FROM ch`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeChannelTypeMismatch(t *testing.T) {
	input := `DIM ch AS CHAN OF INTEGER
SEND "hello" TO ch`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) == 0 {
		t.Error("expected channel type mismatch error")
	}
}

func TestAnalyzePointerType(t *testing.T) {
	input := `DIM x AS INTEGER = 42
DIM ptr AS POINTER TO INTEGER`

	program := parse(input)
	a := New()
	symbols, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	sym := symbols.Resolve("ptr")
	if sym == nil {
		t.Fatal("expected 'ptr' to be defined")
	}

	if sym.Type.Kind != TypePointer {
		t.Errorf("expected pointer type, got %s", sym.Type.String())
	}
}

func TestAnalyzeImport(t *testing.T) {
	input := `IMPORT "fmt"
IMPORT "net/http" AS http`

	program := parse(input)
	a := New()
	symbols, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	// Check that imports are registered
	if symbols.GetImport("fmt") == nil {
		t.Error("expected 'fmt' import to be registered")
	}

	if symbols.GetImport("http") == nil {
		t.Error("expected 'http' import alias to be registered")
	}
}

func TestAnalyzeMultipleReturnValues(t *testing.T) {
	input := `FUNCTION Divide(a AS INTEGER, b AS INTEGER) AS (INTEGER, BOOLEAN)
    IF b = 0 THEN
        RETURN 0, FALSE
    ENDIF
    RETURN a / b, TRUE
END FUNCTION

SUB Main()
    DIM result AS INTEGER
    DIM ok AS BOOLEAN
    result, ok = Divide(10, 2)
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeArithmeticOperators(t *testing.T) {
	input := `SUB Main()
    DIM a AS INTEGER = 5 + 3
    DIM b AS INTEGER = 10 - 2
    DIM c AS INTEGER = 4 * 2
    DIM d AS INTEGER = 8 / 2
    DIM e AS INTEGER = 10 MOD 3
    DIM f AS DOUBLE = 2 ^ 3
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeStringConcatenation(t *testing.T) {
	input := `SUB Main()
    DIM s AS STRING = "Hello" & " " & "World"
    DIM t AS STRING = "A" + "B"
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeLogicalOperators(t *testing.T) {
	input := `SUB Main()
    DIM a AS BOOLEAN = TRUE AND FALSE
    DIM b AS BOOLEAN = TRUE OR FALSE
    DIM c AS BOOLEAN = NOT TRUE
    DIM d AS BOOLEAN = TRUE XOR FALSE
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeComparisonOperators(t *testing.T) {
	input := `SUB Main()
    DIM a AS BOOLEAN = 5 > 3
    DIM b AS BOOLEAN = 5 < 10
    DIM c AS BOOLEAN = 5 >= 5
    DIM d AS BOOLEAN = 5 <= 5
    DIM e AS BOOLEAN = 5 = 5
    DIM f AS BOOLEAN = 5 <> 3
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeArrayType(t *testing.T) {
	input := `DIM arr(10) AS INTEGER`

	program := parse(input)
	a := New()
	symbols, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	sym := symbols.Resolve("arr")
	if sym == nil {
		t.Fatal("expected 'arr' to be defined")
	}
}

func TestAnalyzeJSONType(t *testing.T) {
	input := `DIM data AS JSON = {"name": "John", "age": 30}

SUB Main()
    PRINT data.name
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestHasMain(t *testing.T) {
	tests := []struct {
		input   string
		hasMain bool
	}{
		{`SUB Main()
END SUB`, true},
		{`SUB Other()
END SUB`, false},
		{`DIM x AS INTEGER`, false},
	}

	for i, tt := range tests {
		program := parse(tt.input)
		a := New()
		a.Analyze(program)

		if a.HasMain() != tt.hasMain {
			t.Errorf("test[%d]: expected HasMain()=%v, got %v", i, tt.hasMain, a.HasMain())
		}
	}
}

func TestAnalyzeLabel(t *testing.T) {
	input := `SUB Main()
start:
    PRINT "Hello"
    GOTO start
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestAnalyzeSelectCase(t *testing.T) {
	input := `SUB Main()
    DIM x AS INTEGER = 2
    SELECT CASE x
    CASE 1
        PRINT "one"
    CASE 2
        PRINT "two"
    CASE ELSE
        PRINT "other"
    END SELECT
END SUB`

	program := parse(input)
	a := New()
	_, errors := a.Analyze(program)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}
}
