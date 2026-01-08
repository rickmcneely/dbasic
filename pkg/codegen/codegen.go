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
	currentFunc     string            // Current function/sub name for error context
}

// New creates a new code generator
func New(program *parser.Program, symbols *analyzer.SymbolTable) *Generator {
	return &Generator{
		program:      program,
		symbols:      symbols,
		currentScope: symbols.GlobalScope,
		imports:      make(map[string]string),
		runtimeFuncs: make(map[string]bool),
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

	// Generate functions, subs, and methods
	g.generateFunctions()

	// Generate main function if needed
	if g.hasMain {
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
	case *parser.InputStatement:
		g.imports["bufio"] = ""
		g.imports["os"] = ""
		g.imports["strings"] = ""
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
	// Always include fmt for PRINT
	g.imports["fmt"] = ""

	// Add user imports with their aliases
	for _, imp := range g.symbols.AllImports() {
		g.imports[imp.Path] = imp.Alias
	}
}

// runtimeFuncDefs contains the Go source for runtime functions
var runtimeFuncDefs = map[string]string{
	"Int": `// Int converts to int
func Int(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
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
	case *parser.DimStatement:
		if s.Value != nil {
			g.scanExprForRuntimeFuncs(s.Value)
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
	g.currentScope = analyzer.NewScope(stmt.Name.Value, g.symbols.GlobalScope)
	g.currentFunc = stmt.Name.Value
	// Add parameters to local scope
	for _, p := range stmt.Params {
		paramType := g.typeFromTypeSpec(p.Type)
		g.currentScope.Define(&analyzer.Symbol{
			Name: p.Name.Value,
			Kind: analyzer.SymParameter,
			Type: paramType,
		})
	}
	g.generateBlockStatement(stmt.Body)
	g.currentScope = oldScope
	g.currentFunc = oldFunc
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
	g.currentScope = analyzer.NewScope(stmt.Name.Value, g.symbols.GlobalScope)
	g.currentFunc = stmt.Name.Value
	// Add parameters to local scope
	for _, p := range stmt.Params {
		paramType := g.typeFromTypeSpec(p.Type)
		g.currentScope.Define(&analyzer.Symbol{
			Name: p.Name.Value,
			Kind: analyzer.SymParameter,
			Type: paramType,
		})
	}
	g.generateBlockStatement(stmt.Body)
	g.currentScope = oldScope
	g.currentFunc = oldFunc
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
	g.currentScope = analyzer.NewScope(stmt.Name.Value, g.symbols.GlobalScope)
	g.currentFunc = stmt.Name.Value

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

	g.generateBlockStatement(stmt.Body)
	g.currentScope = oldScope
	g.currentFunc = oldFunc
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

func (g *Generator) generateStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.DimStatement:
		g.generateLocalDim(s)
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
	case *parser.GotoStatement:
		g.generateGoto(s)
	case *parser.LabelStatement:
		g.generateLabel(s)
	case *parser.SpawnStatement:
		g.generateSpawn(s)
	case *parser.SendStatement:
		g.generateSend(s)
	case *parser.ReceiveStatement:
		g.generateReceive(s)
	case *parser.ExpressionStatement:
		if s.Expression != nil {
			g.writeLine(g.exprToGo(s.Expression))
		}
	}
}

func (g *Generator) generateLocalDim(stmt *parser.DimStatement) {
	varName := g.toGoIdent(stmt.Name.Value)
	varType := g.typeSpecToGo(stmt.Type)

	// Track the variable type in current scope
	t := g.typeFromTypeSpec(stmt.Type)
	g.currentScope.Define(&analyzer.Symbol{
		Name: stmt.Name.Value,
		Kind: analyzer.SymVariable,
		Type: t,
	})

	if stmt.ArraySize != nil {
		g.writeLineWithSource(fmt.Sprintf("%s := make([]%s, %s)", varName, varType, g.exprToGo(stmt.ArraySize)), stmt.Token.Line)
	} else if stmt.Value != nil {
		g.writeLineWithSource(fmt.Sprintf("var %s %s = %s", varName, varType, g.exprToGo(stmt.Value)), stmt.Token.Line)
	} else {
		g.writeLineWithSource(fmt.Sprintf("var %s %s", varName, varType), stmt.Token.Line)
	}
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

	// In BASIC, FOR loop variables are typically pre-declared
	// Use = instead of := to avoid redeclaration
	g.writeLine(fmt.Sprintf("for %s = %s; %s <= %s; %s += %s {",
		varName, start, varName, end, varName, step))
	g.indent++
	g.generateBlockStatement(stmt.Body)
	g.indent--
	g.writeLine("}")
}

func (g *Generator) generateWhile(stmt *parser.WhileStatement) {
	g.writeLine(fmt.Sprintf("for %s {", g.exprToGo(stmt.Condition)))
	g.indent++
	g.generateBlockStatement(stmt.Body)
	g.indent--
	g.writeLine("}")
}

func (g *Generator) generateDoLoop(stmt *parser.DoLoopStatement) {
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
	} else {
		// Post-condition - use for loop with break
		g.writeLine("for {")
		g.indent++
		g.generateBlockStatement(stmt.Body)
		cond := g.exprToGo(stmt.Condition)
		if stmt.IsWhile {
			g.writeLine(fmt.Sprintf("if !(%s) { break }", cond))
		} else {
			g.writeLine(fmt.Sprintf("if %s { break }", cond))
		}
		g.indent--
		g.writeLine("}")
	}
}

func (g *Generator) generateSelect(stmt *parser.SelectStatement) {
	testExpr := g.exprToGo(stmt.TestExpr)
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
		g.writeLine("break")
	case "SUB", "FUNCTION":
		g.writeLine("return")
	}
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

func (g *Generator) generateSend(stmt *parser.SendStatement) {
	g.writeLine(fmt.Sprintf("%s <- %s", g.exprToGo(stmt.Channel), g.exprToGo(stmt.Value)))
}

func (g *Generator) generateReceive(stmt *parser.ReceiveStatement) {
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
		return fmt.Sprintf("%s[%s]", g.exprToGo(e.Left), g.exprToGo(e.Index))
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
		return fmt.Sprintf("!(%s)", right)
	case "-":
		return fmt.Sprintf("-%s", right)
	default:
		return fmt.Sprintf("%s%s", expr.Operator, right)
	}
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
		return fmt.Sprintf("(%s && %s)", left, right)
	case "OR":
		return fmt.Sprintf("(%s || %s)", left, right)
	case "XOR":
		return fmt.Sprintf("((%s || %s) && !(%s && %s))", left, right, left, right)
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
		return "chan " + g.typeSpecToGo(spec.ElementType)
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
				return t.Name
			}
		}
		// Could be a custom type name
		return spec.Name
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

	lower := strings.ToLower(name)
	if replacement, ok := reserved[lower]; ok {
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

// writeComment writes a comment line
func (g *Generator) writeComment(s string) {
	for i := 0; i < g.indent; i++ {
		g.output.WriteString("\t")
	}
	g.output.WriteString("// ")
	g.output.WriteString(s)
	g.output.WriteString("\n")
}
