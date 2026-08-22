package codegen

import (
	"fmt"
	"strings"

	"github.com/zditech/dbasic/pkg/analyzer"
	"github.com/zditech/dbasic/pkg/parser"
)

// Generator generates Go code from a DBasic AST
type Generator struct {
	program         *parser.Program
	symbols         *analyzer.SymbolTable
	types           *analyzer.TypeRegistry
	currentScope    *analyzer.Scope
	output          strings.Builder
	indent          int
	imports         map[string]string // path -> alias (empty string if no alias)
	runtimeFuncs    map[string]bool   // Runtime functions that need to be embedded
	hasMain         bool
	labelCount      int
	debugMode       bool
	sourceFile      string
	currentFunc     string                       // Current function/sub name for error context
	currentOnErr    *parser.OnErrStatement       // Active ONERR handler for the current function (nil = none). Reset at every sub/function/method entry; updated by ONERR statements.
	statics         map[string]map[string]string // funcName → varName → uniquified package-level name
	optionBase      int                          // OPTION BASE setting (0 default; if 1, indices are emitted as `idx-1`)
	testMode        bool                         // When set, emit a test-runner main() instead of calling Main()
	testNames       []string                     // Names of TestXxx subs to invoke when testMode is true

	// Loop bookkeeping, so EXIT and CONTINUE reach the right statement.
	// loopLabels holds one entry per enclosing loop ("" when the loop was
	// emitted without a label); switchDepth counts SELECT CASE nesting
	// within the innermost loop; switchDepths saves that count per loop.
	loopLabels   []string
	switchDepth  int
	switchDepths []int
	loopSeq      int
}

// SetTestMode enables test-runner output: instead of calling Main(),
// the emitted main() iterates testNames, calling each one inside a
// recover() wrapper and printing PASS/FAIL.
func (g *Generator) SetTestMode(testNames []string) {
	g.testMode = true
	g.testNames = testNames
}

// New creates a new code generator
func New(program *parser.Program, symbols *analyzer.SymbolTable) *Generator {
	return &Generator{
		program:      program,
		symbols:      symbols,
		currentScope: symbols.GlobalScope,
		imports:      make(map[string]string),
		runtimeFuncs: make(map[string]bool),
		statics:      make(map[string]map[string]string),
	}
}

// SetDebugMode enables or disables debug mode
func (g *Generator) SetDebugMode(enabled bool) {
	g.debugMode = enabled
}

// SetSourceFile sets the source file name for debug comments
func (g *Generator) SetSourceFile(filename string) {
	g.sourceFile = filename
}

// SetTypeRegistry sets the type registry for custom types
func (g *Generator) SetTypeRegistry(types *analyzer.TypeRegistry) {
	g.types = types
}

// Generate generates Go source code
func (g *Generator) Generate() string {
	// Apply OPTION pragmas (OPTION BASE 0|1, etc.) before any other pass.
	for _, st := range g.program.Statements {
		if op, ok := st.(*parser.OptionStatement); ok && op.Kind == "BASE" {
			g.optionBase = op.Value
		}
	}

	// Test mode needs fmt + os in the runner; declare them up front so the
	// import block emitted by generateImports() includes them.
	if g.testMode {
		g.imports["fmt"] = ""
		g.imports["os"] = ""
	}

	// Collect imports from explicit IMPORT statements
	g.collectImports()

	// Pre-scan for additional required imports and runtime functions
	g.scanForRequiredImports()
	g.scanForRuntimeFunctions()

	// Check for Main sub
	mainSym := g.symbols.GlobalScope.Resolve("Main")
	g.hasMain = mainSym != nil

	// Generate package declaration
	g.writeLine("package main")
	g.writeLine("")

	// Generate imports
	g.generateImports()

	// Generate runtime helper functions
	g.generateRuntimeFunctions()

	// Generate type definitions (structs)
	g.generateTypeDefinitions()

	// Generate global variables
	g.generateGlobalVariables()

	// Hoist STATIC declarations from sub bodies into package-level vars,
	// and emit them before the function definitions.
	g.collectAndEmitStatics()

	// Generate functions, subs, and methods
	g.generateFunctions()

	// Generate main function. In test mode we ignore Main() and emit a
	// runner that calls each TestXxx in turn with a recover wrapper.
	if g.testMode {
		g.writeLine("")
		g.writeLine("func main() {")
		g.indent++
		g.writeLine("passed, failed := 0, 0")
		g.writeLine("tests := []struct{ name string; fn func() }{")
		g.indent++
		for _, n := range g.testNames {
			g.writeLine(fmt.Sprintf("{%q, %s},", n, n))
		}
		g.indent--
		g.writeLine("}")
		g.writeLine("for _, t := range tests {")
		g.indent++
		g.writeLine("if dbasic_runOneTest(t.name, t.fn) { passed++ } else { failed++ }")
		g.indent--
		g.writeLine("}")
		g.writeLine(`fmt.Printf("\n=== %d passed, %d failed ===\n", passed, failed)`)
		g.writeLine("if failed > 0 { os.Exit(1) }")
		g.indent--
		g.writeLine("}")
		g.writeLine("")
		g.writeLine("func dbasic_runOneTest(name string, fn func()) (ok bool) {")
		g.indent++
		g.writeLine("ok = true")
		g.writeLine("defer func() {")
		g.indent++
		g.writeLine("if r := recover(); r != nil {")
		g.indent++
		g.writeLine(`fmt.Printf("FAIL %s: %v\n", name, r)`)
		g.writeLine("ok = false")
		g.indent--
		g.writeLine("}")
		g.indent--
		g.writeLine("}()")
		g.writeLine(`fmt.Printf("RUN  %s\n", name)`)
		g.writeLine("fn()")
		g.writeLine(`fmt.Printf("PASS %s\n", name)`)
		g.writeLine("return")
		g.indent--
		g.writeLine("}")
	} else if g.hasMain {
		g.writeLine("")
		g.writeLine("func main() {")
		g.indent++
		g.writeLine("Main()")
		g.indent--
		g.writeLine("}")
	}

	return g.output.String()
}

// scanForRequiredImports pre-scans the AST to find required imports
func (g *Generator) scanForRequiredImports() {
	for _, stmt := range g.program.Statements {
		g.scanStatementForImports(stmt)
	}
}

func (g *Generator) scanStatementForImports(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.SubStatement:
		g.scanBlockForImports(s.Body)
	case *parser.FunctionStatement:
		g.scanBlockForImports(s.Body)
	case *parser.MethodStatement:
		g.scanBlockForImports(s.Body)
	case *parser.InputStatement:
		g.imports["bufio"] = ""
		g.imports["os"] = ""
		g.imports["strings"] = ""
	case *parser.PrintStatement:
		g.imports["fmt"] = ""
	}
}

func (g *Generator) scanBlockForImports(block *parser.BlockStatement) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *parser.InputStatement:
			g.imports["bufio"] = ""
			g.imports["os"] = ""
			g.imports["strings"] = ""
		case *parser.PrintStatement:
			g.imports["fmt"] = ""
		case *parser.IfStatement:
			g.scanBlockForImports(s.Consequence)
			for _, elseif := range s.ElseIfs {
				g.scanBlockForImports(elseif.Consequence)
			}
			g.scanBlockForImports(s.Alternative)
		case *parser.ForStatement:
			g.scanBlockForImports(s.Body)
		case *parser.WhileStatement:
			g.scanBlockForImports(s.Body)
		case *parser.DoLoopStatement:
			g.scanBlockForImports(s.Body)
		case *parser.SelectStatement:
			for _, c := range s.Cases {
				g.scanBlockForImports(c.Body)
			}
			g.scanBlockForImports(s.Default)
		}
		// Check for math.Pow usage in expressions
		g.scanExpressionForImports(stmt)
	}
}

func (g *Generator) scanExpressionForImports(stmt parser.Statement) {
	// Look for exponentiation which requires math package
	switch s := stmt.(type) {
	case *parser.AssignmentStatement:
		if g.exprNeedsMath(s.Value) {
			g.imports["math"] = ""
		}
	case *parser.ExpressionStatement:
		if s.Expression != nil && g.exprNeedsMath(s.Expression) {
			g.imports["math"] = ""
		}
	case *parser.DimStatement:
		if s.Value != nil && g.exprNeedsMath(s.Value) {
			g.imports["math"] = ""
		}
	case *parser.PrintStatement:
		for _, v := range s.Values {
			if g.exprNeedsMath(v) {
				g.imports["math"] = ""
				break
			}
		}
	case *parser.ReturnStatement:
		for _, v := range s.Values {
			if g.exprNeedsMath(v) {
				g.imports["math"] = ""
				break
			}
		}
	}
}

func (g *Generator) exprNeedsMath(expr parser.Expression) bool {
	switch e := expr.(type) {
	case *parser.InfixExpression:
		if e.Operator == "^" {
			return true
		}
		return g.exprNeedsMath(e.Left) || g.exprNeedsMath(e.Right)
	case *parser.PrefixExpression:
		return g.exprNeedsMath(e.Right)
	case *parser.CallExpression:
		for _, arg := range e.Arguments {
			if g.exprNeedsMath(arg) {
				return true
			}
		}
	case *parser.IndexExpression:
		result := g.exprNeedsMath(e.Left)
		if e.Index != nil {
			result = result || g.exprNeedsMath(e.Index)
		}
		if e.End != nil {
			result = result || g.exprNeedsMath(e.End)
		}
		return result
	}
	return false
}

func (g *Generator) collectImports() {
	// fmt is added conditionally: by scanBlockForImports for PRINT,
	// and at use-sites for Printf/Sprintf/Str and runtime helpers
	// that emit fmt.* calls. Adding it unconditionally caused
	// "fmt imported and not used" errors in programs that don't print.

	// Add user imports with their aliases
	for _, imp := range g.symbols.AllImports() {
		g.imports[imp.Path] = imp.Alias
	}
}

