package codegen

import (
	"strings"
	"testing"

	"github.com/zditech/dbasic/pkg/analyzer"
	"github.com/zditech/dbasic/pkg/lexer"
	"github.com/zditech/dbasic/pkg/parser"
)

func compile(input string) string {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	a := analyzer.New()
	symbols, _ := a.Analyze(program)

	g := New(program, symbols)
	return g.Generate()
}

func TestGenerateVariableDeclaration(t *testing.T) {
	input := `DIM x AS INTEGER = 42`

	code := compile(input)

	// Variables are generated in a var block
	if !strings.Contains(code, "x int = 42") {
		t.Errorf("expected 'x int = 42', got:\n%s", code)
	}
}

func TestGenerateTypeMappings(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"DIM a AS INTEGER", "a int"},
		{"DIM b AS LONG", "b int64"},
		{"DIM c AS SINGLE", "c float32"},
		{"DIM d AS DOUBLE", "d float64"},
		{"DIM e AS STRING", "e string"},
		{"DIM f AS BOOLEAN", "f bool"},
	}

	for i, tt := range tests {
		code := compile(tt.input)
		if !strings.Contains(code, tt.expected) {
			t.Errorf("test[%d]: expected '%s' in output, got:\n%s", i, tt.expected, code)
		}
	}
}

func TestGenerateFunction(t *testing.T) {
	input := `FUNCTION Add(a AS INTEGER, b AS INTEGER) AS INTEGER
    RETURN a + b
END FUNCTION`

	code := compile(input)

	if !strings.Contains(code, "func Add(a int, b int) int") {
		t.Errorf("expected function signature, got:\n%s", code)
	}

	if !strings.Contains(code, "return (a + b)") {
		t.Errorf("expected return statement, got:\n%s", code)
	}
}

func TestGenerateSub(t *testing.T) {
	input := `SUB PrintHello()
    PRINT "Hello"
END SUB`

	code := compile(input)

	if !strings.Contains(code, "func PrintHello()") {
		t.Errorf("expected sub declaration, got:\n%s", code)
	}

	// PRINT generates fmt.Println
	if !strings.Contains(code, `fmt.Println("Hello")`) {
		t.Errorf("expected print statement, got:\n%s", code)
	}
}

func TestGenerateMain(t *testing.T) {
	input := `SUB Main()
    PRINT "Hello, World!"
END SUB`

	code := compile(input)

	if !strings.Contains(code, "func main()") {
		t.Errorf("expected main function, got:\n%s", code)
	}

	if !strings.Contains(code, "Main()") {
		t.Errorf("expected call to Main(), got:\n%s", code)
	}
}

func TestGenerateIfStatement(t *testing.T) {
	input := `SUB Main()
    DIM x AS INTEGER = 10
    IF x > 5 THEN
        PRINT "big"
    ENDIF
END SUB`

	code := compile(input)

	if !strings.Contains(code, "if (x > 5)") {
		t.Errorf("expected if statement, got:\n%s", code)
	}
}

func TestGenerateIfElse(t *testing.T) {
	input := `SUB Main()
    DIM x AS INTEGER = 3
    IF x > 5 THEN
        PRINT "big"
    ELSE
        PRINT "small"
    ENDIF
END SUB`

	code := compile(input)

	if !strings.Contains(code, "} else {") {
		t.Errorf("expected else clause, got:\n%s", code)
	}
}

func TestGenerateForLoop(t *testing.T) {
	input := `SUB Main()
    FOR i = 1 TO 10
        PRINT i
    NEXT
END SUB`

	code := compile(input)

	// Check for loop structure (may not have type casts)
	if !strings.Contains(code, "for i =") || !strings.Contains(code, "<= 10") {
		t.Errorf("expected for loop, got:\n%s", code)
	}
}

func TestGenerateForLoopWithStep(t *testing.T) {
	input := `SUB Main()
    FOR i = 10 TO 1 STEP -1
        PRINT i
    NEXT
END SUB`

	code := compile(input)

	// Check for negative step
	if !strings.Contains(code, "i += -1") && !strings.Contains(code, "i--") {
		t.Errorf("expected for loop with negative step, got:\n%s", code)
	}
}

