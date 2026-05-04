package formatter

import (
	"strings"
	"testing"
)

func TestFormatBasicSpacing(t *testing.T) {
	in := `SUB Main()
DIM x AS INTEGER=5
PRINT  "hello"
END SUB`
	want := `SUB Main()
    DIM x AS INTEGER = 5
    PRINT "hello"
END SUB
`
	got, err := Format(in)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatIfElseIndent(t *testing.T) {
	in := `SUB Main()
IF x = 1 THEN
PRINT "one"
ELSEIF x = 2 THEN
PRINT "two"
ELSE
PRINT "other"
ENDIF
END SUB`
	got, _ := Format(in)
	expected := []string{
		"SUB Main()",
		"    IF x = 1 THEN",
		"        PRINT \"one\"",
		"    ELSEIF x = 2 THEN",
		"        PRINT \"two\"",
		"    ELSE",
		"        PRINT \"other\"",
		"    ENDIF",
		"END SUB",
	}
	for _, line := range expected {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("missing line %q in:\n%s", line, got)
		}
	}
}

func TestFormatSelectCase(t *testing.T) {
	in := `SUB Main()
SELECT CASE x
CASE 1
PRINT "one"
CASE 2
PRINT "two"
END SELECT
END SUB`
	got, _ := Format(in)
	expected := []string{
		"    SELECT CASE x",
		"        CASE 1",
		"            PRINT \"one\"",
		"        CASE 2",
		"            PRINT \"two\"",
		"    END SELECT",
	}
	for _, line := range expected {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("missing line %q in:\n%s", line, got)
		}
	}
}

func TestFormatPreservesComments(t *testing.T) {
	in := `' header
SUB Main()
    ' inline note
    PRINT "hi" ' trailing
END SUB`
	got, _ := Format(in)
	if !strings.Contains(got, "' header") {
		t.Error("lost top-level comment")
	}
	if !strings.Contains(got, "    ' inline note") {
		t.Error("inline comment lost or not indented")
	}
	if !strings.Contains(got, `PRINT "hi"  ' trailing`) {
		t.Errorf("trailing comment lost or wrong spacing:\n%s", got)
	}
}

func TestFormatIdempotent(t *testing.T) {
	in := `SUB Main()
    DIM xs(3) AS INTEGER
    FOR i = 1 TO 10
        IF i MOD 2 = 0 THEN
            PRINT i
        ENDIF
    NEXT i
END SUB
`
	once, _ := Format(in)
	twice, _ := Format(once)
	if once != twice {
		t.Errorf("not idempotent:\nonce:\n%s\ntwice:\n%s", once, twice)
	}
}

// TestFormatKeywordMethodNames covers the regression where reserved
// words used as method names after `.` (e.g. .Error(), .String(),
// .Bytes(), .Type()) were uppercased AND had a stray space inserted
// before the call paren — both of which broke the resulting Go.
func TestFormatKeywordMethodNames(t *testing.T) {
	in := `SUB Demo(e AS ERROR)
PRINT e.Error()
PRINT e.String()
DIM b AS BYTES
PRINT b.Bytes()
END SUB
`
	got, err := Format(in)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	for _, want := range []string{
		"e.Error()",
		"e.String()",
		"b.Bytes()",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in formatted output, got:\n%s", want, got)
		}
	}
	// Negative checks: must not have uppercased the method name nor
	// inserted a space before the paren.
	for _, bad := range []string{"e.ERROR", "e.STRING", "b.BYTES", "Error ()", "String ()", "Bytes ()"} {
		if strings.Contains(got, bad) {
			t.Errorf("unexpected %q in formatted output:\n%s", bad, got)
		}
	}
}

// TestFormatLabelAtColumnZero covers the regression where GOTO target
// labels (e.g. `Bad:`) were indented along with surrounding code
// instead of emitting flush-left.
func TestFormatLabelAtColumnZero(t *testing.T) {
	in := `SUB Demo()
ONERR GOTO Bad
PRINT "ok"
RETURN
Bad:
PRINT "fail"
END SUB
`
	got, err := Format(in)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if !strings.Contains(got, "\nBad:\n") {
		t.Errorf("label `Bad:` not flush-left, got:\n%s", got)
	}
	// Make sure surrounding lines stayed indented.
	if !strings.Contains(got, "    ONERR GOTO Bad") {
		t.Errorf("surrounding ONERR line not indented, got:\n%s", got)
	}
	if !strings.Contains(got, "    RETURN") {
		t.Errorf("surrounding RETURN not indented, got:\n%s", got)
	}
}