// runtimeFuncDefs contains the Go source for runtime functions
var runtimeFuncDefs = map[string]string{
	"Int": `// Int converts to int.
func Int(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	}
	// Named types (e.g. ` + "`vt10x.Color`" + ` whose underlying type is uint32)
	// don't match the bare type cases above — fall back to reflection.
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return int(rv.Float())
	}
	return 0
}`,
	"Lng": `// Lng converts to int64 (LONG)
func Lng(val interface{}) int64 {
	switch v := val.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}`,
	"Sng": `// Sng converts to float32 (SINGLE)
func Sng(val interface{}) float32 {
	switch v := val.(type) {
	case int:
		return float32(v)
	case int32:
		return float32(v)
	case int64:
		return float32(v)
	case float32:
		return v
	case float64:
		return float32(v)
	default:
		return 0
	}
}`,
	"Dbl": `// Dbl converts to float64 (DOUBLE)
func Dbl(val interface{}) float64 {
	switch v := val.(type) {
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}`,
	"Sleep": `// Sleep pauses execution for specified milliseconds
func Sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}`,
	"Sqr": `// Sqr returns the square root
func Sqr(val float64) float64 {
	return math.Sqrt(val)
}`,
	"Abs": `// Abs returns the absolute value
func Abs(val float64) float64 {
	return math.Abs(val)
}`,
	"Sin": `// Sin returns the sine
func Sin(val float64) float64 {
	return math.Sin(val)
}`,
	"Cos": `// Cos returns the cosine
func Cos(val float64) float64 {
	return math.Cos(val)
}`,
	"Len": `// Len returns the length of a string
func Len(s string) int {
	return len(s)
}`,
	"Left": `// Left returns the leftmost n characters
func Left(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if n >= len(s) {
		return s
	}
	return s[:n]
}`,
	"Right": `// Right returns the rightmost n characters
func Right(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if n >= len(s) {
		return s
	}
	return s[len(s)-n:]
}`,
	"Mid": `// Mid returns a substring starting at position start with length ln
func Mid(s string, start, ln int) string {
	if start < 1 {
		start = 1
	}
	startIdx := start - 1
	if startIdx >= len(s) {
		return ""
	}
	endIdx := startIdx + ln
	if endIdx > len(s) {
		endIdx = len(s)
	}
	return s[startIdx:endIdx]
}`,
	"Str": `// Str converts a number to string
func Str(val interface{}) string {
	return fmt.Sprintf("%v", val)
}`,
	"Val": `// Val converts a string to float64
func Val(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}`,
	"UCase": `// UCase converts to uppercase
func UCase(s string) string {
	return strings.ToUpper(s)
}`,
	"LCase": `// LCase converts to lowercase
func LCase(s string) string {
	return strings.ToLower(s)
}`,
	"Trim": `// Trim removes leading and trailing whitespace
func Trim(s string) string {
	return strings.TrimSpace(s)
}`,
	"Rnd": `// Rnd returns a random float64 between 0 and 1
func Rnd() float64 {
	return rand.Float64()
}`,
	"RndInt": `// RndInt returns a random integer between 0 and max-1
func RndInt(max int) int {
	return rand.Intn(max)
}`,
	"Instr": `// Instr finds the position of substring in string (1-based)
func Instr(s, substr string) int {
	idx := strings.Index(s, substr)
	if idx == -1 {
		return 0
	}
	return idx + 1
}`,
	"Chr": `// Chr returns the character for an ASCII code
func Chr(code int) string {
	return string(rune(code))
}`,
	"Asc": `// Asc returns the ASCII code of the first character
func Asc(s string) int {
	if len(s) == 0 {
		return 0
	}
	return int(s[0])
}`,
	"FileExists": `// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}`,
	"ReadFile": `// ReadFile reads entire file contents
func ReadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}`,
	"WriteFile": `// WriteFile writes string to file
func WriteFile(path, content string) {
	os.WriteFile(path, []byte(content), 0644)
}`,
	"Shell": `// Shell executes a command via the system shell and returns output, exit code, and error
func Shell(command string) (string, int, string) {
	cmd := exec.Command("/bin/sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, err.Error()
		}
	}
	if stderr.Len() > 0 {
		return stdout.String(), exitCode, stderr.String()
	}
	return stdout.String(), exitCode, ""
}`,
	"ShellExec": `// ShellExec executes a program with space-separated arguments and returns output, exit code, and error
func ShellExec(program string, args string) (string, int, string) {
	var argList []string
	if args != "" {
		argList = strings.Fields(args)
	}
	cmd := exec.Command(program, argList...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, err.Error()
		}
	}
	if stderr.Len() > 0 {
		return stdout.String(), exitCode, stderr.String()
	}
	return stdout.String(), exitCode, ""
}`,
	"ShellStart": `// ShellStart starts a command in the background and returns pid and error
func ShellStart(command string) (int, string) {
	cmd := exec.Command("/bin/sh", "-c", command)
	err := cmd.Start()
	if err != nil {
		return 0, err.Error()
	}
	return cmd.Process.Pid, ""
}`,
	"JSONParse": `// JSONParse parses a JSON string into a map
func JSONParse(s string) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal([]byte(s), &result)
	return result
}`,
	"JSONStringify": `// JSONStringify converts a map to JSON string
func JSONStringify(data map[string]interface{}) string {
	b, _ := json.Marshal(data)
	return string(b)
}`,
	"JSONPretty": `// JSONPretty converts a map to pretty-printed JSON string
func JSONPretty(data map[string]interface{}) string {
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}`,
	"JSONGet": `// JSONGet retrieves a value from a JSON map by path
func JSONGet(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return nil
		}
	}
	return current
}`,
	"JSONSet": `// JSONSet sets a value in a JSON map by path
func JSONSet(data map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := data
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if _, ok := current[part]; !ok {
			current[part] = make(map[string]interface{})
		}
		current = current[part].(map[string]interface{})
	}
	current[parts[len(parts)-1]] = value
}`,
	"StructToJSON": `// StructToJSON converts a struct to a JSON map
func StructToJSON(v interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return result
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := field.Tag.Get("json")
		if name == "" {
			name = field.Name
		}
		result[name] = fieldVal.Interface()
	}
	return result
}`,
	"JSONToStruct": `// JSONToStruct populates a struct from a JSON map
func JSONToStruct(data map[string]interface{}, v interface{}) interface{} {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return v
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return v
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		if field.PkgPath != "" || !fieldVal.CanSet() {
			continue
		}
		name := field.Tag.Get("json")
		if name == "" {
			name = field.Name
		}
		jsonVal, ok := data[name]
		if !ok {
			continue
		}
		if jsonVal != nil {
			setJSONFieldValue(fieldVal, jsonVal)
		}
	}
	return v
}

func setJSONFieldValue(field reflect.Value, value interface{}) {
	if value == nil {
		return
	}
	switch field.Kind() {
	case reflect.String:
		if s, ok := value.(string); ok {
			field.SetString(s)
		}
	case reflect.Int, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case float64:
			field.SetInt(int64(v))
		case int:
			field.SetInt(int64(v))
		case int64:
			field.SetInt(v)
		}
	case reflect.Float32, reflect.Float64:
		if f, ok := value.(float64); ok {
			field.SetFloat(f)
		}
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
		}
	}
}`,
	"NewErrorAtFunc": `// DBasicError represents a runtime error with source location
type DBasicError struct {
	Message  string
	File     string
	Line     int
	Function string
	Wrapped  error
}

func (e *DBasicError) Error() string {
	var result string
	if e.File != "" && e.Line > 0 {
		if e.Function != "" {
			result = fmt.Sprintf("%s:%d (%s): %s", e.File, e.Line, e.Function, e.Message)
		} else {
			result = fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
		}
	} else {
		result = e.Message
	}
	if e.Wrapped != nil {
		if dbErr, ok := e.Wrapped.(*DBasicError); ok {
			result += "\n  caused by: " + dbErr.Error()
		} else {
			result += "\n  caused by: " + e.Wrapped.Error()
		}
	}
	return result
}

func (e *DBasicError) Unwrap() error {
	return e.Wrapped
}

// NewErrorAtFunc creates a new error with source location and function name
func NewErrorAtFunc(file string, line int, function string, message string) error {
	return &DBasicError{
		Message:  message,
		File:     file,
		Line:     line,
		Function: function,
	}
}`,
	"ErrorfFunc": `// ErrorfFunc creates a formatted error with source location and function name
func ErrorfFunc(file string, line int, function string, format string, args ...interface{}) error {
	return &DBasicError{
		Message:  fmt.Sprintf(format, args...),
		File:     file,
		Line:     line,
		Function: function,
	}
}`,
	"WrapError": `// WrapError wraps an existing error with additional context and location
func WrapError(err error, file string, line int, function string, message string) error {
	if err == nil {
		return nil
	}
	return &DBasicError{
		Message:  message,
		File:     file,
		Line:     line,
		Function: function,
		Wrapped:  err,
	}
}`,
	"LineInput": `// LineInput prints prompt (if any) and reads a full line from stdin,
// including embedded spaces. Returns the line without the trailing newline.
var _lineInputScanner *bufio.Scanner

func LineInput(prompt string) string {
	if prompt != "" {
		fmt.Print(prompt)
	}
	if _lineInputScanner == nil {
		_lineInputScanner = bufio.NewScanner(os.Stdin)
		_lineInputScanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	}
	if !_lineInputScanner.Scan() {
		return ""
	}
	return _lineInputScanner.Text()
}`,
	"Encode": `// Encode converts a STRING to BYTES (UTF-8 encoding).
func Encode(s string) []byte {
	return []byte(s)
}`,
	"Decode": `// Decode converts BYTES to a STRING (UTF-8 decoding).
func Decode(b []byte) string {
	return string(b)
}`,
	"MakeBytes": `// MakeBytes allocates a zero-filled byte slice of the given size.
func MakeBytes(size int) []byte {
	return make([]byte, size)
}`,
	"LenBytes": `// LenBytes returns the length of a BYTES slice.
func LenBytes(b []byte) int {
	return len(b)
}`,
	"Uint8": `// Uint8 casts any integer/float value to uint8 (delegates to Int).
func Uint8(val interface{}) uint8  { return uint8(Int(val)) }`,
	"Uint16": `// Uint16 casts any integer/float value to uint16 (delegates to Int).
func Uint16(val interface{}) uint16 { return uint16(Int(val)) }`,
	"Uint32": `// Uint32 casts any integer/float value to uint32 (delegates to Int).
func Uint32(val interface{}) uint32 { return uint32(Int(val)) }`,
	"Uint64": `// Uint64 casts any integer/float value to uint64 (delegates to Int).
func Uint64(val interface{}) uint64 { return uint64(Int(val)) }`,
	"BitAnd": `// BitAnd returns a & b. Both args flow through Int so named
// external types (vt10x.Color, etc.) work.
func BitAnd(a, b interface{}) int { return Int(a) & Int(b) }`,
	"BitOr": `// BitOr returns a | b.
func BitOr(a, b interface{}) int { return Int(a) | Int(b) }`,
	"BitXor": `// BitXor returns a ^ b.
func BitXor(a, b interface{}) int { return Int(a) ^ Int(b) }`,
	"BitNot": `// BitNot returns ^a.
func BitNot(a interface{}) int { return ^Int(a) }`,
	"BitShl": `// BitShl returns a << b.
func BitShl(a, b interface{}) int { return Int(a) << uint(Int(b)) }`,
	"BitShr": `// BitShr returns a >> b.
func BitShr(a, b interface{}) int { return Int(a) >> uint(Int(b)) }`,
}

// runtimeFuncImports maps runtime functions to required imports
var runtimeFuncImports = map[string][]string{
	"Sleep":          {"time"},
	"Sqr":            {"math"},
	"Abs":            {"math"},
	"Sin":            {"math"},
	"Cos":            {"math"},
	"Val":            {"strconv", "strings"},
	"UCase":          {"strings"},
	"LCase":          {"strings"},
	"Trim":           {"strings"},
	"Rnd":            {"math/rand"},
	"RndInt":         {"math/rand"},
	"Instr":          {"strings"},
	"FileExists":     {"os"},
	"ReadFile":       {"os"},
	"WriteFile":      {"os"},
	"Shell":          {"os/exec", "bytes"},
	"ShellExec":      {"os/exec", "bytes", "strings"},
	"ShellStart":     {"os/exec"},
	"JSONParse":      {"encoding/json"},
	"JSONStringify":  {"encoding/json"},
	"JSONPretty":     {"encoding/json"},
	"JSONGet":        {"strings"},
	"JSONSet":        {"strings"},
	"StructToJSON":   {"reflect"},
	"JSONToStruct":   {"reflect"},
	"NewErrorAtFunc": {"fmt"},
	"ErrorfFunc":     {"fmt"},
	"WrapError":      {"fmt"},
	"LineInput":      {"bufio", "fmt", "os"},
	"Int":            {"reflect"},
	"Uint8":          {"reflect"},
	"Uint16":         {"reflect"},
	"Uint32":         {"reflect"},
	"Uint64":         {"reflect"},
	"BitAnd":         {"reflect"},
	"BitOr":          {"reflect"},
	"BitXor":         {"reflect"},
	"BitNot":         {"reflect"},
	"BitShl":         {"reflect"},
	"BitShr":         {"reflect"},
}

// scanForRuntimeFunctions scans the AST for calls to runtime functions
func (g *Generator) scanForRuntimeFunctions() {
	for _, stmt := range g.program.Statements {
		g.scanStatementForRuntimeFuncs(stmt)
	}
}

func (g *Generator) scanStatementForRuntimeFuncs(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.SubStatement:
		g.scanBlockForRuntimeFuncs(s.Body)
	case *parser.FunctionStatement:
		g.scanBlockForRuntimeFuncs(s.Body)
	case *parser.MethodStatement:
		g.scanBlockForRuntimeFuncs(s.Body)
	}
}

func (g *Generator) scanBlockForRuntimeFuncs(block *parser.BlockStatement) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		g.scanStmtExprForRuntimeFuncs(stmt)
		// Recurse into nested blocks
		switch s := stmt.(type) {
		case *parser.IfStatement:
			g.scanBlockForRuntimeFuncs(s.Consequence)
			for _, elseif := range s.ElseIfs {
				g.scanBlockForRuntimeFuncs(elseif.Consequence)
			}
			g.scanBlockForRuntimeFuncs(s.Alternative)
		case *parser.ForStatement:
			g.scanBlockForRuntimeFuncs(s.Body)
		case *parser.WhileStatement:
			g.scanBlockForRuntimeFuncs(s.Body)
		case *parser.DoLoopStatement:
			g.scanBlockForRuntimeFuncs(s.Body)
		case *parser.SelectStatement:
			for _, c := range s.Cases {
				g.scanBlockForRuntimeFuncs(c.Body)
			}
			g.scanBlockForRuntimeFuncs(s.Default)
		}
	}
}