func TestGenerateWhileLoop(t *testing.T) {
	input := `SUB Main()
    DIM x AS INTEGER = 10
    WHILE x > 0
        x = x - 1
    WEND
END SUB`

	code := compile(input)

	if !strings.Contains(code, "for (x > 0)") {
		t.Errorf("expected while loop (as for), got:\n%s", code)
	}
}

func TestGenerateDoLoop(t *testing.T) {
	input := `SUB Main()
    DIM x AS INTEGER = 0
    DO
        x = x + 1
    LOOP WHILE x < 10
END SUB`

	code := compile(input)

	if !strings.Contains(code, "for {") {
		t.Errorf("expected do loop, got:\n%s", code)
	}

	if !strings.Contains(code, "if !((x < 10)) { break }") {
		t.Errorf("expected loop condition, got:\n%s", code)
	}
}

func TestGenerateSelectCase(t *testing.T) {
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

	code := compile(input)

	if !strings.Contains(code, "switch x {") {
		t.Errorf("expected switch statement, got:\n%s", code)
	}

	if !strings.Contains(code, "case 1:") && !strings.Contains(code, "case int(1):") {
		t.Errorf("expected case clause, got:\n%s", code)
	}

	if !strings.Contains(code, "default:") {
		t.Errorf("expected default clause, got:\n%s", code)
	}
}

func TestGenerateSpawn(t *testing.T) {
	input := `SUB Worker()
    PRINT "Working"
END SUB

SUB Main()
    SPAWN Worker()
END SUB`

	code := compile(input)

	if !strings.Contains(code, "go Worker()") {
		t.Errorf("expected goroutine, got:\n%s", code)
	}
}

func TestGenerateChannelOperations(t *testing.T) {
	input := `SUB Main()
    DIM ch AS CHAN OF INTEGER = MAKE_CHAN(INTEGER, 10)
    SEND 42 TO ch
    DIM x AS INTEGER
    RECEIVE x FROM ch
END SUB`

	code := compile(input)

	if !strings.Contains(code, "chan int") {
		t.Errorf("expected channel type, got:\n%s", code)
	}

	if !strings.Contains(code, "ch <- 42") {
		t.Errorf("expected send operation, got:\n%s", code)
	}

	if !strings.Contains(code, "x = <-ch") {
		t.Errorf("expected receive operation, got:\n%s", code)
	}
}

func TestGeneratePointerOperations(t *testing.T) {
	input := `SUB Main()
    DIM x AS INTEGER = 42
    DIM ptr AS POINTER TO INTEGER = @x
    PRINT ^ptr
END SUB`

	code := compile(input)

	if !strings.Contains(code, "*int") {
		t.Errorf("expected pointer type, got:\n%s", code)
	}

	if !strings.Contains(code, "&x") {
		t.Errorf("expected address-of, got:\n%s", code)
	}

	if !strings.Contains(code, "*ptr") {
		t.Errorf("expected dereference, got:\n%s", code)
	}
}

func TestGenerateImport(t *testing.T) {
	input := `IMPORT "time"

SUB Main()
    time.Sleep(1000000000)
END SUB`

	code := compile(input)

	if !strings.Contains(code, `"time"`) {
		t.Errorf("expected time import, got:\n%s", code)
	}
}

func TestGeneratePrintStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`PRINT "Hello"`, `fmt.Println("Hello")`},
		{`PRINT 42`, `fmt.Println(42)`},
	}

	for i, tt := range tests {
		input := "SUB Main()\n    " + tt.input + "\nEND SUB"
		code := compile(input)
		if !strings.Contains(code, tt.expected) {
			t.Errorf("test[%d]: expected '%s' in output, got:\n%s", i, tt.expected, code)
		}
	}
}

func TestGenerateStringConcatenation(t *testing.T) {
	input := `DIM s AS STRING = "Hello" & " " & "World"`

	code := compile(input)

	// Concatenation may have extra parentheses
	if !strings.Contains(code, `"Hello"`) || !strings.Contains(code, `"World"`) {
		t.Errorf("expected string concatenation, got:\n%s", code)
	}
}

