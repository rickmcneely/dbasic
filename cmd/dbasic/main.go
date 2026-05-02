package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	targetSpec  string
	offlineMode bool
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
	modTidy.Stderr = os.Stderr
	modTidy.Env = tidyEnv
	if err := modTidy.Run(); err != nil {
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
	cmd.Stderr = os.Stderr
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
		errorf("building executable: %v", err)
		os.Exit(1)
	}

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