func (g *Generator) scanStmtExprForRuntimeFuncs(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.AssignmentStatement:
		g.scanExprForRuntimeFuncs(s.Value)
		g.scanExprForRuntimeFuncs(s.Left)
	case *parser.MultiAssignmentStatement:
		g.scanExprForRuntimeFuncs(s.Value)
		for _, target := range s.Targets {
			g.scanExprForRuntimeFuncs(target)
		}
	case *parser.DimStatement:
		if s.Value != nil {
			g.scanExprForRuntimeFuncs(s.Value)
		}
	case *parser.LineInputStatement:
		g.runtimeFuncs["LineInput"] = true
		for _, imp := range runtimeFuncImports["LineInput"] {
			g.imports[imp] = ""
		}
		if s.Prompt != nil {
			g.scanExprForRuntimeFuncs(s.Prompt)
		}
	case *parser.PrintStatement:
		for _, v := range s.Values {
			g.scanExprForRuntimeFuncs(v)
		}
	case *parser.ExpressionStatement:
		if s.Expression != nil {
			g.scanExprForRuntimeFuncs(s.Expression)
		}
	case *parser.ReturnStatement:
		for _, v := range s.Values {
			g.scanExprForRuntimeFuncs(v)
		}
	case *parser.IfStatement:
		g.scanExprForRuntimeFuncs(s.Condition)
	case *parser.WhileStatement:
		g.scanExprForRuntimeFuncs(s.Condition)
	case *parser.ForStatement:
		g.scanExprForRuntimeFuncs(s.Start)
		g.scanExprForRuntimeFuncs(s.End)
		if s.Step != nil {
			g.scanExprForRuntimeFuncs(s.Step)
		}
	}
}

// builtinFuncImports maps builtin function names to their required imports
var builtinFuncImports = map[string][]string{
	"Printf":  {"fmt"},
	"Sprintf": {"fmt"},
	// Note: NewError, Errorf, WrapError use embedded runtime functions, not imports
}

func (g *Generator) scanExprForRuntimeFuncs(expr parser.Expression) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *parser.CallExpression:
		// Check if this is a runtime function call
		if ident, ok := e.Function.(*parser.Identifier); ok {
			if _, isRuntime := runtimeFuncDefs[ident.Value]; isRuntime {
				g.runtimeFuncs[ident.Value] = true
				// Add required imports for this runtime function
				if imports, ok := runtimeFuncImports[ident.Value]; ok {
					for _, imp := range imports {
						g.imports[imp] = ""
					}
				}
				// Pull in transitive runtime dependencies. Uint* + Bit*
				// helpers call Int() internally, so Int's body has to
				// be in the emitted runtime block too even if the user
				// didn't call Int directly.
				switch ident.Value {
				case "Uint8", "Uint16", "Uint32", "Uint64",
					"BitAnd", "BitOr", "BitXor", "BitNot",
					"BitShl", "BitShr":
					g.runtimeFuncs["Int"] = true
					for _, imp := range runtimeFuncImports["Int"] {
						g.imports[imp] = ""
					}
				}
			}
			// Check if this is a builtin function that needs imports
			if imports, ok := builtinFuncImports[ident.Value]; ok {
				for _, imp := range imports {
					g.imports[imp] = ""
				}
			}
			// Check if this is an error handling function that needs runtime embedding
			switch strings.ToUpper(ident.Value) {
			case "NEWERROR":
				g.runtimeFuncs["NewErrorAtFunc"] = true
			case "ERRORF":
				g.runtimeFuncs["ErrorfFunc"] = true
				g.runtimeFuncs["NewErrorAtFunc"] = true // ErrorfFunc depends on DBasicError type
			case "WRAPERROR":
				g.runtimeFuncs["WrapError"] = true
				g.runtimeFuncs["NewErrorAtFunc"] = true // WrapError depends on DBasicError type
			}
		}
		// Scan arguments
		for _, arg := range e.Arguments {
			g.scanExprForRuntimeFuncs(arg)
		}
	case *parser.InfixExpression:
		g.scanExprForRuntimeFuncs(e.Left)
		g.scanExprForRuntimeFuncs(e.Right)
	case *parser.PrefixExpression:
		g.scanExprForRuntimeFuncs(e.Right)
	case *parser.IndexExpression:
		g.scanExprForRuntimeFuncs(e.Left)
		if e.Index != nil {
			g.scanExprForRuntimeFuncs(e.Index)
		}
		if e.End != nil {
			g.scanExprForRuntimeFuncs(e.End)
		}
	case *parser.MemberExpression:
		g.scanExprForRuntimeFuncs(e.Object)
	case *parser.DereferenceExpression:
		g.scanExprForRuntimeFuncs(e.Value)
	case *parser.AddressOfExpression:
		g.scanExprForRuntimeFuncs(e.Value)
	}
}

func (g *Generator) generateRuntimeFunctions() {
	if len(g.runtimeFuncs) == 0 {
		return
	}

	g.writeLine("// Runtime helper functions")
	for funcName := range g.runtimeFuncs {
		if def, ok := runtimeFuncDefs[funcName]; ok {
			g.writeLine("")
			// Write each line of the function definition
			for _, line := range strings.Split(def, "\n") {
				g.output.WriteString(line)
				g.output.WriteString("\n")
			}
		}
	}
	g.writeLine("")
}

func (g *Generator) generateImports() {
	if len(g.imports) == 0 {
		return
	}

	g.writeLine("import (")
	g.indent++
	for path, alias := range g.imports {
		if alias != "" {
			g.writeLine(fmt.Sprintf(`%s "%s"`, alias, path))
		} else {
			g.writeLine(fmt.Sprintf(`"%s"`, path))
		}
	}
	g.indent--
	g.writeLine(")")
	g.writeLine("")
}

func (g *Generator) generateTypeDefinitions() {
	hasTypes := false
	for _, stmt := range g.program.Statements {
		if ts, ok := stmt.(*parser.TypeStatement); ok {
			if !hasTypes {
				hasTypes = true
			}
			g.generateTypeStatement(ts)
		}
	}
	if hasTypes {
		g.writeLine("")
	}
}

func (g *Generator) generateTypeStatement(stmt *parser.TypeStatement) {
	typeName := g.toGoIdent(stmt.Name.Value)
	g.writeLine(fmt.Sprintf("type %s struct {", typeName))
	g.indent++

	// Generate embedded types first (anonymous embedding)
	for _, embed := range stmt.Embedded {
		g.writeLine(embed.TypeName)
	}

	// Generate named fields
	for _, field := range stmt.Fields {
		fieldName := g.toGoIdent(field.Name.Value)
		fieldType := g.typeSpecToGo(field.Type)
		g.writeLine(fmt.Sprintf("%s %s", fieldName, fieldType))
	}
	g.indent--
	g.writeLine("}")
	g.writeLine("")
}

func (g *Generator) generateGlobalVariables() {
	hasGlobals := false

	for _, stmt := range g.program.Statements {
		switch s := stmt.(type) {
		case *parser.DimStatement:
			if !hasGlobals {
				g.writeLine("var (")
				g.indent++
				hasGlobals = true
			}
			g.generateDimStatement(s)
		case *parser.ConstStatement:
			if hasGlobals {
				g.indent--
				g.writeLine(")")
				hasGlobals = false
			}
			g.generateConstStatement(s)
		}
	}

	if hasGlobals {
		g.indent--
		g.writeLine(")")
		g.writeLine("")
	}
}

func (g *Generator) generateFunctions() {
	for _, stmt := range g.program.Statements {
		switch s := stmt.(type) {
		case *parser.SubStatement:
			g.generateSubStatement(s)
		case *parser.FunctionStatement:
			g.generateFunctionStatement(s)
		case *parser.MethodStatement:
			g.generateMethodStatement(s)
		}
	}
}

func (g *Generator) generateDimStatement(stmt *parser.DimStatement) {
	varName := g.toGoIdent(stmt.Name.Value)
	varType := g.typeSpecToGo(stmt.Type)

	if stmt.Value != nil {
		g.writeLine(fmt.Sprintf("%s %s = %s", varName, varType, g.exprToGo(stmt.Value)))
	} else if stmt.ArraySize != nil {
		g.writeLine(fmt.Sprintf("%s = make([]%s, %s)", varName, varType, g.exprToGo(stmt.ArraySize)))
	} else if strings.HasPrefix(varType, "map[") {
		// Auto-initialize package-level maps to an empty map so writes don't
		// panic on a nil map — matches the auto-init locals already get
		// (DIM ... AS MAP OF K TO V is documented as ready to write).
		g.writeLine(fmt.Sprintf("%s = %s{}", varName, varType))
	} else {
		g.writeLine(fmt.Sprintf("%s %s", varName, varType))
	}
}

func (g *Generator) generateConstStatement(stmt *parser.ConstStatement) {
	constName := g.toGoIdent(stmt.Name.Value)
	g.writeLine(fmt.Sprintf("const %s = %s", constName, g.exprToGo(stmt.Value)))
}

func (g *Generator) generateSubStatement(stmt *parser.SubStatement) {
	g.writeLine("")
	funcName := g.toGoIdent(stmt.Name.Value)
	params := g.generateParams(stmt.Params)
	g.writeLine(fmt.Sprintf("func %s(%s) {", funcName, params))
	g.indent++
	// Track local variables for this sub
	oldScope := g.currentScope
	oldFunc := g.currentFunc
	oldOnErr := g.currentOnErr
	g.currentScope = analyzer.NewScope(stmt.Name.Value, g.symbols.GlobalScope)
	g.currentFunc = stmt.Name.Value
	g.currentOnErr = nil
	// Add parameters to local scope
	for _, p := range stmt.Params {
		paramType := g.typeFromTypeSpec(p.Type)
		g.currentScope.Define(&analyzer.Symbol{
			Name: p.Name.Value,
			Kind: analyzer.SymParameter,
			Type: paramType,
		})
	}
	if g.containsOnErrGoto(stmt.Body) {
		g.hoistDimsForOnErrGoto(stmt.Body)
	}
	g.generateBlockStatement(stmt.Body)
	g.currentScope = oldScope
	g.currentFunc = oldFunc
	g.currentOnErr = oldOnErr
	g.indent--
	g.writeLine("}")
}

func (g *Generator) generateFunctionStatement(stmt *parser.FunctionStatement) {
	g.writeLine("")
	funcName := g.toGoIdent(stmt.Name.Value)
	params := g.generateParams(stmt.Params)
	returns := g.generateReturnTypes(stmt.ReturnTypes)
	g.writeLine(fmt.Sprintf("func %s(%s) %s {", funcName, params, returns))
	g.indent++
	// Track local variables for this function
	oldScope := g.currentScope
	oldFunc := g.currentFunc
	oldOnErr := g.currentOnErr
	g.currentScope = analyzer.NewScope(stmt.Name.Value, g.symbols.GlobalScope)
	g.currentFunc = stmt.Name.Value
	g.currentOnErr = nil
	// Add parameters to local scope
	for _, p := range stmt.Params {
		paramType := g.typeFromTypeSpec(p.Type)
		g.currentScope.Define(&analyzer.Symbol{
			Name: p.Name.Value,
			Kind: analyzer.SymParameter,
			Type: paramType,
		})
	}
	if g.containsOnErrGoto(stmt.Body) {
		g.hoistDimsForOnErrGoto(stmt.Body)
	}
	g.generateBlockStatement(stmt.Body)
	g.currentScope = oldScope
	g.currentFunc = oldFunc
	g.currentOnErr = oldOnErr
	g.indent--
	g.writeLine("}")
}

