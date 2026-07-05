package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/zditech/dbasic/pkg/analyzer"
	"github.com/zditech/dbasic/pkg/codegen"
	"github.com/zditech/dbasic/pkg/formatter"
	"github.com/zditech/dbasic/pkg/lexer"
	"github.com/zditech/dbasic/pkg/parser"
	"github.com/zditech/dbasic/pkg/preprocessor"
)

const version = "0.2.0"

var (
	debugMode             bool
	verboseMode           bool
	outputFile            string
	targetSpec            string
	offlineMode           bool
	fmtWrite              bool
	fmtList               bool
	allowExternalIncludes bool
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Handle flags after command
	flagSet := flag.NewFlagSet(command, flag.ExitOnError)
	flagSet.BoolVar(&debugMode, "debug", false, "Enable debug mode (include source line comments)")
	flagSet.BoolVar(&verboseMode, "v", false, "Verbose output")
	flagSet.StringVar(&outputFile, "o", "", "Output file name")
	flagSet.StringVar(&targetSpec, "target", "", "Cross-compile target as os/arch (e.g. windows/amd64, linux/arm64, darwin/arm64)")
	flagSet.BoolVar(&offlineMode, "offline", false, "Build using only cached Go modules (sets GOPROXY=off and pins to the latest cached versions; useful when proxy.golang.org is unreachable)")
	flagSet.BoolVar(&fmtWrite, "w", false, "Write formatted output back to source file (fmt only)")
	flagSet.BoolVar(&fmtList, "l", false, "List files whose formatting differs from the formatter's output (fmt only)")
	flagSet.BoolVar(&allowExternalIncludes, "allow-external-includes", false, "Permit INCLUDE of files outside the project directory (absolute paths or ../ traversal). Off by default for safety.")

	switch command {
	case "build":
		if len(os.Args) < 3 {
			errorf("no input file specified")
			fmt.Fprintln(os.Stderr, "Usage: dbasic build [-o output] [-debug] <file.dbas>")
			os.Exit(1)
		}
		flagSet.Parse(os.Args[3:])
		build(os.Args[2], outputFile)
	case "run":
		if len(os.Args) < 3 {
			errorf("no input file specified")
			fmt.Fprintln(os.Stderr, "Usage: dbasic run [-debug] <file.dbas>")
			os.Exit(1)
		}
		flagSet.Parse(os.Args[3:])
		run(os.Args[2])
	case "emit":
		if len(os.Args) < 3 {
			errorf("no input file specified")
			fmt.Fprintln(os.Stderr, "Usage: dbasic emit [-debug] <file.dbas>")
			os.Exit(1)
		}
		flagSet.Parse(os.Args[3:])
		emit(os.Args[2])
	case "check":
		if len(os.Args) < 3 {
			errorf("no input file specified")
			fmt.Fprintln(os.Stderr, "Usage: dbasic check <file.dbas>")
			os.Exit(1)
		}
		flagSet.Parse(os.Args[3:])
		check(os.Args[2])
	case "fmt", "format":
		if len(os.Args) < 3 {
			errorf("no input file specified")
			fmt.Fprintln(os.Stderr, "Usage: dbasic fmt [-w] [-l] <file.dbas>...")
			os.Exit(1)
		}
		flagSet.Parse(os.Args[3:])
		formatFiles(append([]string{os.Args[2]}, flagSet.Args()...))
	case "doc":
		if len(os.Args) < 3 {
			errorf("no input file specified")
			fmt.Fprintln(os.Stderr, "Usage: dbasic doc [-o output.md] <file.dbas>")
			os.Exit(1)
		}
		flagSet.Parse(os.Args[3:])
		docFile(os.Args[2], outputFile)
	case "test":
		path := "."
		if len(os.Args) >= 3 {
			path = os.Args[2]
			flagSet.Parse(os.Args[3:])
		}
		testCommand(path)
	case "version", "-version", "--version":
		fmt.Printf("DBasic Compiler v%s\n", version)
	case "help", "-help", "--help", "-h":
		printUsage()
	default:
		// Treat as file if it ends with .dbas
		if strings.HasSuffix(command, ".dbas") {
			run(command)
		} else {
			errorf("unknown command: %s", command)
			printUsage()
			os.Exit(1)
		}
	}
}

// errorf prints an error message to stderr
func errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}

// goToolFilter strips noise from `go build` / `go mod tidy` stderr so the
// user sees DBasic-flavored diagnostics. Drops the `# dbasic_program` and
// `# command-line-arguments` package-header lines and rewrites Go-side path
// hints when they reference the temp file `main.go` or `dbasic_program`.
type goToolFilter struct {
	out io.Writer
	buf bytes.Buffer
}

