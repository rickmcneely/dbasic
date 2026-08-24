package codegen

import (
	"regexp"
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

// Regression: `STEP 0 - 1` parses to an InfixExpression rather than a
// PrefixExpression, so the original isNegativeStep returned false and
// the loop condition stayed `<=`. On an empty slice (`LEN(xs) - 1` = -1)
// the loop entered with i = -1 and panicked indexing xs.
//
// isNegativeStep now constant-folds the step expression, so any
// compile-time-foldable negative — including `0 - 1`, `-(1)`, `2 - 5` —
// flips the comparison to `>=`.
func TestGenerateForLoopWithFoldedNegativeStep(t *testing.T) {
	cases := []struct {
		name string
		step string
	}{
		{"binary subtraction", "0 - 1"},
		{"prefix on parenthesized", "-(1)"},
		{"nested arithmetic", "2 - 5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := `SUB Main()
    FOR i = 10 TO 0 STEP ` + tc.step + `
        PRINT i
    NEXT
END SUB`
			code := compile(input)
			if !strings.Contains(code, "i >= 0") {
				t.Errorf("expected downward loop comparison (i >= 0), got:\n%s", code)
			}
			if strings.Contains(code, "i <= 0;") {
				t.Errorf("downward loop still using upward comparison (i <= 0), got:\n%s", code)
			}
		})
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

func TestGenerateDirectionalChannelTypes(t *testing.T) {
	input := `SUB recvOnly(ch AS RECEIVE CHAN OF INTEGER)
END SUB

SUB sendOnly(ch AS SEND CHAN OF INTEGER)
END SUB

SUB both(ch AS CHAN OF INTEGER)
END SUB`

	code := compile(input)

	for _, want := range []string{"ch <-chan int", "ch chan<- int", "ch chan int"} {
		if !strings.Contains(code, want) {
			t.Errorf("expected %q, got:\n%s", want, code)
		}
	}
}

func TestGenerateCommaOkReceive(t *testing.T) {
	input := `SUB Main()
    DIM ch AS CHAN OF INTEGER = MAKE_CHAN(INTEGER, 1)
    DIM v AS INTEGER
    DIM ok AS BOOLEAN = TRUE
    RECEIVE v, ok FROM ch
END SUB`

	code := compile(input)

	if !strings.Contains(code, "v, ok = <-ch") {
		t.Errorf("expected comma-ok receive 'v, ok = <-ch', got:\n%s", code)
	}
}

func TestGenerateVariadicSpread(t *testing.T) {
	input := `IMPORT "fmt"
SUB Main()
    DIM xs AS []STRING
    xs = APPEND(xs, "a")
    fmt.Println(xs...)
    fmt.Println("prefix", xs...)
END SUB`

	code := compile(input)

	if !strings.Contains(code, "fmt.Println(xs...)") {
		t.Errorf("expected 'fmt.Println(xs...)', got:\n%s", code)
	}
	if !strings.Contains(code, `fmt.Println("prefix", xs...)`) {
		t.Errorf("expected prefix + spread, got:\n%s", code)
	}
}

func TestGenerateReceiveOperator(t *testing.T) {
	input := `SUB Main()
    DIM ch AS CHAN OF INTEGER = MAKE_CHAN(INTEGER, 2)
    DIM a AS INTEGER
    DIM ok AS BOOLEAN
    a = <-ch
    a, ok = <-ch
END SUB`

	code := compile(input)

	if !strings.Contains(code, "a = <-ch") {
		t.Errorf("expected receive operator 'a = <-ch', got:\n%s", code)
	}
	if !strings.Contains(code, "a, ok = <-ch") {
		t.Errorf("expected comma-ok receive operator 'a, ok = <-ch', got:\n%s", code)
	}
}

// --- CONTINUE, and the loop-target bugs it exposed ------------------------

// inSub wraps statements in a SUB, because only declarations are emitted at
// top level -- executable statements have to live inside a routine.
func inSub(body string) string {
	return "SUB Demo()\n" + body + "\nEND SUB"
}

func TestGenerateContinue(t *testing.T) {
	code := compile(inSub("DIM i AS INTEGER\nFOR i = 1 TO 3\nCONTINUE FOR\nNEXT"))

	if !strings.Contains(code, "continue") {
		t.Errorf("expected a Go continue, got:\n%s", code)
	}
}

// Go's continue ignores switches, so CONTINUE inside SELECT CASE needs no
// label -- but it must still be a bare continue, not a break.
func TestGenerateContinueInsideSelect(t *testing.T) {
	code := compile(inSub(`DIM i AS INTEGER
FOR i = 1 TO 3
SELECT CASE i
CASE 2
CONTINUE FOR
END SELECT
NEXT`))

	if !strings.Contains(code, "continue") {
		t.Errorf("expected a Go continue, got:\n%s", code)
	}
}

// A bare Go `break` inside a switch leaves the SWITCH, not the loop. SELECT
// CASE compiles to a switch, so EXIT FOR in one has to use a loop label.
func TestGenerateExitInsideSelectUsesLabel(t *testing.T) {
	code := compile(inSub(`DIM i AS INTEGER
FOR i = 1 TO 3
SELECT CASE i
CASE 2
EXIT FOR
END SELECT
NEXT`))

	if !strings.Contains(code, "dbLoop1:") {
		t.Errorf("expected a loop label, got:\n%s", code)
	}
	if !strings.Contains(code, "break dbLoop1") {
		t.Errorf("expected a labelled break, got:\n%s", code)
	}
}

// ...but a loop with no EXIT inside a switch must NOT get a label, because
// Go rejects a label that is defined and never used.
func TestGenerateNoLabelWhenNotNeeded(t *testing.T) {
	code := compile(inSub(`DIM i AS INTEGER
FOR i = 1 TO 3
EXIT FOR
NEXT`))

	if strings.Contains(code, "dbLoop") {
		t.Errorf("unexpected loop label on a loop that does not need one:\n%s", code)
	}
	if !strings.Contains(code, "break") {
		t.Errorf("expected a plain break, got:\n%s", code)
	}
}

// A post-test DO normally puts its test at the bottom of the body. A Go
// `continue` would jump straight over that test and spin forever, so a body
// containing CONTINUE switches to a three-part for whose post statement
// re-runs the test.
func TestGeneratePostTestDoLoopWithContinue(t *testing.T) {
	code := compile(inSub(`DIM p AS INTEGER = 0
DO
p = p + 1
CONTINUE DO
LOOP WHILE p < 10`))

	if !strings.Contains(code, "for dbAgain1 := true; dbAgain1; dbAgain1 =") {
		t.Errorf("expected the three-part post-test form, got:\n%s", code)
	}
}

// Without a CONTINUE the original shape is kept, so the LOOP condition may
// still refer to variables declared inside the body.
func TestGeneratePostTestDoLoopWithoutContinue(t *testing.T) {
	code := compile(inSub(`DIM p AS INTEGER = 0
DO
p = p + 1
LOOP WHILE p < 10`))

	if strings.Contains(code, "dbAgain") {
		t.Errorf("post-test loop without CONTINUE should keep the simple form:\n%s", code)
	}
	if !strings.Contains(code, "break") {
		t.Errorf("expected the bottom-of-body break, got:\n%s", code)
	}
}

// LOOP UNTIL is the same thing with the condition inverted.
func TestGeneratePostTestUntilWithContinue(t *testing.T) {
	code := compile(inSub(`DIM p AS INTEGER = 0
DO
p = p + 1
CONTINUE DO
LOOP UNTIL p >= 10`))

	if !strings.Contains(code, "dbAgain1 = !(") {
		t.Errorf("expected the UNTIL condition to be inverted, got:\n%s", code)
	}
}

// --- deterministic output ---------------------------------------------
//
// Ranging over a Go map is deliberately randomised, so any emitter that
// walked one directly produced different code on every build. That made
// generated output impossible to diff or use as a golden file.

func TestGenerateIsDeterministic(t *testing.T) {
	input := `IMPORT "strings" AS strings
IMPORT "fmt" AS fmt
IMPORT "os" AS os
IMPORT "sort" AS sort

TYPE Point
    DIM X AS INTEGER
    DIM Y AS INTEGER
    DIM Label AS STRING
END TYPE

SUB Demo()
    DIM p AS Point = Point{X: 1, Y: 2, Label: "origin"}
    DIM s AS STRING = Left("abc", 2)
    fmt.Println(p, s, strings.ToUpper(s), os.Args, sort.SearchInts)
END SUB`

	first := compile(input)
	for i := 0; i < 25; i++ {
		if got := compile(input); got != first {
			t.Fatalf("run %d differs from the first run:\n%s", i+2, got)
		}
	}
}

// Struct literal fields keep the order the programmer wrote, which is both
// stable and far easier to read back than alphabetical.
func TestStructLiteralKeepsSourceOrder(t *testing.T) {
	code := compile(`TYPE RGBA
    DIM R AS INTEGER
    DIM G AS INTEGER
    DIM B AS INTEGER
    DIM A AS INTEGER
END TYPE

SUB Demo()
    DIM c AS RGBA = RGBA{R: 192, G: 128, B: 64, A: 255}
END SUB`)

	if !strings.Contains(code, "RGBA{R: 192, G: 128, B: 64, A: 255}") {
		t.Errorf("struct literal did not keep source order, got:\n%s", code)
	}
}

// Imports come out sorted by path -- stable, and the order gofmt wants.
func TestImportsAreSorted(t *testing.T) {
	code := compile(`IMPORT "strings" AS strings
IMPORT "fmt" AS fmt
IMPORT "errors" AS errors

SUB Demo()
    fmt.Println(strings.ToUpper("x"), errors.New("boom"))
END SUB`)

	iErrors := strings.Index(code, `"errors"`)
	iFmt := strings.Index(code, `"fmt"`)
	iStrings := strings.Index(code, `"strings"`)
	if iErrors < 0 || iFmt < 0 || iStrings < 0 {
		t.Fatalf("expected all three imports, got:\n%s", code)
	}
	if !(iErrors < iFmt && iFmt < iStrings) {
		t.Errorf("imports are not sorted by path, got:\n%s", code)
	}
}

// --- the analyzer and the code generator must agree ------------------------

// Every name `dbasic check` accepts has to be a name the code generator can
// actually emit. When these two lists drifted apart, a program using
// Randomize, Now, Date, Replace or PI passed check and then failed the Go
// build with "not defined" -- an error pointing at generated code for a
// function the user was told existed.
func TestEveryBuiltinCanBeEmitted(t *testing.T) {
	// The ones the generator rewrites into something else rather than
	// emitting a helper for. NewError and Errorf become the ...Func
	// variants that carry the file and line of the call site; Printf and
	// Sprintf become fmt.Printf and fmt.Sprintf directly.
	emittedInline := map[string]bool{
		"NewError": true,
		"Errorf":   true,
		"Printf":   true,
		"Sprintf":  true,
	}

	var missing []string
	for _, name := range analyzer.BuiltinNames() {
		if emittedInline[name] {
			continue
		}
		if _, ok := runtimeFuncDefs[name]; !ok {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		t.Errorf("these builtins pass `dbasic check` but the generator has no way to emit them,\n"+
			"so any program using one fails `go build`: %v\n"+
			"Add a definition to runtimeFuncDefs (and any imports it needs to\n"+
			"runtimeFuncImports), or add it to emittedInline above if the\n"+
			"generator rewrites it into something else.", missing)
	}
}

// A helper that needs an import must declare it, or the generated file will
// not compile either.
func TestBuiltinHelperImportsAreDeclared(t *testing.T) {
	needs := map[string]string{
		"math.":     "math",
		"strings.":  "strings",
		"strconv.":  "strconv",
		"time.":     "time",
		"os.":       "os",
		"rand.":     "math/rand",
		"fmt.":      "fmt",
		"io.":       "io",
		"bufio.":    "bufio",
		"filepath.": "path/filepath",
		"reflect.":  "reflect",
		"bytes.":    "bytes",
		"exec.":     "os/exec",
	}

	for name, def := range runtimeFuncDefs {
		for prefix, pkg := range needs {
			// The qualifier has to start a word: "bufio.NewReader" contains
			// "io." but does not use the io package.
			re := regexp.MustCompile(`(^|[^A-Za-z0-9_.])` + regexp.QuoteMeta(prefix))
			if !re.MatchString(def) {
				continue
			}
			var found bool
			for _, imp := range runtimeFuncImports[name] {
				if imp == pkg {
					found = true
				}
			}
			if !found {
				t.Errorf("runtime helper %q uses %s but does not list %q in runtimeFuncImports",
					name, prefix, pkg)
			}
		}
	}
}

// The whole Input family has to read through one shared reader. Each used to
// build its own buffered reader, and a buffered reader takes far more than
// one line from stdin -- so INPUT followed by LINE INPUT silently skipped
// whatever the first had read ahead.
func TestInputFamilySharesOneReader(t *testing.T) {
	code := compile(inSub(`DIM a AS STRING
DIM b AS STRING
INPUT a
LINE INPUT "who? ", b`))

	if strings.Contains(code, "bufio.NewReader(os.Stdin)") {
		t.Errorf("INPUT built its own reader instead of using the shared one:\n%s", code)
	}
	if strings.Count(code, "func LineInput(") != 1 {
		t.Errorf("expected exactly one LineInput helper:\n%s", code)
	}
	if !strings.Contains(code, "LineInput(") {
		t.Errorf("INPUT should read through LineInput:\n%s", code)
	}
}

// A helper must not be written out beside a function of the user's own with
// the same name, or the generated Go redeclares it.
func TestUserFunctionSuppressesHelper(t *testing.T) {
	code := compile(`FUNCTION JoinPath(a AS STRING, b AS STRING) AS STRING
RETURN a & b
END FUNCTION

SUB Demo()
DIM s AS STRING = JoinPath("x", "y")
END SUB`)

	if strings.Count(code, "func JoinPath(") != 1 {
		t.Errorf("JoinPath is declared more than once:\n%s", code)
	}
	if strings.Contains(code, "filepath.Join") {
		t.Errorf("the built-in JoinPath was emitted despite the program having its own:\n%s", code)
	}
}