func (g *Generator) generateMethodStatement(stmt *parser.MethodStatement) {
	g.writeLine("")

	// Generate receiver
	receiverName := g.toGoIdent(stmt.ReceiverName.Value)
	receiverType := g.typeSpecToGo(stmt.ReceiverType)

	// Generate method name
	methodName := g.toGoIdent(stmt.Name.Value)

	// Generate parameters
	params := g.generateParams(stmt.Params)

	// Generate return types
	returns := g.generateReturnTypes(stmt.ReturnTypes)

	g.writeLine(fmt.Sprintf("func (%s %s) %s(%s) %s {", receiverName, receiverType, methodName, params, returns))
	g.indent++

	// Track local variables for this method
	oldScope := g.currentScope
	oldFunc := g.currentFunc
	oldOnErr := g.currentOnErr
	g.currentScope = analyzer.NewScope(stmt.Name.Value, g.symbols.GlobalScope)
	g.currentFunc = stmt.Name.Value
	g.currentOnErr = nil

	// Add receiver to local scope
	recvType := g.typeFromTypeSpec(stmt.ReceiverType)
	g.currentScope.Define(&analyzer.Symbol{
		Name: stmt.ReceiverName.Value,
		Kind: analyzer.SymParameter,
		Type: recvType,
	})

	// Add parameters to local scope
	for _, p := range stmt.Params {
		paramType := g.typeFromTypeSpec(p.Type)
		g.currentScope.Define(&analyzer.Symbol{
			Name: p.Name.Value,
			Kind: analyzer.SymParameter,
			Type: paramType,
		})
	}

	if g.containsOnErrGoto(stmt.Body) {
		g.hoistDimsForOnErrGoto(stmt.Body)
	}
	g.generateBlockStatement(stmt.Body)
	g.currentScope = oldScope
	g.currentFunc = oldFunc
	g.currentOnErr = oldOnErr
	g.indent--
	g.writeLine("}")
}

func (g *Generator) generateParams(params []*parser.Parameter) string {
	var parts []string
	for _, p := range params {
		paramName := g.toGoIdent(p.Name.Value)
		paramType := g.typeSpecToGo(p.Type)
		if p.ByRef {
			paramType = "*" + paramType
		}
		parts = append(parts, fmt.Sprintf("%s %s", paramName, paramType))
	}
	return strings.Join(parts, ", ")
}

func (g *Generator) generateReturnTypes(types []*parser.TypeSpec) string {
	if len(types) == 0 {
		return ""
	}
	if len(types) == 1 {
		return g.typeSpecToGo(types[0])
	}

	var parts []string
	for _, t := range types {
		parts = append(parts, g.typeSpecToGo(t))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func (g *Generator) generateBlockStatement(block *parser.BlockStatement) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		g.generateStatement(stmt)
	}
}

// stmtLine returns the source line number associated with a statement,
// or 0 if it cannot be determined.
func stmtLine(stmt parser.Statement) int {
	switch s := stmt.(type) {
	case *parser.DimStatement:
		return s.Token.Line
	case *parser.LetStatement:
		return s.Token.Line
	case *parser.ConstStatement:
		return s.Token.Line
	case *parser.AssignmentStatement:
		return s.Token.Line
	case *parser.MultiAssignmentStatement:
		return s.Token.Line
	case *parser.PrintStatement:
		return s.Token.Line
	case *parser.InputStatement:
		return s.Token.Line
	case *parser.IfStatement:
		return s.Token.Line
	case *parser.ForStatement:
		return s.Token.Line
	case *parser.WhileStatement:
		return s.Token.Line
	case *parser.DoLoopStatement:
		return s.Token.Line
	case *parser.SelectStatement:
		return s.Token.Line
	case *parser.ReturnStatement:
		return s.Token.Line
	case *parser.ExitStatement:
		return s.Token.Line
	case *parser.ContinueStatement:
		return s.Token.Line
	case *parser.GotoStatement:
		return s.Token.Line
	case *parser.LabelStatement:
		return s.Token.Line
	case *parser.SpawnStatement:
		return s.Token.Line
	case *parser.DeferStatement:
		return s.Token.Line
	case *parser.SendStatement:
		return s.Token.Line
	case *parser.ReceiveStatement:
		return s.Token.Line
	case *parser.ExpressionStatement:
		return s.Token.Line
	case *parser.SubStatement:
		return s.Token.Line
	case *parser.FunctionStatement:
		return s.Token.Line
	case *parser.MethodStatement:
		return s.Token.Line
	case *parser.TypeStatement:
		return s.Token.Line
	case *parser.ImportStatement:
		return s.Token.Line
	}
	return 0
}

func (g *Generator) generateStatement(stmt parser.Statement) {
	g.emitLineDirective(stmtLine(stmt))
	switch s := stmt.(type) {
	case *parser.DimStatement:
		g.generateLocalDim(s)
	case *parser.ReDimStatement:
		g.generateReDim(s)
	case *parser.LetStatement:
		g.generateLet(s)
	case *parser.AssignmentStatement:
		g.generateAssignment(s)
	case *parser.MultiAssignmentStatement:
		g.generateMultiAssignment(s)
	case *parser.PrintStatement:
		g.generatePrint(s)
	case *parser.InputStatement:
		g.generateInput(s)
	case *parser.LineInputStatement:
		g.generateLineInput(s)
	case *parser.WithStatement:
		g.generateWith(s)
	case *parser.IfStatement:
		g.generateIf(s)
	case *parser.ForStatement:
		g.generateFor(s)
	case *parser.WhileStatement:
		g.generateWhile(s)
	case *parser.DoLoopStatement:
		g.generateDoLoop(s)
	case *parser.SelectStatement:
		g.generateSelect(s)
	case *parser.ReturnStatement:
		g.generateReturn(s)
	case *parser.ExitStatement:
		g.generateExit(s)
	case *parser.ContinueStatement:
		g.generateContinue(s)
	case *parser.GotoStatement:
		g.generateGoto(s)
	case *parser.LabelStatement:
		g.generateLabel(s)
	case *parser.SpawnStatement:
		g.generateSpawn(s)
	case *parser.DeferStatement:
		g.generateDefer(s)
	case *parser.SendStatement:
		g.generateSend(s)
	case *parser.ReceiveStatement:
		g.generateReceive(s)
	case *parser.OnErrStatement:
		g.generateOnErr(s)
	case *parser.ExpressionStatement:
		if s.Expression != nil {
			// Special-case: APPEND(slice, ...) used as a statement is a
			// mutation form. Rewrite to `slice = append(slice, ...)` so
			// users don't have to write `slice = APPEND(slice, ...)`.
			if g.tryEmitAppendMutation(s.Expression) {
				return
			}
			g.writeLine(g.exprToGo(s.Expression))
		}
	}
}

// tryEmitAppendMutation rewrites a top-level APPEND call statement into
// `slice = append(slice, args...)`. Returns true if it emitted code.
func (g *Generator) tryEmitAppendMutation(expr parser.Expression) bool {
	call, ok := expr.(*parser.CallExpression)
	if !ok || call.Function == nil || len(call.Arguments) < 2 {
		return false
	}
	ident, ok := call.Function.(*parser.Identifier)
	if !ok || strings.ToUpper(ident.Value) != "APPEND" {
		return false
	}
	// First argument must be an assignable target (the slice variable).
	target, ok := call.Arguments[0].(*parser.Identifier)
	if !ok {
		return false
	}
	args := make([]string, 0, len(call.Arguments))
	args = append(args, g.toGoIdent(target.Value))
	for _, a := range call.Arguments[1:] {
		args = append(args, g.exprToGo(a))
	}
	g.writeLine(fmt.Sprintf("%s = append(%s)", g.toGoIdent(target.Value), strings.Join(args, ", ")))
	return true
}

// collectAndEmitStatics walks every sub/function/method body, gathers
// STATIC declarations, and emits them as package-level vars with
// uniquified names (`_static_<funcName>_<varName>`). The corresponding
// in-body declarations are skipped at codegen time, and identifier
// references inside the function are routed to the uniquified name via
// toGoIdent's static lookup.
func (g *Generator) collectAndEmitStatics() {
	type entry struct {
		funcName string
		uniq     string
		stmt     *parser.DimStatement
	}
	var entries []entry

	visit := func(funcName string, body *parser.BlockStatement) {
		if body == nil {
			return
		}
		for _, st := range body.Statements {
			ds, ok := st.(*parser.DimStatement)
			if !ok || !ds.IsStatic {
				continue
			}
			uniq := fmt.Sprintf("_static_%s_%s", funcName, ds.Name.Value)
			if g.statics[funcName] == nil {
				g.statics[funcName] = map[string]string{}
			}
			g.statics[funcName][ds.Name.Value] = uniq
			entries = append(entries, entry{funcName: funcName, uniq: uniq, stmt: ds})
		}
	}

	for _, st := range g.program.Statements {
		switch s := st.(type) {
		case *parser.SubStatement:
			visit(s.Name.Value, s.Body)
		case *parser.FunctionStatement:
			visit(s.Name.Value, s.Body)
		case *parser.MethodStatement:
			visit(s.Name.Value, s.Body)
		}
	}

	if len(entries) == 0 {
		return
	}

	g.writeLine("var (")
	g.indent++
	for _, e := range entries {
		typeStr := g.typeSpecToGo(e.stmt.Type)
		if e.stmt.Value != nil {
			oldFunc := g.currentFunc
			g.currentFunc = e.funcName
			val := g.exprToGo(e.stmt.Value)
			g.currentFunc = oldFunc
			g.writeLine(fmt.Sprintf("%s %s = %s", e.uniq, typeStr, val))
		} else {
			g.writeLine(fmt.Sprintf("%s %s", e.uniq, typeStr))
		}
	}
	g.indent--
	g.writeLine(")")
	g.writeLine("")
}

// generateWith emits a WITH block. The body's `.field` shortcuts were
// already expanded at parse time into full MemberExpressions on the
// receiver, so codegen here just emits the body — no synthetic binding
// is needed. This keeps QB-compatible semantics where assignments
// through `.field` modify the original receiver, not a copy.
func (g *Generator) generateWith(stmt *parser.WithStatement) {
	g.generateBlockStatement(stmt.Body)
}

// generateLineInput emits a call to the LineInput runtime helper.
func (g *Generator) generateLineInput(stmt *parser.LineInputStatement) {
	g.runtimeFuncs["LineInput"] = true
	prompt := `""`
	if stmt.Prompt != nil {
		prompt = g.exprToGo(stmt.Prompt)
	}
	g.writeLine(fmt.Sprintf("%s = LineInput(%s)", g.toGoIdent(stmt.Var.Value), prompt))
}

// generateReDim emits a slice resize. Without PRESERVE the new slice
// is fresh; with PRESERVE the old elements are copied into it (truncated
// or zero-padded as needed).
func (g *Generator) generateReDim(stmt *parser.ReDimStatement) {
	name := g.toGoIdent(stmt.Name.Value)
	elemType := g.typeSpecToGo(stmt.Type)
	size := g.exprToGo(stmt.ArraySize)
	if stmt.Preserve {
		old := name + "_old"
		g.writeLine(fmt.Sprintf("%s := %s", old, name))
		g.writeLine(fmt.Sprintf("%s = make([]%s, %s)", name, elemType, size))
		g.writeLine(fmt.Sprintf("copy(%s, %s)", name, old))
	} else {
		g.writeLine(fmt.Sprintf("%s = make([]%s, %s)", name, elemType, size))
	}
}