func (f *goToolFilter) Write(p []byte) (int, error) {
	f.buf.Write(p)
	for {
		idx := bytes.IndexByte(f.buf.Bytes(), '\n')
		if idx < 0 {
			break
		}
		line := f.buf.Next(idx + 1)
		clean := cleanGoToolLine(string(line))
		if clean == "" {
			continue
		}
		if _, err := io.WriteString(f.out, clean); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (f *goToolFilter) Flush() {
	if f.buf.Len() == 0 {
		return
	}
	clean := cleanGoToolLine(f.buf.String())
	if clean != "" {
		io.WriteString(f.out, clean)
	}
	f.buf.Reset()
}

func cleanGoToolLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "# dbasic_program") ||
		strings.HasPrefix(trimmed, "# command-line-arguments") {
		return ""
	}
	// Strip the temp `main.go` / `dbasic_program` references that occasionally
	// surface in dependency errors. Leave file:line references that already
	// point to .dbas alone (handled by //line directives).
	if strings.Contains(line, "/dbasic-") && strings.Contains(line, "/main.go") {
		// Pattern: /tmp/dbasic-XXXX/main.go:LINE: msg → main.go:LINE: msg
		if i := strings.Index(line, "/main.go"); i >= 0 {
			j := strings.LastIndex(line[:i], "/")
			if j >= 0 {
				line = line[j+1:]
			}
		}
	}
	line = rewriteGoErrorPhrasing(line)
	return cleanGoToolMultilineHints(line)
}

// rewriteGoErrorPhrasing translates common Go compiler diagnostic wordings
// into BASIC-flavored phrasing so users aren't reading Go-isms in errors
// that originated from their .dbas source. The transforms are line-based;
// each rule is tried in order, and the first match wins.
func rewriteGoErrorPhrasing(line string) string {
	// Preserve trailing newline so the caller's line buffering still works.
	suffix := ""
	if strings.HasSuffix(line, "\n") {
		suffix = "\n"
		line = strings.TrimRight(line, "\n")
	}
	for _, r := range goErrorRewrites {
		if m := r.re.FindStringSubmatch(line); m != nil {
			return r.fn(m) + suffix
		}
	}
	return line + suffix
}

// simplifyGoTypeDesc reduces Go's wordy in-paren type descriptions to a bare
// type name when possible. Examples:
//
//	"variable of type int"          → "int"
//	"untyped int constant"          → "int"
//	"untyped string constant"       → "string"
//	"constant 5 of type int"        → "int"
//	"value of type (int, error)"    → "(int, error)"
//	anything else                   → returned unchanged
func simplifyGoTypeDesc(desc string) string {
	if i := strings.Index(desc, " of type "); i >= 0 {
		return strings.TrimSpace(desc[i+len(" of type "):])
	}
	if strings.HasPrefix(desc, "untyped ") && strings.HasSuffix(desc, " constant") {
		return strings.TrimSuffix(strings.TrimPrefix(desc, "untyped "), " constant")
	}
	return desc
}