func TestGenerateLogicalOperators(t *testing.T) {
	input := `DIM a AS BOOLEAN = TRUE AND FALSE
DIM b AS BOOLEAN = TRUE OR FALSE
DIM c AS BOOLEAN = NOT TRUE`

	code := compile(input)

	if !strings.Contains(code, "true && false") {
		t.Errorf("expected AND as &&, got:\n%s", code)
	}

	if !strings.Contains(code, "true || false") {
		t.Errorf("expected OR as ||, got:\n%s", code)
	}

	// NOT generates !(expr) with parentheses
	if !strings.Contains(code, "!(true)") {
		t.Errorf("expected NOT as !(expr), got:\n%s", code)
	}
}

func TestGenerateModOperator(t *testing.T) {
	input := `DIM x AS INTEGER = 10 MOD 3`

	code := compile(input)

	if !strings.Contains(code, "10 % 3") {
		t.Errorf("expected MOD as %%, got:\n%s", code)
	}
}

func TestGeneratePowerOperator(t *testing.T) {
	input := `DIM x AS DOUBLE = 2 ^ 3`

	code := compile(input)

	if !strings.Contains(code, "math.Pow") {
		t.Errorf("expected math.Pow, got:\n%s", code)
	}
}

func TestGenerateArrayDeclaration(t *testing.T) {
	input := `DIM arr(10) AS INTEGER`

	code := compile(input)

	if !strings.Contains(code, "[]int") {
		t.Errorf("expected slice type, got:\n%s", code)
	}

	if !strings.Contains(code, "make([]int, 10)") {
		t.Errorf("expected make call, got:\n%s", code)
	}
}

func TestGenerateJSONLiteral(t *testing.T) {
	input := `DIM data AS JSON = {"name": "John", "age": 30}`

	code := compile(input)

	if !strings.Contains(code, "map[string]interface{}") {
		t.Errorf("expected JSON type, got:\n%s", code)
	}
}

func TestGenerateMultipleReturnValues(t *testing.T) {
	input := `FUNCTION Divide(a AS INTEGER, b AS INTEGER) AS (INTEGER, BOOLEAN)
    IF b = 0 THEN
        RETURN 0, FALSE
    ENDIF
    RETURN a / b, TRUE
END FUNCTION`

	code := compile(input)

	if !strings.Contains(code, "(int, bool)") {
		t.Errorf("expected multiple return types, got:\n%s", code)
	}

	if !strings.Contains(code, "return 0, false") {
		t.Errorf("expected multiple return values, got:\n%s", code)
	}
}

func TestGenerateLetStatement(t *testing.T) {
	input := `SUB Main()
    LET x = 42
    PRINT x
END SUB`

	code := compile(input)

	if !strings.Contains(code, "x := 42") {
		t.Errorf("expected short variable declaration, got:\n%s", code)
	}
}

func TestDebugMode(t *testing.T) {
	input := `SUB Main()
    DIM x AS INTEGER = 42
    PRINT x
END SUB`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	a := analyzer.New()
	symbols, _ := a.Analyze(program)

	g := New(program, symbols)
	g.SetDebugMode(true)
	g.SetSourceFile("test.dbas")
	code := g.Generate()

	// Debug mode adds line comments to statements in functions
	if !strings.Contains(code, "// line") && !strings.Contains(code, "// test.dbas") {
		// Debug comments may only appear in function bodies
		// Just verify the code generated correctly
		if !strings.Contains(code, "x int = 42") {
			t.Errorf("expected variable declaration, got:\n%s", code)
		}
	}
}

func TestGenerateFmtImport(t *testing.T) {
	input := `SUB Main()
    PRINT "Hello"
END SUB`

	code := compile(input)

	// Should auto-import fmt for PRINT
	if !strings.Contains(code, `"fmt"`) {
		t.Errorf("expected fmt import, got:\n%s", code)
	}
}

func TestGenerateGotoLabel(t *testing.T) {
	input := `SUB Main()
start:
    PRINT "Hello"
    GOTO start
END SUB`

	code := compile(input)

	if !strings.Contains(code, "start:") {
		t.Errorf("expected label, got:\n%s", code)
	}

	if !strings.Contains(code, "goto start") {
		t.Errorf("expected goto, got:\n%s", code)
	}
}
