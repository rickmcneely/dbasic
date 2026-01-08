package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zditech/dbasic/pkg/analyzer"
	"github.com/zditech/dbasic/pkg/codegen"
	"github.com/zditech/dbasic/pkg/lexer"
	"github.com/zditech/dbasic/pkg/parser"
	"github.com/zditech/dbasic/pkg/preprocessor"
)

const version = "0.2.0"

var (
	debugMode   bool
	verboseMode bool
	outputFile  string
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
		check(os.Args[2])
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
	fmt.Println("  version               Print version")
	fmt.Println("  help                  Print this help")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -o <file>             Output file name (for build)")
	fmt.Println("  -debug                Include source line comments in output")
	fmt.Println("  -v                    Verbose output")
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

	// Determine output name
	if outputName == "" {
		base := filepath.Base(filename)
		outputName = strings.TrimSuffix(base, filepath.Ext(base))
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

	// Run go mod tidy to fetch dependencies
	modTidy := exec.Command("go", "mod", "tidy")
	modTidy.Dir = tempDir
	modTidy.Stdout = nil
	modTidy.Stderr = os.Stderr
	if err := modTidy.Run(); err != nil {
		errorf("fetching dependencies: %v", err)
		os.Exit(1)
	}

	// Get the current working directory for output
	cwd, err := os.Getwd()
	if err != nil {
		errorf("getting current directory: %v", err)
		os.Exit(1)
	}

	outputPath := filepath.Join(cwd, outputName)

	// Build executable
	infof("building %s", outputPath)
	cmd := exec.Command("go", "build", "-o", outputPath, ".")
	cmd.Dir = tempDir
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		errorf("building executable: %v", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Built: %s\n", outputPath)
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
	modTidy.Stderr = os.Stderr
	if err := modTidy.Run(); err != nil {
		errorf("fetching dependencies: %v", err)
		os.Exit(1)
	}

	// Run the program
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tempDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		errorf("running program: %v", err)
		os.Exit(1)
	}
}