var goErrorRewrites = []struct {
	re *regexp.Regexp
	fn func([]string) string
}{
	// cannot use X (DESC) as T2 value in argument to F
	// DESC is "variable of type T", "untyped int constant", "constant 5 of type int", etc.
	{
		regexp.MustCompile(`^(.*?):\s*cannot use (.+?) \(([^)]+)\) as (\S+?) (?:value )?in argument to (\S+)`),
		func(m []string) string {
			callee := strings.TrimRight(m[5], ":,;")
			return fmt.Sprintf("%s: type mismatch: %s is %s but %s expects %s",
				m[1], m[2], simplifyGoTypeDesc(m[3]), callee, m[4])
		},
	},
	// cannot use X (DESC) as T2 value in <context>
	{
		regexp.MustCompile(`^(.*?):\s*cannot use (.+?) \(([^)]+)\) as (\S+?) (?:value )?in (\S+)`),
		func(m []string) string {
			return fmt.Sprintf("%s: type mismatch: cannot use %s (%s) as %s",
				m[1], m[2], simplifyGoTypeDesc(m[3]), m[4])
		},
	},
	// multiple-value F(...) (value of type (T1, T2)) in single-value context
	{
		regexp.MustCompile(`^(.*?):\s*multiple-value (.+?) \([^)]*type \(([^)]+)\)\) in single-value context`),
		func(m []string) string {
			return fmt.Sprintf("%s: %s returns multiple values (%s); assign all of them or discard with _", m[1], m[2], m[3])
		},
	},
	// undefined: X
	{
		regexp.MustCompile(`^(.*?):\s*undefined: (.+)$`),
		func(m []string) string {
			return fmt.Sprintf("%s: %s is not defined", m[1], m[2])
		},
	},
	// X (variable of type T) is not used  /  declared and not used
	{
		regexp.MustCompile(`^(.*?):\s*(\S+?) \([^)]*?of type (.+?)\) (?:is |declared and )(?:not used|declared but not used)`),
		func(m []string) string {
			return fmt.Sprintf("%s: variable %s (%s) is declared but never used", m[1], m[2], m[3])
		},
	},
	// declared and not used: X (older Go phrasing)
	{
		regexp.MustCompile(`^(.*?):\s*(\S+) declared and not used$`),
		func(m []string) string {
			return fmt.Sprintf("%s: variable %s is declared but never used", m[1], m[2])
		},
	},
	// X redeclared in this block
	{
		regexp.MustCompile(`^(.*?):\s*(\S+?) redeclared in this block`),
		func(m []string) string {
			return fmt.Sprintf("%s: %s is already declared in this scope", m[1], m[2])
		},
	},
	// not enough arguments in call to F
	{
		regexp.MustCompile(`^(.*?):\s*not enough arguments in call to (\S+)`),
		func(m []string) string {
			return fmt.Sprintf("%s: too few arguments passed to %s", m[1], m[2])
		},
	},
	// too many arguments in call to F
	{
		regexp.MustCompile(`^(.*?):\s*too many arguments in call to (\S+)`),
		func(m []string) string {
			return fmt.Sprintf("%s: too many arguments passed to %s", m[1], m[2])
		},
	},
	// assignment mismatch: N variables but M values
	{
		regexp.MustCompile(`^(.*?):\s*assignment mismatch: (\d+) variables? but (\d+) values?`),
		func(m []string) string {
			return fmt.Sprintf("%s: assignment mismatch: %s names on the left, %s values on the right", m[1], m[2], m[3])
		},
	},
	// invalid operation: X (mismatched types T1 and T2)
	{
		regexp.MustCompile(`^(.*?):\s*invalid operation: (.+?) \(mismatched types (.+?) and (.+?)\)`),
		func(m []string) string {
			return fmt.Sprintf("%s: cannot combine %s — types %s and %s do not match", m[1], m[2], m[3], m[4])
		},
	},
	// X does not implement Y (missing Z method)
	{
		regexp.MustCompile(`^(.*?):\s*(\S+) does not implement (\S+) \(missing (?:method )?(\S+) method\)`),
		func(m []string) string {
			return fmt.Sprintf("%s: %s does not satisfy interface %s — missing method %s", m[1], m[2], m[3], m[4])
		},
	},
	// X does not implement Y (M method has pointer receiver)
	{
		regexp.MustCompile(`^(.*?):\s*(\S+) does not implement (\S+) \((\S+) method has pointer receiver\)`),
		func(m []string) string {
			return fmt.Sprintf("%s: %s does not satisfy %s — method %s needs a POINTER TO receiver; pass @%s instead", m[1], m[2], m[3], m[4], m[2])
		},
	},
	// cannot send on receive-only channel
	{
		regexp.MustCompile(`^(.*?):\s*cannot send on receive-only channel`),
		func(m []string) string {
			return fmt.Sprintf("%s: cannot SEND on a receive-only channel", m[1])
		},
	},
	// cannot receive from send-only channel
	{
		regexp.MustCompile(`^(.*?):\s*(?:cannot receive from send-only channel|receive from send-only channel)`),
		func(m []string) string {
			return fmt.Sprintf("%s: cannot RECEIVE from a send-only channel", m[1])
		},
	},
	// X (untyped int constant N) overflows T
	{
		regexp.MustCompile(`^(.*?):\s*(.+?) \(untyped (\S+) constant (.+?)\) overflows (\S+)`),
		func(m []string) string {
			return fmt.Sprintf("%s: literal %s does not fit in %s", m[1], m[4], m[5])
		},
	},
	// non-name X on left side of :=
	{
		regexp.MustCompile(`^(.*?):\s*non-name (.+?) on left side of :=`),
		func(m []string) string {
			return fmt.Sprintf("%s: %s cannot appear on the left of an assignment — use a variable name", m[1], m[2])
		},
	},
	// cannot range over X (...)
	{
		regexp.MustCompile(`^(.*?):\s*cannot range over (.+?) \(.*?\)`),
		func(m []string) string {
			return fmt.Sprintf("%s: cannot iterate over %s — needs to be a slice, array, map, or channel", m[1], m[2])
		},
	},
	// missing return at end of function
	{
		regexp.MustCompile(`^(.*?):\s*missing return(?:\b| at end of function)`),
		func(m []string) string {
			return fmt.Sprintf("%s: function reached its end without RETURN", m[1])
		},
	},
}

// cleanGoToolMultilineHints rewrites tab-indented continuation lines that
// Go emits after certain errors (have/want blocks). These come on lines
// after the main error and use Go-flavored phrasing.
func cleanGoToolMultilineHints(line string) string {
	if strings.HasPrefix(line, "\thave ") {
		return "\tgot:      " + strings.TrimPrefix(line, "\thave ")
	}
	if strings.HasPrefix(line, "\twant ") {
		return "\texpected: " + strings.TrimPrefix(line, "\twant ")
	}
	return line
}

// warnf prints a warning message to stderr
func warnf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// infof prints an info message to stderr (only in verbose mode)
func infof(format string, args ...interface{}) {
	if verboseMode {
		fmt.Fprintf(os.Stderr, "info: "+format+"\n", args...)
	}
}