func (g *Generator) generateLocalDim(stmt *parser.DimStatement) {
	// STATIC dims are hoisted to package-level vars in collectAndEmitStatics.
	// Skip the in-body emission entirely; identifier references are routed
	// to the uniquified package-level name via toGoIdent.
	if stmt.IsStatic {
		return
	}
	varName := g.toGoIdent(stmt.Name.Value)
	varType := g.typeSpecToGo(stmt.Type)

	// Detect rebind: if a variable of this name was already declared in
	// THIS scope (not just shadowed from an outer scope), emit an
	// assignment to the existing slot instead of re-declaring. Use
	// ResolveLocal — Resolve would walk up to parents and false-trigger
	// on shadowing.
	rebind := false
	if g.currentScope != nil && g.currentScope.ResolveLocal(stmt.Name.Value) != nil {
		rebind = true
	} else if g.currentScope != nil {
		t := g.typeFromTypeSpec(stmt.Type)
		g.currentScope.Define(&analyzer.Symbol{
			Name: stmt.Name.Value,
			Kind: analyzer.SymVariable,
			Type: t,
		})
	}

	if stmt.ArraySize != nil {
		op := ":="
		if rebind {
			op = "="
		}
		g.writeLineWithSource(fmt.Sprintf("%s %s make([]%s, %s)", varName, op, varType, g.exprToGo(stmt.ArraySize)), stmt.Token.Line)
	} else if stmt.Value != nil {
		if rebind {
			g.writeLineWithSource(fmt.Sprintf("%s = %s", varName, g.exprToGo(stmt.Value)), stmt.Token.Line)
		} else {
			g.writeLineWithSource(fmt.Sprintf("var %s %s = %s", varName, varType, g.exprToGo(stmt.Value)), stmt.Token.Line)
		}
	} else if stmt.Type != nil && stmt.Type.IsMap {
		op := ":="
		if rebind {
			op = "="
		}
		g.writeLineWithSource(fmt.Sprintf("%s %s make(%s)", varName, op, varType), stmt.Token.Line)
	} else if !rebind {
		g.writeLineWithSource(fmt.Sprintf("var %s %s", varName, varType), stmt.Token.Line)
	}
	// Bare `DIM x AS T` rebind with no value is a no-op (var already exists).
}

func (g *Generator) generateLet(stmt *parser.LetStatement) {
	varName := g.toGoIdent(stmt.Name.Value)
	// Use := for type inference
	g.writeLineWithSource(fmt.Sprintf("%s := %s", varName, g.exprToGo(stmt.Value)), stmt.Token.Line)
}

func (g *Generator) generateAssignment(stmt *parser.AssignmentStatement) {
	left := g.exprToGo(stmt.Left)
	right := g.exprToGo(stmt.Value)
	g.writeLine(fmt.Sprintf("%s = %s", left, right))
}

func (g *Generator) generateMultiAssignment(stmt *parser.MultiAssignmentStatement) {
	var targets []string
	for _, t := range stmt.Targets {
		targets = append(targets, g.exprToGo(t))
	}
	g.writeLine(fmt.Sprintf("%s = %s", strings.Join(targets, ", "), g.exprToGo(stmt.Value)))

	// ONERR auto-check: when an ONERR handler is active and the analyzer
	// determined that the last value of this multi-assignment is of type
	// ERROR, emit the canonical Go `if err != nil { ... }` block. This is
	// purely a code-rewrite — no defer/recover, no try/catch.
	if g.currentOnErr != nil && stmt.LastTargetIsError && len(targets) > 0 {
		errVar := targets[len(targets)-1]
		g.emitOnErrCheck(errVar)
	}
}

// containsOnErrGoto walks a block and its nested blocks looking for an
// ONERR GOTO directive. Used by sub/function/method codegen to decide
// whether to hoist DIM declarations to the function top — Go's `goto`
// cannot jump over a variable declaration, so when ONERR GOTO is in
// play we move the declarations above any potential jump origin.
func (g *Generator) containsOnErrGoto(block *parser.BlockStatement) bool {
	if block == nil {
		return false
	}
	for _, st := range block.Statements {
		if g.stmtContainsOnErrGoto(st) {
			return true
		}
	}
	return false
}

func (g *Generator) stmtContainsOnErrGoto(stmt parser.Statement) bool {
	switch s := stmt.(type) {
	case *parser.OnErrStatement:
		return s.Action == parser.OnErrGoto
	case *parser.IfStatement:
		if g.containsOnErrGoto(s.Consequence) {
			return true
		}
		for _, ei := range s.ElseIfs {
			if g.containsOnErrGoto(ei.Consequence) {
				return true
			}
		}
		return g.containsOnErrGoto(s.Alternative)
	case *parser.ForStatement:
		return g.containsOnErrGoto(s.Body)
	case *parser.WhileStatement:
		return g.containsOnErrGoto(s.Body)
	case *parser.DoLoopStatement:
		return g.containsOnErrGoto(s.Body)
	case *parser.SelectStatement:
		for _, c := range s.Cases {
			if g.containsOnErrGoto(c.Body) {
				return true
			}
		}
		return g.containsOnErrGoto(s.Default)
	case *parser.WithStatement:
		return g.containsOnErrGoto(s.Body)
	}
	return false
}

// hoistDimsForOnErrGoto pre-emits a `var name type` for every non-STATIC
// DIM in the function body and pre-populates the codegen scope so the
// existing generateLocalDim "rebind" path takes over: each DIM in the
// body becomes an assignment-only emission. The net effect is identical
// to writing declarations at the top of every BASIC sub by hand, which
// is what classic BASIC programmers expect anyway. Without this, an
// ONERR GOTO that jumps past a DIM would be rejected by Go.
func (g *Generator) hoistDimsForOnErrGoto(block *parser.BlockStatement) {
	seen := map[string]bool{}
	g.collectAndEmitHoistedDims(block, seen)
}

func (g *Generator) collectAndEmitHoistedDims(block *parser.BlockStatement, seen map[string]bool) {
	if block == nil {
		return
	}
	for _, st := range block.Statements {
		switch s := st.(type) {
		case *parser.DimStatement:
			if s == nil || s.IsStatic || s.Name == nil {
				continue
			}
			if seen[s.Name.Value] {
				continue
			}
			seen[s.Name.Value] = true
			g.emitHoistedDim(s)
		case *parser.IfStatement:
			g.collectAndEmitHoistedDims(s.Consequence, seen)
			for _, ei := range s.ElseIfs {
				g.collectAndEmitHoistedDims(ei.Consequence, seen)
			}
			g.collectAndEmitHoistedDims(s.Alternative, seen)
		case *parser.ForStatement:
			g.collectAndEmitHoistedDims(s.Body, seen)
		case *parser.WhileStatement:
			g.collectAndEmitHoistedDims(s.Body, seen)
		case *parser.DoLoopStatement:
			g.collectAndEmitHoistedDims(s.Body, seen)
		case *parser.SelectStatement:
			for _, c := range s.Cases {
				g.collectAndEmitHoistedDims(c.Body, seen)
			}
			g.collectAndEmitHoistedDims(s.Default, seen)
		case *parser.WithStatement:
			g.collectAndEmitHoistedDims(s.Body, seen)
		}
	}
}

// emitHoistedDim emits a single hoisted `var <name> <type>` and registers
// the name in the current scope so the in-body DIM emission takes the
// rebind path (assignment-only).
func (g *Generator) emitHoistedDim(d *parser.DimStatement) {
	name := g.toGoIdent(d.Name.Value)
	var goType string
	if d.ArraySize != nil {
		goType = "[]" + g.typeSpecToGo(d.Type)
	} else {
		goType = g.typeSpecToGo(d.Type)
	}
	g.writeLine(fmt.Sprintf("var %s %s", name, goType))
	if g.currentScope != nil {
		g.currentScope.Define(&analyzer.Symbol{
			Name: d.Name.Value,
			Kind: analyzer.SymVariable,
			Type: g.typeFromTypeSpec(d.Type),
		})
	}
}

// generateOnErr applies the ONERR directive to the current function's
// codegen state. It produces no Go output by itself — the effect lives
// in subsequent multi-assignments.
func (g *Generator) generateOnErr(stmt *parser.OnErrStatement) {
	if stmt.Action == parser.OnErrClear {
		g.currentOnErr = nil
		return
	}
	g.currentOnErr = stmt
}

// emitOnErrCheck writes the conditional that triggers the active ONERR
// handler. errVar is the already-rendered Go identifier of the error
// value (last target of the preceding multi-assignment).
func (g *Generator) emitOnErrCheck(errVar string) {
	switch g.currentOnErr.Action {
	case parser.OnErrGoto:
		label := g.toGoIdent(g.currentOnErr.Target.Value)
		g.writeLine(fmt.Sprintf("if %s != nil { goto %s }", errVar, label))
	case parser.OnErrGoFunc:
		handler := g.toGoIdent(g.currentOnErr.Target.Value)
		g.writeLine(fmt.Sprintf("if %s != nil { %s(%s); return }", errVar, handler, errVar))
	}
}

func (g *Generator) generatePrint(stmt *parser.PrintStatement) {
	if len(stmt.Values) == 0 {
		g.writeLine("fmt.Println()")
		return
	}

	var args []string
	for _, v := range stmt.Values {
		args = append(args, g.exprToGo(v))
	}

	// Trailing separator means no newline
	// If there are as many separators as values, the last separator is trailing
	suppressNewline := len(stmt.Separators) >= len(stmt.Values)

	if suppressNewline {
		g.writeLine(fmt.Sprintf("fmt.Print(%s)", strings.Join(args, ", ")))
	} else {
		g.writeLine(fmt.Sprintf("fmt.Println(%s)", strings.Join(args, ", ")))
	}
}

func (g *Generator) generateInput(stmt *parser.InputStatement) {
	g.imports["bufio"] = ""
	g.imports["os"] = ""
	g.imports["strings"] = ""

	varName := g.toGoIdent(stmt.Variable.Value)

	if stmt.Prompt != nil {
		g.writeLine(fmt.Sprintf("fmt.Print(%s)", g.exprToGo(stmt.Prompt)))
	}

	g.writeLine("_reader := bufio.NewReader(os.Stdin)")
	g.writeLine(fmt.Sprintf("%s, _ = _reader.ReadString('\\n')", varName))
	g.writeLine(fmt.Sprintf("%s = strings.TrimRight(%s, \"\\r\\n\")", varName, varName))
}

func (g *Generator) generateIf(stmt *parser.IfStatement) {
	g.writeLine(fmt.Sprintf("if %s {", g.exprToGo(stmt.Condition)))
	g.indent++
	g.generateBlockStatement(stmt.Consequence)
	g.indent--

	for _, elseif := range stmt.ElseIfs {
		g.writeLine(fmt.Sprintf("} else if %s {", g.exprToGo(elseif.Condition)))
		g.indent++
		g.generateBlockStatement(elseif.Consequence)
		g.indent--
	}

	if stmt.Alternative != nil {
		g.writeLine("} else {")
		g.indent++
		g.generateBlockStatement(stmt.Alternative)
		g.indent--
	}

	g.writeLine("}")
}

func (g *Generator) generateFor(stmt *parser.ForStatement) {
	varName := g.toGoIdent(stmt.Variable.Value)
	start := g.exprToGo(stmt.Start)
	end := g.exprToGo(stmt.End)

	step := "1"
	if stmt.Step != nil {
		step = g.exprToGo(stmt.Step)
	}

	// Determine comparison operator based on step direction
	comparison := "<="
	if g.isNegativeStep(stmt.Step) {
		comparison = ">="
	}

	// Auto-declare the loop variable if it isn't already in scope.
	// This matches traditional BASIC, where `FOR i = 1 TO 10` does not
	// require a separate DIM. We emit `var i int` before the loop so the
	// outer scope can read i's final value (matching the legacy semantics
	// when the user does pre-declare).
	if g.currentScope != nil && g.currentScope.Resolve(stmt.Variable.Value) == nil {
		g.writeLine(fmt.Sprintf("var %s int", varName))
		g.currentScope.Define(&analyzer.Symbol{
			Name: stmt.Variable.Value,
			Kind: analyzer.SymVariable,
			Type: analyzer.IntegerType,
		})
	}

	g.beginLoop(stmt.Body)
	g.writeLine(fmt.Sprintf("for %s = %s; %s %s %s; %s += %s {",
		varName, start, varName, comparison, end, varName, step))
	g.indent++
	g.generateBlockStatement(stmt.Body)
	g.indent--
	g.writeLine("}")
	g.popLoop()
}

// isNegativeStep checks if the step expression evaluates to a negative
// value. Handles literal negatives (`-1`), prefix-minus on a literal
// (`-(1)`), and any compile-time-foldable integer arithmetic — most
// importantly `0 - 1`, which is what BASIC code typically writes when
// it can't use the prefix-minus form (e.g. parser context disallows it).
// Float literals fall back to a direct sign check.
//
// When this returns true the FOR codegen flips the loop comparison
// from `<=` to `>=`; an incorrect false here is the classic "downward
// FOR runs once when the slice is empty" bug.
func (g *Generator) isNegativeStep(step parser.Expression) bool {
	if step == nil {
		return false
	}
	if v, ok := constFoldInt(step); ok {
		return v < 0
	}
	if floatLit, ok := step.(*parser.FloatLiteral); ok {
		return floatLit.Value < 0
	}
	return false
}

// constFoldInt walks an expression tree and, if every leaf is an
// integer literal joined by +, -, *, / or unary +/-, returns the
// folded value. Returns (0, false) for anything that touches an
// identifier, function call, or non-int literal — those have to
// stay as runtime expressions in the emitted Go.
func constFoldInt(e parser.Expression) (int64, bool) {
	switch n := e.(type) {
	case *parser.IntegerLiteral:
		return n.Value, true
	case *parser.PrefixExpression:
		v, ok := constFoldInt(n.Right)
		if !ok {
			return 0, false
		}
		switch n.Operator {
		case "-":
			return -v, true
		case "+":
			return v, true
		}
		return 0, false
	case *parser.InfixExpression:
		l, lok := constFoldInt(n.Left)
		r, rok := constFoldInt(n.Right)
		if !lok || !rok {
			return 0, false
		}
		switch n.Operator {
		case "+":
			return l + r, true
		case "-":
			return l - r, true
		case "*":
			return l * r, true
		case "/":
			if r == 0 {
				return 0, false
			}
			return l / r, true
		}
	}
	return 0, false
}

func (g *Generator) generateWhile(stmt *parser.WhileStatement) {
	g.beginLoop(stmt.Body)
	g.writeLine(fmt.Sprintf("for %s {", g.exprToGo(stmt.Condition)))
	g.indent++
	g.generateBlockStatement(stmt.Body)
	g.indent--
	g.writeLine("}")
	g.popLoop()
}

func (g *Generator) generateDoLoop(stmt *parser.DoLoopStatement) {
	g.beginLoop(stmt.Body)
	defer g.popLoop()

	if stmt.Condition == nil {
		// Infinite loop
		g.writeLine("for {")
		g.indent++
		g.generateBlockStatement(stmt.Body)
		g.indent--
		g.writeLine("}")
		return
	}

	if stmt.IsPreCondition {
		cond := g.exprToGo(stmt.Condition)
		if !stmt.IsWhile {
			cond = "!(" + cond + ")"
		}
		g.writeLine(fmt.Sprintf("for %s {", cond))
		g.indent++
		g.generateBlockStatement(stmt.Body)
		g.indent--
		g.writeLine("}")
		return
	}

	// --- Post-condition (DO ... LOOP WHILE/UNTIL) ---
	//
	// The natural shape is a `for {}` with the test at the bottom of the
	// body. That is what we emit, and it has one virtue worth keeping: the
	// test sits inside the body's block, so a condition may refer to a
	// variable DIMmed inside the loop.
	//
	// But Go's `continue` jumps straight past the bottom of the body to the
	// top of the loop, which would skip that test and spin forever. So when
	// the body contains a CONTINUE we switch to a three-part for loop and
	// put the test in the post statement, which `continue` DOES run.
	//
	// The trade-off of that second form is the mirror of the first: the post
	// statement lives outside the body's block, so a LOOP WHILE condition
	// cannot refer to a variable declared inside the loop. Declare it just
	// before the DO if you need both.
	cond := g.exprToGo(stmt.Condition)

	if bodyHasContinue(stmt.Body) {
		keepGoing := cond
		if !stmt.IsWhile {
			keepGoing = "!(" + cond + ")" // LOOP UNTIL x == carry on while NOT x
		}
		g.loopSeq++
		again := fmt.Sprintf("dbAgain%d", g.loopSeq)
		// Starts true so the body always runs at least once, then the post
		// statement re-tests it after every iteration -- including after a
		// CONTINUE.
		g.writeLine(fmt.Sprintf("for %s := true; %s; %s = %s {", again, again, again, keepGoing))
		g.indent++
		g.generateBlockStatement(stmt.Body)
		g.indent--
		g.writeLine("}")
		return
	}

	g.writeLine("for {")
	g.indent++
	g.generateBlockStatement(stmt.Body)
	if stmt.IsWhile {
		g.writeLine(fmt.Sprintf("if !(%s) { break }", cond))
	} else {
		g.writeLine(fmt.Sprintf("if %s { break }", cond))
	}
	g.indent--
	g.writeLine("}")
}

func (g *Generator) generateSelect(stmt *parser.SelectStatement) {
	testExpr := g.exprToGo(stmt.TestExpr)
	// SELECT CASE becomes a Go switch, which swallows a bare `break`.
	// Track that so EXIT inside here knows to use the loop's label.
	g.switchDepth++
	defer func() { g.switchDepth-- }()
	g.writeLine(fmt.Sprintf("switch %s {", testExpr))

	for _, caseClause := range stmt.Cases {
		var vals []string
		for _, v := range caseClause.Values {
			vals = append(vals, g.exprToGo(v))
		}
		g.writeLine(fmt.Sprintf("case %s:", strings.Join(vals, ", ")))
		g.indent++
		g.generateBlockStatement(caseClause.Body)
		g.indent--
	}

	if stmt.Default != nil {
		g.writeLine("default:")
		g.indent++
		g.generateBlockStatement(stmt.Default)
		g.indent--
	}

	g.writeLine("}")
}

func (g *Generator) generateReturn(stmt *parser.ReturnStatement) {
	if len(stmt.Values) == 0 {
		g.writeLine("return")
		return
	}

	var vals []string
	for _, v := range stmt.Values {
		vals = append(vals, g.exprToGo(v))
	}
	g.writeLine(fmt.Sprintf("return %s", strings.Join(vals, ", ")))
}

func (g *Generator) generateExit(stmt *parser.ExitStatement) {
	// EXIT FOR, EXIT WHILE, EXIT DO all become break
	// EXIT SUB, EXIT FUNCTION become return
	switch strings.ToUpper(stmt.ExitType) {
	case "FOR", "WHILE", "DO":
		// A bare `break` inside a Go switch breaks the SWITCH, not the loop.
		// SELECT CASE compiles to a switch, so when the EXIT sits inside one
		// we have to name the loop we mean. enclosingLoopLabel gives us the
		// label the loop was emitted with, or "" when a plain break is right.
		if label := g.enclosingLoopLabel(); label != "" {
			g.writeLine("break " + label)
			return
		}
		g.writeLine("break")
	case "SUB", "FUNCTION":
		g.writeLine("return")
	}
}

// generateContinue emits Go's `continue`.
//
// Unlike `break`, Go's `continue` only ever binds to a for loop -- switches
// and selects are invisible to it -- so this never needs a label.
func (g *Generator) generateContinue(stmt *parser.ContinueStatement) {
	g.writeLine("continue")
}

// --- loop bookkeeping -------------------------------------------------
//
// Two things need to know how loops nest:
//
//   * EXIT inside a SELECT CASE has to say WHICH loop it is leaving, because
//     Go's bare `break` would only leave the switch. That means the loop must
//     have been emitted with a label -- and we have to know that before we
//     write the `for` line, so loopNeedsLabel scans the body first.
//   * A post-test DO loop is emitted differently when it contains a CONTINUE
//     (see generateDoLoop).

// pushLoop starts a loop context. Passing a non-empty label records that the
// loop was emitted with that label, so EXIT can refer to it.
func (g *Generator) pushLoop(label string) {
	g.loopLabels = append(g.loopLabels, label)
	g.switchDepths = append(g.switchDepths, g.switchDepth)
	g.switchDepth = 0
}

func (g *Generator) popLoop() {
	g.loopLabels = g.loopLabels[:len(g.loopLabels)-1]
	g.switchDepth = g.switchDepths[len(g.switchDepths)-1]
	g.switchDepths = g.switchDepths[:len(g.switchDepths)-1]
}

// enclosingLoopLabel returns the label of the innermost loop when a plain
// `break` would be captured by a switch, and "" when a plain break is fine.
func (g *Generator) enclosingLoopLabel() string {
	if g.switchDepth == 0 || len(g.loopLabels) == 0 {
		return ""
	}
	return g.loopLabels[len(g.loopLabels)-1]
}

// nextLoopLabel hands out a fresh, collision-proof loop label.
func (g *Generator) nextLoopLabel() string {
	g.loopSeq++
	return fmt.Sprintf("dbLoop%d", g.loopSeq)
}

// beginLoop decides whether this loop needs a label, writes the label line if
// so, and pushes the loop context. The caller must call g.popLoop() at the end.
func (g *Generator) beginLoop(body *parser.BlockStatement) {
	label := ""
	if loopNeedsLabel(body, 0) {
		label = g.nextLoopLabel()
		// A Go label goes on its own line immediately before the statement.
		g.writeLine(label + ":")
	}
	g.pushLoop(label)
}

// loopNeedsLabel reports whether a loop body contains an EXIT FOR/WHILE/DO
// that sits inside a SELECT CASE, which is the only situation where a bare
// Go `break` would go to the wrong place.
//
// switchDepth counts the SELECT CASE statements we have descended into. We
// deliberately do NOT descend into nested loops: a break or continue inside
// one of those belongs to it, not to us.
func loopNeedsLabel(node parser.Node, switchDepth int) bool {
	switch n := node.(type) {
	case nil:
		return false

	case *parser.BlockStatement:
		if n == nil {
			return false
		}
		for _, st := range n.Statements {
			if loopNeedsLabel(st, switchDepth) {
				return true
			}
		}

	case *parser.ExitStatement:
		switch strings.ToUpper(n.ExitType) {
		case "FOR", "WHILE", "DO":
			return switchDepth > 0
		}

	case *parser.IfStatement:
		if loopNeedsLabel(n.Consequence, switchDepth) {
			return true
		}
		for _, elif := range n.ElseIfs {
			if loopNeedsLabel(elif.Consequence, switchDepth) {
				return true
			}
		}
		if loopNeedsLabel(n.Alternative, switchDepth) {
			return true
		}

	case *parser.SelectStatement:
		for _, c := range n.Cases {
			if loopNeedsLabel(c.Body, switchDepth+1) {
				return true
			}
		}
		if loopNeedsLabel(n.Default, switchDepth+1) {
			return true
		}

	case *parser.WithStatement:
		return loopNeedsLabel(n.Body, switchDepth)

		// ForStatement, WhileStatement and DoLoopStatement are intentionally
		// absent: they capture break and continue themselves.
	}
	return false
}

// bodyHasContinue reports whether a loop body contains a CONTINUE belonging
// to this loop (so, again, not descending into nested loops). Switches are
// transparent to `continue`, so unlike loopNeedsLabel we descend into them
// without counting.
func bodyHasContinue(node parser.Node) bool {
	switch n := node.(type) {
	case nil:
		return false

	case *parser.BlockStatement:
		if n == nil {
			return false
		}
		for _, st := range n.Statements {
			if bodyHasContinue(st) {
				return true
			}
		}

	case *parser.ContinueStatement:
		return true

	case *parser.IfStatement:
		if bodyHasContinue(n.Consequence) {
			return true
		}
		for _, elif := range n.ElseIfs {
			if bodyHasContinue(elif.Consequence) {
				return true
			}
		}
		return bodyHasContinue(n.Alternative)

	case *parser.SelectStatement:
		for _, c := range n.Cases {
			if bodyHasContinue(c.Body) {
				return true
			}
		}
		return bodyHasContinue(n.Default)

	case *parser.WithStatement:
		return bodyHasContinue(n.Body)
	}
	return false
}

func (g *Generator) generateGoto(stmt *parser.GotoStatement) {
	g.writeLine(fmt.Sprintf("goto %s", g.toGoIdent(stmt.Label)))
}

func (g *Generator) generateLabel(stmt *parser.LabelStatement) {
	// Labels need to be at column 0 in Go
	label := g.toGoIdent(stmt.Name)
	g.output.WriteString(label + ":\n")
}

func (g *Generator) generateSpawn(stmt *parser.SpawnStatement) {
	g.writeLine(fmt.Sprintf("go %s", g.exprToGo(stmt.Call)))
}

func (g *Generator) generateDefer(stmt *parser.DeferStatement) {
	g.writeLine(fmt.Sprintf("defer %s", g.exprToGo(stmt.Call)))
}