func printUsage() {
	fmt.Println("DBasic Compiler - A BASIC to Go transpiler")
	fmt.Printf("Version %s\n\n", version)
	fmt.Println("Usage: dbasic <command> [options] [arguments]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  build <file.dbas>     Compile to executable")
	fmt.Println("  run <file.dbas>       Compile and run")
	fmt.Println("  emit <file.dbas>      Output generated Go code")
	fmt.Println("  check <file.dbas>     Check for errors without compiling")
	fmt.Println("  fmt <file.dbas>...    Reformat source (use -w to write, -l to list)")
	fmt.Println("  doc <file.dbas>       Emit Markdown API docs (use -o to write to file)")
	fmt.Println("  test [path]           Run *_test.dbas files (subs named Test*)")
	fmt.Println("  version               Print version")
	fmt.Println("  help                  Print this help")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -o <file>             Output file name (for build)")
	fmt.Println("  -debug                Include source line comments in output")
	fmt.Println("  -v                    Verbose output")
	fmt.Println("  --target os/arch      Cross-compile (e.g. windows/amd64, linux/arm64)")
	fmt.Println("  --offline             Build using only cached Go modules")
	fmt.Println("  --allow-external-includes")
	fmt.Println("                        Permit INCLUDE outside the project dir (off by default)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  dbasic build hello.dbas           # Creates hello executable")
	fmt.Println("  dbasic build -o myapp hello.dbas  # Creates myapp executable")
	fmt.Println("  dbasic run hello.dbas             # Compile and run")
	fmt.Println("  dbasic emit hello.dbas            # Print Go code to stdout")
	fmt.Println("  dbasic check hello.dbas           # Syntax/semantic check only")
}

// CompileResult holds the result of compilation
type CompileResult struct {
	GoCode     string
	SourceFile string
	Errors     []CompileError
	Warnings   []CompileError
}

// CompileError represents a compilation error with location
type CompileError struct {
	File    string
	Line    int
	Column  int
	Message string
	Phase   string // "lexer", "parser", "analyzer", "codegen"
}

func (e CompileError) String() string {
	if e.Line > 0 {
		if e.Column > 0 {
			return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
		}
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.File, e.Message)
}

func compile(filename string) (*CompileResult, error) {
	result := &CompileResult{
		SourceFile: filename,
	}

	// Preprocess (handle INCLUDE directives)
	pp := preprocessor.New(filepath.Dir(filename))
	pp.SetAllowExternal(allowExternalIncludes)
	ppResult, err := pp.Process(filename)
	if err != nil {
		return nil, err
	}

	source := ppResult.Source

	if len(ppResult.IncludedFiles) > 1 {
		infof("preprocessing complete: %d files included", len(ppResult.IncludedFiles)-1)
	}

	infof("compiling %s (%d bytes)", filename, len(source))

	// Tokenize
	l := lexer.New(source)

	// Parse
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			result.Errors = append(result.Errors, CompileError{
				File:    filename,
				Message: e,
				Phase:   "parser",
			})
		}
		return result, fmt.Errorf("parsing failed with %d error(s)", len(p.Errors()))
	}

	infof("parsed %d statements", len(program.Statements))

	// Analyze
	a := analyzer.New()
	a.SetSource(string(source)) // Set source for error context
	symbols, errors := a.Analyze(program)

	if len(errors) > 0 {
		for _, e := range errors {
			result.Errors = append(result.Errors, CompileError{
				File:    filename,
				Message: e,
				Phase:   "analyzer",
			})
		}
		return result, fmt.Errorf("analysis failed with %d error(s)", len(errors))
	}

	// Check for Main sub
	if !a.HasMain() {
		result.Warnings = append(result.Warnings, CompileError{
			File:    filename,
			Message: "no Main() sub found - program may not execute",
			Phase:   "analyzer",
		})
	}

	infof("analysis complete, %d symbols defined", len(symbols.GlobalScope.AllSymbols()))

	// Generate Go code
	g := codegen.New(program, symbols)
	g.SetDebugMode(debugMode)
	g.SetTypeRegistry(a.TypeRegistry())
	g.SetSourceFile(filepath.Base(filename)) // Set source file for error messages
	result.GoCode = g.Generate()

	infof("generated %d bytes of Go code", len(result.GoCode))

	return result, nil
}

func printErrors(result *CompileResult) {
	if result == nil {
		return
	}
	for _, e := range result.Errors {
		// If the message already contains formatting (newlines), print as-is
		if strings.Contains(e.Message, "\n") {
			fmt.Fprint(os.Stderr, e.Message)
		} else {
			fmt.Fprintln(os.Stderr, e.String())
		}
	}
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "\n") {
			fmt.Fprint(os.Stderr, w.Message)
		} else {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w.String())
		}
	}
}

func check(filename string) {
	result, err := compile(filename)
	if err != nil {
		printErrors(result)
		errorf("%v", err)
		os.Exit(1)
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w.String())
	}

	fmt.Printf("%s: OK\n", filename)
}

// testCommand finds *_test.dbas files under path (or just path if it's
// a single file) and runs each one in test-runner mode: every SUB whose
// name starts with "Test" is invoked inside a recover() wrapper, and
// PASS/FAIL is reported. Exits non-zero if any test fails.
func testCommand(path string) {
	files, err := findTestFiles(path)
	if err != nil {
		errorf("scanning %s: %v", path, err)
		os.Exit(1)
	}
	if len(files) == 0 {
		errorf("no *_test.dbas files found under %s", path)
		os.Exit(1)
	}
	totalPassed, totalFailed := 0, 0
	for _, f := range files {
		fmt.Fprintf(os.Stderr, "--- %s ---\n", f)
		passed, failed, err := runTestFile(f)
		if err != nil {
			errorf("%s: %v", f, err)
			totalFailed++
			continue
		}
		totalPassed += passed
		totalFailed += failed
	}
	fmt.Fprintf(os.Stderr, "\n=== TOTAL: %d passed, %d failed across %d file(s) ===\n",
		totalPassed, totalFailed, len(files))
	if totalFailed > 0 {
		os.Exit(1)
	}
}

func findTestFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if strings.HasSuffix(path, "_test.dbas") {
			return []string{path}, nil
		}
		return nil, fmt.Errorf("%s is not a *_test.dbas file", path)
	}
	var files []string
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, "_test.dbas") {
			files = append(files, p)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func runTestFile(filename string) (passed, failed int, err error) {
	pp := preprocessor.New(filepath.Dir(filename))
	ppResult, perr := pp.Process(filename)
	if perr != nil {
		return 0, 0, perr
	}
	source := ppResult.Source
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return 0, 0, fmt.Errorf("parse errors: %v", p.Errors())
	}

	// Find Test* SUBs.
	var testNames []string
	for _, st := range program.Statements {
		if ss, ok := st.(*parser.SubStatement); ok {
			if strings.HasPrefix(ss.Name.Value, "Test") {
				testNames = append(testNames, ss.Name.Value)
			}
		}
	}
	if len(testNames) == 0 {
		fmt.Fprintf(os.Stderr, "  (no Test* subs)\n")
		return 0, 0, nil
	}

	a := analyzer.New()
	a.SetSource(source)
	symbols, errs := a.Analyze(program)
	if len(errs) > 0 {
		return 0, 0, fmt.Errorf("analyze errors: %v", errs)
	}

	g := codegen.New(program, symbols)
	g.SetTypeRegistry(a.TypeRegistry())
	g.SetSourceFile(filepath.Base(filename))
	g.SetTestMode(testNames)
	goCode := g.Generate()

	tempDir, err := os.MkdirTemp("", "dbasic-test-*")
	if err != nil {
		return 0, 0, err
	}
	defer os.RemoveAll(tempDir)
	goFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(goFile, []byte(goCode), 0644); err != nil {
		return 0, 0, err
	}
	cmdInit := exec.Command("go", "mod", "init", "dbasic_test")
	cmdInit.Dir = tempDir
	if err := cmdInit.Run(); err != nil {
		return 0, 0, err
	}
	cmdTidy := exec.Command("go", "mod", "tidy")
	cmdTidy.Dir = tempDir
	cmdTidy.Stderr = &goToolFilter{out: os.Stderr}
	_ = cmdTidy.Run()

	cmdRun := exec.Command("go", "run", ".")
	cmdRun.Dir = tempDir
	out, runErr := cmdRun.CombinedOutput()
	os.Stdout.Write(out)

	// Parse the summary line "=== N passed, M failed ==="
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "=== ") {
			var p, f int
			if _, scanErr := fmt.Sscanf(line, "=== %d passed, %d failed ===", &p, &f); scanErr == nil {
				passed, failed = p, f
			}
		}
	}
	if runErr != nil && failed == 0 {
		// non-zero exit but we couldn't parse a summary — treat as one fail
		failed = 1
	}
	return passed, failed, nil
}