// functionLiteralToGo emits a Go func literal for an anonymous FUNCTION/SUB.
// The body is generated into a swapped output buffer so it can be embedded
// inline as part of the surrounding expression's string.
func (g *Generator) functionLiteralToGo(lit *parser.FunctionLiteral) string {
	// Header: parameters
	paramParts := make([]string, 0, len(lit.Params))
	for _, p := range lit.Params {
		paramName := g.toGoIdent(p.Name.Value)
		paramType := g.typeSpecToGo(p.Type)
		paramParts = append(paramParts, paramName+" "+paramType)
	}
	paramStr := strings.Join(paramParts, ", ")

	// Header: return type(s)
	var returnStr string
	if !lit.IsSub && len(lit.ReturnTypes) > 0 {
		if len(lit.ReturnTypes) == 1 {
			returnStr = " " + g.typeSpecToGo(lit.ReturnTypes[0])
		} else {
			rts := make([]string, 0, len(lit.ReturnTypes))
			for _, rt := range lit.ReturnTypes {
				rts = append(rts, g.typeSpecToGo(rt))
			}
			returnStr = " (" + strings.Join(rts, ", ") + ")"
		}
	}

	// Body: swap output buffer and scope, generate, capture, restore.
	savedOutput := g.output
	savedScope := g.currentScope
	g.output = strings.Builder{}
	g.currentScope = analyzer.NewScope("__lambda__", g.currentScope)
	for _, p := range lit.Params {
		paramType := g.typeFromTypeSpec(p.Type)
		g.currentScope.Define(&analyzer.Symbol{
			Name: p.Name.Value,
			Kind: analyzer.SymParameter,
			Type: paramType,
		})
	}
	g.indent++
	g.generateBlockStatement(lit.Body)
	g.indent--
	body := g.output.String()
	g.output = savedOutput
	g.currentScope = savedScope

	closingIndent := strings.Repeat("\t", g.indent)
	return fmt.Sprintf("func(%s)%s {\n%s%s}", paramStr, returnStr, body, closingIndent)
}

func (g *Generator) generateSend(stmt *parser.SendStatement) {
	g.writeLine(fmt.Sprintf("%s <- %s", g.exprToGo(stmt.Channel), g.exprToGo(stmt.Value)))
}

func (g *Generator) generateReceive(stmt *parser.ReceiveStatement) {
	if stmt.OkVar != nil {
		g.writeLine(fmt.Sprintf("%s, %s = <-%s",
			g.exprToGo(stmt.Variable), g.exprToGo(stmt.OkVar), g.exprToGo(stmt.Channel)))
		return
	}
	g.writeLine(fmt.Sprintf("%s = <-%s", g.exprToGo(stmt.Variable), g.exprToGo(stmt.Channel)))
}

func (g *Generator) exprToGo(expr parser.Expression) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *parser.Identifier:
		return g.toGoIdent(e.Value)
	case *parser.IntegerLiteral:
		return fmt.Sprintf("%d", e.Value)
	case *parser.FloatLiteral:
		return fmt.Sprintf("%v", e.Value)
	case *parser.StringLiteral:
		return fmt.Sprintf("%q", e.Value)
	case *parser.ByteStringLiteral:
		return fmt.Sprintf("[]byte(%q)", e.Value)
	case *parser.BooleanLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *parser.NilLiteral:
		return "nil"
	case *parser.JSONLiteral:
		return g.jsonLiteralToGo(e)
	case *parser.ArrayLiteral:
		return g.arrayLiteralToGo(e)
	case *parser.StructLiteral:
		return g.structLiteralToGo(e)
	case *parser.FunctionLiteral:
		return g.functionLiteralToGo(e)
	case *parser.SliceLiteral:
		return g.sliceLiteralToGo(e)
	case *parser.PrefixExpression:
		return g.prefixExprToGo(e)
	case *parser.InfixExpression:
		return g.infixExprToGo(e)
	case *parser.CallExpression:
		return g.callExprToGo(e)
	case *parser.IndexExpression:
		if e.IsSlice {
			// Slice operation: [start:end], [start:], [:end], [:]
			start := ""
			end := ""
			if e.Index != nil {
				start = g.exprToGo(e.Index)
			}
			if e.End != nil {
				end = g.exprToGo(e.End)
			}
			return fmt.Sprintf("%s[%s:%s]", g.exprToGo(e.Left), start, end)
		}
		idx := g.exprToGo(e.Index)
		if g.optionBase == 1 {
			idx = "(" + idx + " - 1)"
		}
		return fmt.Sprintf("%s[%s]", g.exprToGo(e.Left), idx)
	case *parser.MemberExpression:
		return g.memberExprToGo(e)
	case *parser.AddressOfExpression:
		return fmt.Sprintf("&%s", g.exprToGo(e.Value))
	case *parser.DereferenceExpression:
		// Wrap in parentheses to ensure correct precedence with member access
		// e.g., (*ptr).field instead of *ptr.field
		return fmt.Sprintf("(*%s)", g.exprToGo(e.Value))
	case *parser.MakeChanExpression:
		chanType := g.typeSpecToGo(e.ChannelType)
		if e.Size != nil {
			return fmt.Sprintf("make(chan %s, %s)", chanType, g.exprToGo(e.Size))
		}
		return fmt.Sprintf("make(chan %s)", chanType)
	case *parser.ReceiveExpression:
		return fmt.Sprintf("<-%s", g.exprToGo(e.Channel))
	case *parser.SpreadExpression:
		return g.exprToGo(e.Value) + "..."
	case *parser.TypeAssertionExpression:
		targetType := g.typeSpecToGo(e.TargetType)
		return fmt.Sprintf("%s.(%s)", g.exprToGo(e.Value), targetType)
	default:
		return "/* unknown expression */"
	}
}

func (g *Generator) jsonLiteralToGo(lit *parser.JSONLiteral) string {
	if len(lit.Pairs) == 0 {
		return "map[string]interface{}{}"
	}

	var pairs []string
	for k, v := range lit.Pairs {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, g.exprToGo(v)))
	}
	return fmt.Sprintf("map[string]interface{}{%s}", strings.Join(pairs, ", "))
}

func (g *Generator) arrayLiteralToGo(lit *parser.ArrayLiteral) string {
	if len(lit.Elements) == 0 {
		return "[]interface{}{}"
	}

	// Infer the element type from the first element
	elemType := g.inferElementType(lit.Elements[0])

	var elems []string
	for _, e := range lit.Elements {
		elems = append(elems, g.exprToGo(e))
	}
	return fmt.Sprintf("[]%s{%s}", elemType, strings.Join(elems, ", "))
}

// inferElementType determines the Go type from an expression
func (g *Generator) inferElementType(expr parser.Expression) string {
	switch e := expr.(type) {
	case *parser.StringLiteral:
		return "string"
	case *parser.IntegerLiteral:
		return "int"
	case *parser.FloatLiteral:
		return "float64"
	case *parser.BooleanLiteral:
		return "bool"
	case *parser.StructLiteral:
		return e.TypeName
	case *parser.SliceLiteral:
		return "[]" + e.ElementType
	case *parser.Identifier:
		// Try to look up the type in the symbol table
		if g.symbols != nil {
			if sym := g.symbols.Resolve(e.Value); sym != nil && sym.Type != nil {
				return sym.Type.GoType()
			}
		}
		return "interface{}"
	default:
		return "interface{}"
	}
}

func (g *Generator) structLiteralToGo(lit *parser.StructLiteral) string {
	typeName := lit.TypeName

	// Check if this is a user-defined type
	if g.types != nil {
		if t := g.types.Lookup(typeName); t != nil {
			typeName = t.Name
		}
	}

	if len(lit.Fields) == 0 {
		return typeName + "{}"
	}

	var pairs []string
	for k, v := range lit.Fields {
		// Convert field name to Go identifier (capitalize first letter)
		goFieldName := g.toGoIdent(k)
		pairs = append(pairs, fmt.Sprintf("%s: %s", goFieldName, g.exprToGo(v)))
	}
	return fmt.Sprintf("%s{%s}", typeName, strings.Join(pairs, ", "))
}

func (g *Generator) sliceLiteralToGo(lit *parser.SliceLiteral) string {
	elementType := g.mapTypeToGo(lit.ElementType)

	if len(lit.Elements) == 0 {
		return "[]" + elementType + "{}"
	}

	var elements []string
	for _, e := range lit.Elements {
		elements = append(elements, g.exprToGo(e))
	}
	return fmt.Sprintf("[]%s{%s}", elementType, strings.Join(elements, ", "))
}

// mapTypeToGo converts a DBasic type name to Go type
func (g *Generator) mapTypeToGo(typeName string) string {
	// Check if this is a user-defined type
	if g.types != nil {
		if t := g.types.Lookup(typeName); t != nil {
			return t.Name
		}
	}

	// Map primitive types
	switch strings.ToUpper(typeName) {
	case "INTEGER":
		return "int"
	case "LONG":
		return "int64"
	case "SINGLE":
		return "float32"
	case "DOUBLE":
		return "float64"
	case "STRING":
		return "string"
	case "BOOLEAN":
		return "bool"
	case "JSON":
		return "map[string]interface{}"
	case "BYTES", "BSTRING":
		return "[]byte"
	case "ANY":
		return "interface{}"
	case "ERROR":
		return "error"
	default:
		return typeName
	}
}

func (g *Generator) prefixExprToGo(expr *parser.PrefixExpression) string {
	right := g.exprToGo(expr.Right)

	switch expr.Operator {
	case "NOT":
		return fmt.Sprintf("!(%s)", g.toBoolGo(expr.Right, right))
	case "-":
		return fmt.Sprintf("-%s", right)
	default:
		return fmt.Sprintf("%s%s", expr.Operator, right)
	}
}

// isProvablyNumeric returns true only when codegen can prove an
// expression is a number — used to decide whether AND/OR/XOR/NOT need
// the (x != 0) truthiness wrap. Anything we cannot positively identify
// as numeric (function calls, struct field access, external types) is
// left alone so Go's normal bool semantics still apply.
func (g *Generator) isProvablyNumeric(expr parser.Expression) bool {
	switch e := expr.(type) {
	case *parser.IntegerLiteral, *parser.FloatLiteral:
		return true
	case *parser.PrefixExpression:
		return e.Operator == "-"
	case *parser.InfixExpression:
		switch e.Operator {
		case "+", "-", "*", "/", "\\", "^", "MOD":
			return true
		}
	case *parser.Identifier:
		if g.currentScope != nil {
			if sym := g.currentScope.Resolve(e.Value); sym != nil && sym.Type != nil {
				return sym.Type.IsNumeric()
			}
		}
	}
	return false
}

// toBoolGo wraps an operand as (x != 0) when we can prove it's numeric
// (so AND/OR/XOR/NOT can apply Go's `&&` / `||` / `!`). Anything we
// can't prove is numeric is passed through unchanged — preserving the
// old strict-boolean behavior for bool-returning calls / struct fields.
func (g *Generator) toBoolGo(expr parser.Expression, rendered string) string {
	if g.isProvablyNumeric(expr) {
		return fmt.Sprintf("(%s != 0)", rendered)
	}
	return rendered
}

func (g *Generator) infixExprToGo(expr *parser.InfixExpression) string {
	left := g.exprToGo(expr.Left)
	right := g.exprToGo(expr.Right)

	switch expr.Operator {
	case "=":
		return fmt.Sprintf("(%s == %s)", left, right)
	case "<>":
		return fmt.Sprintf("(%s != %s)", left, right)
	case "AND":
		return fmt.Sprintf("(%s && %s)", g.toBoolGo(expr.Left, left), g.toBoolGo(expr.Right, right))
	case "OR":
		return fmt.Sprintf("(%s || %s)", g.toBoolGo(expr.Left, left), g.toBoolGo(expr.Right, right))
	case "XOR":
		l := g.toBoolGo(expr.Left, left)
		r := g.toBoolGo(expr.Right, right)
		return fmt.Sprintf("((%s || %s) && !(%s && %s))", l, r, l, r)
	case "MOD":
		return fmt.Sprintf("(%s %% %s)", left, right)
	case "&":
		return fmt.Sprintf("(%s + %s)", left, right) // String concatenation
	case "^":
		g.imports["math"] = ""
		return fmt.Sprintf("math.Pow(float64(%s), float64(%s))", left, right)
	case "\\":
		return fmt.Sprintf("(%s / %s)", left, right) // Integer division
	default:
		return fmt.Sprintf("(%s %s %s)", left, expr.Operator, right)
	}
}

func (g *Generator) callExprToGo(call *parser.CallExpression) string {
	var args []string
	for _, arg := range call.Arguments {
		args = append(args, g.exprToGo(arg))
	}

	funcName := g.exprToGo(call.Function)

	// Handle builtin functions that map directly to Go
	switch strings.ToUpper(funcName) {
	case "APPEND":
		// APPEND(slice, elem) -> append(slice, elem)
		return fmt.Sprintf("append(%s)", strings.Join(args, ", "))
	case "LEN":
		// LEN can work on strings, slices, maps, etc.
		if len(args) == 1 {
			return fmt.Sprintf("len(%s)", args[0])
		}
	case "CAP":
		// CAP for slice capacity
		if len(args) == 1 {
			return fmt.Sprintf("cap(%s)", args[0])
		}
	case "MAKE":
		// MAKE([]TYPE, len) or MAKE([]TYPE, len, cap)
		return fmt.Sprintf("make(%s)", strings.Join(args, ", "))
	case "COPY":
		// COPY(dst, src) -> copy(dst, src)
		return fmt.Sprintf("copy(%s)", strings.Join(args, ", "))
	case "DELETE":
		// DELETE(map, key) -> delete(map, key)
		return fmt.Sprintf("delete(%s)", strings.Join(args, ", "))
	case "CLOSE":
		// CLOSE(channel) -> close(channel)
		return fmt.Sprintf("close(%s)", strings.Join(args, ", "))
	case "PANIC":
		// PANIC(msg) -> panic(msg)
		return fmt.Sprintf("panic(%s)", strings.Join(args, ", "))
	case "RECOVER":
		// RECOVER() -> recover()
		return "recover()"
	case "NEW":
		// NEW(Type) -> new(Type)
		return fmt.Sprintf("new(%s)", strings.Join(args, ", "))
	case "STRING":
		// STRING(bytes) or STRING(runes) -> string(...)
		return fmt.Sprintf("string(%s)", strings.Join(args, ", "))
	case "RUNE":
		// RUNE(int) -> rune(int)
		if len(args) == 1 {
			return fmt.Sprintf("rune(%s)", args[0])
		}
	case "BYTE":
		// BYTE(int) -> byte(int)
		if len(args) == 1 {
			return fmt.Sprintf("byte(%s)", args[0])
		}
	case "PRINTF":
		// Printf(format, args...) -> fmt.Printf(format, args...)
		g.imports["fmt"] = ""
		return fmt.Sprintf("fmt.Printf(%s)", strings.Join(args, ", "))
	case "SPRINTF":
		// Sprintf(format, args...) -> fmt.Sprintf(format, args...)
		g.imports["fmt"] = ""
		return fmt.Sprintf("fmt.Sprintf(%s)", strings.Join(args, ", "))
	case "NEWERROR":
		// NewError(message) -> dbasic.NewErrorAtFunc(file, line, func, message)
		g.runtimeFuncs["NewErrorAtFunc"] = true
		sourceFile := g.sourceFile
		if sourceFile == "" {
			sourceFile = "unknown"
		}
		funcName := g.currentFunc
		if funcName == "" {
			funcName = "main"
		}
		return fmt.Sprintf("NewErrorAtFunc(%q, %d, %q, %s)", sourceFile, call.Token.Line, funcName, strings.Join(args, ", "))
	case "ERRORF":
		// Errorf(format, args...) -> dbasic.ErrorfFunc(file, line, func, format, args...)
		g.runtimeFuncs["ErrorfFunc"] = true
		sourceFile := g.sourceFile
		if sourceFile == "" {
			sourceFile = "unknown"
		}
		funcName := g.currentFunc
		if funcName == "" {
			funcName = "main"
		}
		return fmt.Sprintf("ErrorfFunc(%q, %d, %q, %s)", sourceFile, call.Token.Line, funcName, strings.Join(args, ", "))
	case "WRAPERROR":
		// WrapError(err, message) -> dbasic.WrapError(err, file, line, func, message)
		g.runtimeFuncs["WrapError"] = true
		sourceFile := g.sourceFile
		if sourceFile == "" {
			sourceFile = "unknown"
		}
		funcName := g.currentFunc
		if funcName == "" {
			funcName = "main"
		}
		if len(args) >= 2 {
			return fmt.Sprintf("WrapError(%s, %q, %d, %q, %s)", args[0], sourceFile, call.Token.Line, funcName, strings.Join(args[1:], ", "))
		}
		return fmt.Sprintf("WrapError(%s)", strings.Join(args, ", "))
	}

	if len(call.TypeArgs) > 0 {
		typeArgs := make([]string, 0, len(call.TypeArgs))
		for _, ts := range call.TypeArgs {
			typeArgs = append(typeArgs, g.typeSpecToGo(ts))
		}
		return fmt.Sprintf("%s[%s](%s)", funcName, strings.Join(typeArgs, ", "), strings.Join(args, ", "))
	}

	return fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))
}

func (g *Generator) typeSpecToGo(spec *parser.TypeSpec) string {
	if spec == nil {
		return "interface{}"
	}

	if spec.IsPointer {
		return "*" + g.typeSpecToGo(spec.ElementType)
	}

	if spec.IsChannel {
		switch spec.ChanDir {
		case parser.ChanRecv:
			return "<-chan " + g.typeSpecToGo(spec.ElementType)
		case parser.ChanSend:
			return "chan<- " + g.typeSpecToGo(spec.ElementType)
		}
		return "chan " + g.typeSpecToGo(spec.ElementType)
	}

	if spec.IsMap {
		return fmt.Sprintf("map[%s]%s", g.typeSpecToGo(spec.KeyType), g.typeSpecToGo(spec.ElementType))
	}

	if spec.IsArray {
		// Slice type (dynamic array)
		if spec.ArraySize == nil {
			return "[]" + g.typeSpecToGo(spec.ElementType)
		}
		// Fixed-size array
		return fmt.Sprintf("[%s]%s", g.exprToGo(spec.ArraySize), g.typeSpecToGo(spec.ElementType))
	}

	switch strings.ToUpper(spec.Name) {
	case "INTEGER":
		return "int"
	case "LONG":
		return "int64"
	case "SINGLE":
		return "float32"
	case "DOUBLE":
		return "float64"
	case "STRING":
		return "string"
	case "BOOLEAN":
		return "bool"
	case "JSON":
		return "map[string]interface{}"
	case "BYTES", "BSTRING":
		return "[]byte"
	case "ANY":
		return "interface{}"
	case "ERROR":
		return "error"
	default:
		// Check for custom type
		if g.types != nil {
			if t := g.types.Lookup(spec.Name); t != nil {
				return g.toGoIdent(t.Name)
			}
		}
		// Could be a custom type name; mangle Go reserved words like `type`.
		return g.toGoIdent(spec.Name)
	}
}

// typeFromTypeSpec converts a parser.TypeSpec to an analyzer.Type
func (g *Generator) typeFromTypeSpec(spec *parser.TypeSpec) *analyzer.Type {
	if spec == nil {
		return analyzer.AnyType
	}

	if spec.IsPointer {
		return analyzer.NewPointerType(g.typeFromTypeSpec(spec.ElementType))
	}

	if spec.IsChannel {
		return analyzer.NewChannelType(g.typeFromTypeSpec(spec.ElementType))
	}

	if spec.IsMap {
		return analyzer.NewMapType(g.typeFromTypeSpec(spec.KeyType), g.typeFromTypeSpec(spec.ElementType))
	}

	if spec.IsArray {
		elemType := g.typeFromTypeSpec(spec.ElementType)
		if spec.ArraySize == nil {
			return analyzer.NewSliceType(elemType)
		}
		// Fixed array - we'd need to evaluate the size
		return analyzer.NewSliceType(elemType)
	}

	// Try built-in type first
	if t := analyzer.TypeFromName(spec.Name); t != nil {
		return t
	}

	// Try custom type
	if g.types != nil {
		if t := g.types.Lookup(spec.Name); t != nil {
			return t
		}
	}

	return analyzer.AnyType
}

// isExprJSONType checks if an expression resolves to a JSON type
func (g *Generator) isExprJSONType(expr parser.Expression) bool {
	switch e := expr.(type) {
	case *parser.Identifier:
		sym := g.currentScope.Resolve(e.Value)
		if sym != nil && sym.Type != nil {
			return sym.Type.Kind == analyzer.TypeJSON
		}
	case *parser.MemberExpression:
		// If we're accessing a member of something, check the object
		return g.isExprJSONType(e.Object)
	case *parser.IndexExpression:
		return g.isExprJSONType(e.Left)
	case *parser.JSONLiteral:
		return true
	}
	return false
}

// memberExprToGo generates Go code for a member expression, handling JSON specially
func (g *Generator) memberExprToGo(expr *parser.MemberExpression) string {
	if g.isExprJSONType(expr.Object) {
		// JSON access uses map bracket notation
		return fmt.Sprintf("%s[%q]", g.exprToGo(expr.Object), expr.Member.Value)
	}
	// Regular struct/package access uses dot notation
	return fmt.Sprintf("%s.%s", g.exprToGo(expr.Object), g.toGoIdent(expr.Member.Value))
}

func (g *Generator) toGoIdent(name string) string {
	// Handle reserved words and capitalization
	// For now, just use the name as-is but capitalize first letter
	// for exported functions
	if len(name) == 0 {
		return name
	}

	// STATIC variables defined in the current sub get rewritten to their
	// hoisted package-level names.
	if g.currentFunc != "" {
		if subStatics, ok := g.statics[g.currentFunc]; ok {
			if uniq, ok := subStatics[name]; ok {
				return uniq
			}
		}
	}

	// Check for Go reserved words
	reserved := map[string]string{
		"break":       "break_",
		"case":        "case_",
		"chan":        "chan_",
		"const":       "const_",
		"continue":    "continue_",
		"default":     "default_",
		"defer":       "defer_",
		"else":        "else_",
		"fallthrough": "fallthrough_",
		"for":         "for_",
		"func":        "func_",
		"go":          "go_",
		"goto":        "goto_",
		"if":          "if_",
		"import":      "import_",
		"interface":   "interface_",
		"map":         "map_",
		"package":     "package_",
		"range":       "range_",
		"return":      "return_",
		"select":      "select_",
		"struct":      "struct_",
		"switch":      "switch_",
		"type":        "type_",
		"var":         "var_",
	}

	// Case-SENSITIVE check: Go keywords are all lowercase, so only an
	// identifier written exactly as a keyword (e.g. a DBasic var named
	// `select`) needs the `_` suffix. A capitalized member that merely
	// shares a keyword's spelling — like the foreign Go methods .Select(),
	// .Type(), .Map(), .Range() — is exported and must be emitted verbatim.
	// (Lowercasing first wrongly mangled those into .select_, .type_, etc.)
	if replacement, ok := reserved[name]; ok {
		return replacement
	}

	return name
}

func (g *Generator) writeLine(s string) {
	for i := 0; i < g.indent; i++ {
		g.output.WriteString("\t")
	}
	g.output.WriteString(s)
	g.output.WriteString("\n")
}

// writeLineWithSource writes a line with optional source location comment
func (g *Generator) writeLineWithSource(s string, line int) {
	if g.debugMode && line > 0 {
		for i := 0; i < g.indent; i++ {
			g.output.WriteString("\t")
		}
		g.output.WriteString(s)
		g.output.WriteString(fmt.Sprintf(" // line %d", line))
		g.output.WriteString("\n")
	} else {
		g.writeLine(s)
	}
}

// emitLineDirective writes a //line directive at column 0 so the Go compiler
// reports errors and panics against the original .dbas source, not the
// generated main.go. Must start at column 0 to be honored by Go.
func (g *Generator) emitLineDirective(line int) {
	if g.sourceFile == "" || line <= 0 {
		return
	}
	g.output.WriteString(fmt.Sprintf("//line %s:%d\n", g.sourceFile, line))
}

// writeComment writes a comment line
func (g *Generator) writeComment(s string) {
	for i := 0; i < g.indent; i++ {
		g.output.WriteString("\t")
	}
	g.output.WriteString("// ")
	g.output.WriteString(s)
	g.output.WriteString("\n")
}