// docFile emits Markdown API documentation for a single .dbas source.
// Walks top-level TYPE/SUB/FUNCTION/CONST/method declarations and
// renders each with its signature and any preceding `'` comments as the
// description. Writes to stdout by default; if outFile is non-empty,
// writes to that path.
func docFile(filename, outFile string) {
	src, err := os.ReadFile(filename)
	if err != nil {
		errorf("reading %s: %v", filename, err)
		os.Exit(1)
	}
	l := lexer.New(string(src))
	tokens := l.Tokenize()
	// Build a map of comment-line → text so we can attach leading
	// comment blocks to declarations by line number.
	commentByLine := make(map[int]string)
	for _, t := range tokens {
		if t.Type == lexer.TOKEN_COMMENT {
			commentByLine[t.Line] = t.Literal
		}
	}
	// Re-parse for the AST.
	p := parser.New(lexer.New(string(src)))
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		errorf("could not parse %s — fix errors first:", filename)
		for _, e := range p.Errors() {
			fmt.Fprintln(os.Stderr, e)
		}
		os.Exit(1)
	}

	var sb strings.Builder
	title := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	sb.WriteString("# " + title + "\n\n")
	sb.WriteString("Generated from `" + filepath.Base(filename) + "` by `dbasic doc`.\n\n")

	for _, st := range prog.Statements {
		emitDocFor(&sb, st, commentByLine)
	}

	out := sb.String()
	if outFile == "" {
		fmt.Print(out)
		return
	}
	if err := os.WriteFile(outFile, []byte(out), 0644); err != nil {
		errorf("writing %s: %v", outFile, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Wrote: %s\n", outFile)
}

func emitDocFor(sb *strings.Builder, st parser.Statement, comments map[int]string) {
	var heading, sig string
	var line int
	switch s := st.(type) {
	case *parser.TypeStatement:
		heading = "type " + s.Name.Value
		line = s.Token.Line
		sig = "TYPE " + s.Name.Value + "\n"
		for _, f := range s.Fields {
			sig += "    DIM " + f.Name.Value + " AS " + typeSpecStr(f.Type) + "\n"
		}
		sig += "END TYPE"
	case *parser.SubStatement:
		heading = "sub " + s.Name.Value
		line = s.Token.Line
		sig = "SUB " + s.Name.Value + paramSigDoc(s.Params)
	case *parser.FunctionStatement:
		heading = "function " + s.Name.Value
		line = s.Token.Line
		sig = "FUNCTION " + s.Name.Value + paramSigDoc(s.Params) + retSigDoc(s.ReturnTypes)
	case *parser.MethodStatement:
		recv := typeSpecStr(s.ReceiverType)
		recvName := ""
		if s.ReceiverName != nil {
			recvName = s.ReceiverName.Value
		}
		heading = recv + "." + s.Name.Value
		line = s.Token.Line
		sig = "FUNCTION (" + recvName + " AS " + recv + ") " + s.Name.Value + paramSigDoc(s.Params) + retSigDoc(s.ReturnTypes)
	case *parser.ConstStatement:
		heading = "const " + s.Name.Value
		line = s.Token.Line
		sig = "CONST " + s.Name.Value + " AS " + typeSpecStr(s.Type)
	case *parser.DimStatement:
		heading = "var " + s.Name.Value
		line = s.Token.Line
		sig = "DIM " + s.Name.Value + " AS " + typeSpecStr(s.Type)
	default:
		return
	}

	sb.WriteString("## " + heading + "\n\n")
	if doc := gatherLeadingComments(comments, line); doc != "" {
		sb.WriteString(doc + "\n\n")
	}
	sb.WriteString("```dbasic\n")
	sb.WriteString(sig)
	sb.WriteString("\n```\n\n")
}

// gatherLeadingComments collects consecutive `'` comment lines immediately
// preceding `line` and joins them as a single Markdown paragraph.
func gatherLeadingComments(comments map[int]string, line int) string {
	var lines []string
	for l := line - 1; l > 0; l-- {
		c, ok := comments[l]
		if !ok {
			break
		}
		lines = append([]string{c}, lines...)
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func paramSigDoc(params []*parser.Parameter) string {
	parts := make([]string, 0, len(params))
	for _, p := range params {
		s := p.Name.Value + " AS " + typeSpecStr(p.Type)
		if p.ByRef {
			s = "BYREF " + s
		}
		parts = append(parts, s)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func retSigDoc(rets []*parser.TypeSpec) string {
	if len(rets) == 0 {
		return ""
	}
	if len(rets) == 1 {
		return " AS " + typeSpecStr(rets[0])
	}
	parts := make([]string, len(rets))
	for i, r := range rets {
		parts[i] = typeSpecStr(r)
	}
	return " AS (" + strings.Join(parts, ", ") + ")"
}

func typeSpecStr(ts *parser.TypeSpec) string {
	if ts == nil {
		return ""
	}
	return ts.String()
}

func formatFiles(filenames []string) {
	exitCode := 0
	for _, fn := range filenames {
		src, err := os.ReadFile(fn)
		if err != nil {
			errorf("reading %s: %v", fn, err)
			exitCode = 1
			continue
		}
		out, err := formatter.Format(string(src))
		if err != nil {
			errorf("formatting %s: %v", fn, err)
			exitCode = 1
			continue
		}
		switch {
		case fmtList:
			if string(src) != out {
				fmt.Println(fn)
			}
		case fmtWrite:
			if string(src) == out {
				continue
			}
			if err := os.WriteFile(fn, []byte(out), 0644); err != nil {
				errorf("writing %s: %v", fn, err)
				exitCode = 1
			}
		default:
			fmt.Print(out)
		}
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func emit(filename string) {
	result, err := compile(filename)
	if err != nil {
		printErrors(result)
		errorf("%v", err)
		os.Exit(1)
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w.String())
	}

	fmt.Print(result.GoCode)
}

func build(filename, outputName string) {
	result, err := compile(filename)
	if err != nil {
		printErrors(result)
		errorf("%v", err)
		os.Exit(1)
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w.String())
	}

	// Parse cross-compile target
	goos, goarch, err := parseTarget(targetSpec)
	if err != nil {
		errorf("%v", err)
		os.Exit(1)
	}

	// Determine output name (auto-append .exe for windows targets)
	if outputName == "" {
		base := filepath.Base(filename)
		outputName = strings.TrimSuffix(base, filepath.Ext(base))
		if goos == "windows" && !strings.HasSuffix(outputName, ".exe") {
			outputName += ".exe"
		}
	}

	// Create temp directory for Go files
	tempDir, err := os.MkdirTemp("", "dbasic-*")
	if err != nil {
		errorf("creating temp directory: %v", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	// Write Go source file
	goFile := filepath.Join(tempDir, "main.go")
	err = os.WriteFile(goFile, []byte(result.GoCode), 0644)
	if err != nil {
		errorf("writing Go file: %v", err)
		os.Exit(1)
	}

	// Initialize Go module in temp directory
	modInit := exec.Command("go", "mod", "init", "dbasic_program")
	modInit.Dir = tempDir
	modInit.Stdout = nil
	modInit.Stderr = nil
	if err := modInit.Run(); err != nil {
		errorf("initializing Go module: %v", err)
		os.Exit(1)
	}

	// Offline mode: scan the user's module cache for the modules this
	// program imports, append them as require lines to go.mod, and force
	// `go mod tidy` to use only cached entries (no proxy, no checksum DB).
	tidyEnv := os.Environ()
	if offlineMode {
		projectGo := readGoDirective(filepath.Join(tempDir, "go.mod"))
		requires := findRequiresForOffline(result.GoCode, projectGo)
		if len(requires) > 0 {
			if err := appendRequires(filepath.Join(tempDir, "go.mod"), requires); err != nil {
				errorf("writing offline requires: %v", err)
				os.Exit(1)
			}
			infof("offline: pinned %d cached module(s)", len(requires))
		}
		tidyEnv = append(tidyEnv, "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=mod")
	}

	// Run go mod tidy to fetch dependencies
	modTidy := exec.Command("go", "mod", "tidy")
	modTidy.Dir = tempDir
	modTidy.Stdout = nil
	tidyFilter := &goToolFilter{out: os.Stderr}
	modTidy.Stderr = tidyFilter
	modTidy.Env = tidyEnv
	if err := modTidy.Run(); err != nil {
		tidyFilter.Flush()
		if !offlineMode {
			fmt.Fprintln(os.Stderr, "hint: if proxy.golang.org is unreachable but you have the modules cached locally, retry with --offline")
		}
		errorf("fetching dependencies: %v", err)
		os.Exit(1)
	}

	// Determine output path
	var outputPath string
	if filepath.IsAbs(outputName) {
		// Absolute path - use as-is
		outputPath = outputName
	} else {
		// Relative path - join with current working directory
		cwd, err := os.Getwd()
		if err != nil {
			errorf("getting current directory: %v", err)
			os.Exit(1)
		}
		outputPath = filepath.Join(cwd, outputName)
	}

	// Build executable
	infof("building %s", outputPath)
	cmd := exec.Command("go", "build", "-o", outputPath, ".")
	cmd.Dir = tempDir
	buildFilter := &goToolFilter{out: os.Stderr}
	cmd.Stderr = buildFilter
	buildEnv := os.Environ()
	if offlineMode {
		buildEnv = append(buildEnv, "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=mod")
	}
	if goos != "" || goarch != "" {
		// Cross-compiling: disable CGO (target toolchain typically unavailable)
		// and set the target via env vars.
		buildEnv = append(buildEnv, "CGO_ENABLED=0")
		if goos != "" {
			buildEnv = append(buildEnv, "GOOS="+goos)
		}
		if goarch != "" {
			buildEnv = append(buildEnv, "GOARCH="+goarch)
		}
		infof("cross-compiling for %s/%s (CGO disabled)", goos, goarch)
	}
	if offlineMode || goos != "" || goarch != "" {
		cmd.Env = buildEnv
	}

	if err := cmd.Run(); err != nil {
		buildFilter.Flush()
		errorf("building executable: %v", err)
		os.Exit(1)
	}
	buildFilter.Flush()

	fmt.Fprintf(os.Stderr, "Built: %s\n", outputPath)
}

// --- offline-mode helpers ----------------------------------------------------
//
// These let `dbasic build --offline` use cached Go modules instead of
// reaching out to proxy.golang.org. Strategy: scan $GOMODCACHE for each
// non-stdlib import in the generated code, pick the latest cached version
// whose own `go` directive is compatible with the project, and pin those
// versions in go.mod. Then go mod tidy / go build run with GOPROXY=off.

var goImportRE = regexp.MustCompile(`(?m)^\s*(?:[A-Za-z_][A-Za-z0-9_]*\s+)?"([^"]+)"`)

// extractGoImports returns the unique set of import paths in goSrc.
func extractGoImports(goSrc string) []string {
	// Find the import block; falls back to single-line imports.
	seen := map[string]bool{}
	var paths []string
	// Match `import "path"` and import (...) blocks.
	importBlockRE := regexp.MustCompile(`(?s)import\s*\(([^)]*)\)`)
	if m := importBlockRE.FindStringSubmatch(goSrc); m != nil {
		for _, sub := range goImportRE.FindAllStringSubmatch(m[1], -1) {
			p := sub[1]
			if !seen[p] {
				seen[p] = true
				paths = append(paths, p)
			}
		}
	}
	importLineRE := regexp.MustCompile(`(?m)^import\s+(?:[A-Za-z_][A-Za-z0-9_]*\s+)?"([^"]+)"`)
	for _, sub := range importLineRE.FindAllStringSubmatch(goSrc, -1) {
		p := sub[1]
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	return paths
}

// isThirdPartyImport returns true for module paths that need a require
// entry — anything whose first segment contains a dot (typically a host).
func isThirdPartyImport(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return strings.Contains(first, ".")
}

// goModCacheDir returns $GOMODCACHE, falling back to $GOPATH/pkg/mod.
func goModCacheDir() string {
	if c := os.Getenv("GOMODCACHE"); c != "" {
		return c
	}
	gp := os.Getenv("GOPATH")
	if gp == "" {
		if home, err := os.UserHomeDir(); err == nil {
			gp = filepath.Join(home, "go")
		}
	}
	return filepath.Join(gp, "pkg", "mod")
}

// semverLess compares two version strings like "v1.2.10" / "v1.10.0".
// Pre-release suffixes are ignored beyond the first three numeric parts.
func semverLess(a, b string) bool {
	aa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(aa) && i < len(bb); i++ {
		ai, _ := strconv.Atoi(strings.SplitN(aa[i], "-", 2)[0])
		bi, _ := strconv.Atoi(strings.SplitN(bb[i], "-", 2)[0])
		if ai != bi {
			return ai < bi
		}
	}
	return len(aa) < len(bb)
}

// readGoDirective returns the version after the "go " directive in a go.mod.
func readGoDirective(modPath string) string {
	data, err := os.ReadFile(modPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go "))
		}
	}
	return ""
}

// findCachedVersions lists the cached versions of a module path under cacheDir.
// Walks back up the import path until it finds a directory with @-versioned
// entries, so it works with both `host/owner/repo` and `host/owner/repo/v2`
// (the cache encodes the major version into the directory name).
func findCachedVersions(cacheDir, importPath string) (modPath string, versions []string) {
	parts := strings.Split(importPath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		base := filepath.Base(candidate)
		parent := filepath.Join(cacheDir, filepath.Dir(candidate))
		entries, err := os.ReadDir(parent)
		if err != nil {
			continue
		}
		var found []string
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, base+"@") {
				found = append(found, strings.TrimPrefix(name, base+"@"))
			}
		}
		if len(found) > 0 {
			return candidate, found
		}
	}
	return "", nil
}

// pickBestCachedVersion returns the highest cached version whose own go.mod
// declares a `go` directive ≤ projectGoVersion. Falls back to the highest
// version overall if nothing is found within constraint.
func pickBestCachedVersion(cacheDir, modPath string, versions []string, projectGoVersion string) string {
	sort.Slice(versions, func(i, j int) bool { return semverLess(versions[i], versions[j]) })
	if projectGoVersion != "" {
		for i := len(versions) - 1; i >= 0; i-- {
			v := versions[i]
			req := readGoDirective(filepath.Join(cacheDir, modPath+"@"+v, "go.mod"))
			if req == "" || !semverLess(projectGoVersion, req) {
				return v
			}
		}
	}
	return versions[len(versions)-1]
}

// findRequiresForOffline returns require lines (e.g. "github.com/foo v1.2.3")
// for each third-party import in goSrc, pinned to the latest cached version
// compatible with projectGoVersion.
func findRequiresForOffline(goSrc, projectGoVersion string) []string {
	cache := goModCacheDir()
	seen := map[string]bool{}
	var requires []string
	for _, imp := range extractGoImports(goSrc) {
		if !isThirdPartyImport(imp) {
			continue
		}
		modPath, versions := findCachedVersions(cache, imp)
		if modPath == "" || seen[modPath] {
			continue
		}
		seen[modPath] = true
		v := pickBestCachedVersion(cache, modPath, versions, projectGoVersion)
		requires = append(requires, modPath+" "+v)
	}
	return requires
}

// appendRequires writes a `require (...)` block with the given pinned
// modules to an existing go.mod.
func appendRequires(modPath string, requires []string) error {
	f, err := os.OpenFile(modPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString("\nrequire (\n"); err != nil {
		return err
	}
	for _, r := range requires {
		if _, err := f.WriteString("\t" + r + "\n"); err != nil {
			return err
		}
	}
	if _, err := f.WriteString(")\n"); err != nil {
		return err
	}
	return nil
}

// parseTarget splits a "os/arch" string into separate GOOS / GOARCH values.
// An empty string means "use the host platform" — both returned values empty.
func parseTarget(spec string) (string, string, error) {
	if spec == "" {
		return "", "", nil
	}
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid --target %q: expected os/arch (e.g. windows/amd64)", spec)
	}
	return parts[0], parts[1], nil
}

func run(filename string) {
	result, err := compile(filename)
	if err != nil {
		printErrors(result)
		errorf("%v", err)
		os.Exit(1)
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w.String())
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "dbasic-*")
	if err != nil {
		errorf("creating temp directory: %v", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	// Write Go source file
	goFile := filepath.Join(tempDir, "main.go")
	err = os.WriteFile(goFile, []byte(result.GoCode), 0644)
	if err != nil {
		errorf("writing Go file: %v", err)
		os.Exit(1)
	}

	// Initialize Go module
	modInit := exec.Command("go", "mod", "init", "dbasic_program")
	modInit.Dir = tempDir
	modInit.Stdout = nil
	modInit.Stderr = nil
	if err := modInit.Run(); err != nil {
		errorf("initializing Go module: %v", err)
		os.Exit(1)
	}

	// Run go mod tidy to fetch dependencies
	modTidy := exec.Command("go", "mod", "tidy")
	modTidy.Dir = tempDir
	modTidy.Stdout = nil
	tidyFilter := &goToolFilter{out: os.Stderr}
	modTidy.Stderr = tidyFilter
	if err := modTidy.Run(); err != nil {
		tidyFilter.Flush()
		errorf("fetching dependencies: %v", err)
		os.Exit(1)
	}
	tidyFilter.Flush()

	// Run the program
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tempDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	runFilter := &goToolFilter{out: os.Stderr}
	cmd.Stderr = runFilter

	if err := cmd.Run(); err != nil {
		runFilter.Flush()
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		errorf("running program: %v", err)
		os.Exit(1)
	}
	runFilter.Flush()
}
