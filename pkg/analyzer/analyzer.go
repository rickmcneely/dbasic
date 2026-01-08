package analyzer

import (
	"fmt"
	"strings"

	"github.com/zditech/dbasic/pkg/parser"
)

// Analyzer performs semantic analysis on the AST
type Analyzer struct {
	symbols  *SymbolTable
	types    *TypeRegistry
	errors   []string
	program  *parser.Program
	lines    []string // source lines for error context
}

// New creates a new Analyzer
func New() *Analyzer {
	a := &Analyzer{
		symbols: NewSymbolTable(),
		types:   NewTypeRegistry(),
		errors:  []string{},
	}
	a.registerBuiltins()
	return a
}

// registerBuiltins adds built-in runtime functions to the symbol table
func (a *Analyzer) registerBuiltins() {
	// String functions
	a.addBuiltin("Len", []*Type{StringType}, []*Type{IntegerType})
	a.addBuiltin("Left", []*Type{StringType, IntegerType}, []*Type{StringType})
	a.addBuiltin("Right", []*Type{StringType, IntegerType}, []*Type{StringType})
	a.addBuiltin("Mid", []*Type{StringType, IntegerType, IntegerType}, []*Type{StringType})
	a.addBuiltin("Instr", []*Type{StringType, StringType}, []*Type{IntegerType})
	a.addBuiltin("UCase", []*Type{StringType}, []*Type{StringType})
	a.addBuiltin("LCase", []*Type{StringType}, []*Type{StringType})
	a.addBuiltin("Trim", []*Type{StringType}, []*Type{StringType})
	a.addBuiltin("LTrim", []*Type{StringType}, []*Type{StringType})
	a.addBuiltin("RTrim", []*Type{StringType}, []*Type{StringType})
	a.addBuiltin("Replace", []*Type{StringType, StringType, StringType}, []*Type{StringType})
	a.addBuiltin("Str", []*Type{AnyType}, []*Type{StringType})
	a.addBuiltin("Val", []*Type{StringType}, []*Type{DoubleType})
	a.addBuiltin("Asc", []*Type{StringType}, []*Type{IntegerType})
	a.addBuiltin("Chr", []*Type{IntegerType}, []*Type{StringType})
	a.addBuiltin("Space", []*Type{IntegerType}, []*Type{StringType})

	// Type conversion
	a.addBuiltin("Int", []*Type{AnyType}, []*Type{IntegerType})
	a.addBuiltin("Lng", []*Type{AnyType}, []*Type{LongType})
	a.addBuiltin("Sng", []*Type{AnyType}, []*Type{SingleType})
	a.addBuiltin("Dbl", []*Type{AnyType}, []*Type{DoubleType})
	a.addBuiltin("Bool", []*Type{AnyType}, []*Type{BooleanType})

	// Byte functions
	a.addBuiltin("Encode", []*Type{StringType}, []*Type{BytesType})
	a.addBuiltin("Decode", []*Type{BytesType}, []*Type{StringType})
	a.addBuiltin("MakeBytes", []*Type{IntegerType}, []*Type{BytesType})
	a.addBuiltin("LenBytes", []*Type{BytesType}, []*Type{IntegerType})

	// Math functions
	a.addBuiltin("Abs", []*Type{DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Sqr", []*Type{DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Sin", []*Type{DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Cos", []*Type{DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Tan", []*Type{DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Atn", []*Type{DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Atn2", []*Type{DoubleType, DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Log", []*Type{DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Log10", []*Type{DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Exp", []*Type{DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Pow", []*Type{DoubleType, DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Sgn", []*Type{DoubleType}, []*Type{IntegerType})
	a.addBuiltin("Fix", []*Type{DoubleType}, []*Type{LongType})
	a.addBuiltin("Floor", []*Type{DoubleType}, []*Type{LongType})
	a.addBuiltin("Ceil", []*Type{DoubleType}, []*Type{LongType})
	a.addBuiltin("Round", []*Type{DoubleType}, []*Type{LongType})
	a.addBuiltin("Min", []*Type{DoubleType, DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Max", []*Type{DoubleType, DoubleType}, []*Type{DoubleType})
	a.addBuiltin("Clamp", []*Type{DoubleType, DoubleType, DoubleType}, []*Type{DoubleType})
	a.addBuiltin("PI", []*Type{}, []*Type{DoubleType})

	// Random functions
	a.addBuiltin("Rnd", []*Type{}, []*Type{DoubleType})
	a.addBuiltin("RndInt", []*Type{IntegerType}, []*Type{IntegerType})
	a.addBuiltin("RndRange", []*Type{IntegerType, IntegerType}, []*Type{IntegerType})
	a.addBuiltin("Randomize", []*Type{LongType}, []*Type{})

	// Date/Time functions
	a.addBuiltin("Timer", []*Type{}, []*Type{DoubleType})
	a.addBuiltin("Now", []*Type{}, []*Type{LongType})
	a.addBuiltin("Date", []*Type{}, []*Type{StringType})
	a.addBuiltin("Year", []*Type{}, []*Type{IntegerType})
	a.addBuiltin("Month", []*Type{}, []*Type{IntegerType})
	a.addBuiltin("Day", []*Type{}, []*Type{IntegerType})
	a.addBuiltin("Hour", []*Type{}, []*Type{IntegerType})
	a.addBuiltin("Minute", []*Type{}, []*Type{IntegerType})
	a.addBuiltin("Second", []*Type{}, []*Type{IntegerType})
	a.addBuiltin("Sleep", []*Type{IntegerType}, []*Type{})

	// File functions
	a.addBuiltin("FileExists", []*Type{StringType}, []*Type{BooleanType})
	a.addBuiltin("ReadFile", []*Type{StringType}, []*Type{StringType})
	a.addBuiltin("WriteFile", []*Type{StringType, StringType}, []*Type{})
	a.addBuiltin("AppendFile", []*Type{StringType, StringType}, []*Type{})
	a.addBuiltin("DeleteFile", []*Type{StringType}, []*Type{})
	a.addBuiltin("MkDir", []*Type{StringType}, []*Type{})
	a.addBuiltin("RmDir", []*Type{StringType}, []*Type{})

	// Printf functions (variadic - additional args not type-checked)
	a.addVariadicBuiltin("Printf", []*Type{StringType}, []*Type{})            // fmt.Printf(format, args...)
	a.addVariadicBuiltin("Sprintf", []*Type{StringType}, []*Type{StringType}) // fmt.Sprintf(format, args...)

	// Error functions
	a.addBuiltin("NewError", []*Type{StringType}, []*Type{ErrorType})                  // errors.New(message)
	a.addVariadicBuiltin("Errorf", []*Type{StringType}, []*Type{ErrorType})            // fmt.Errorf(format, args...)
	a.addBuiltin("WrapError", []*Type{ErrorType, StringType}, []*Type{ErrorType})      // Wrap error with context

	// JSON functions
	a.addBuiltin("JSONParse", []*Type{StringType}, []*Type{JSONType})
	a.addBuiltin("JSONStringify", []*Type{JSONType}, []*Type{StringType})
	a.addBuiltin("JSONPretty", []*Type{JSONType}, []*Type{StringType})
	a.addBuiltin("JSONGet", []*Type{JSONType, StringType}, []*Type{AnyType})
	a.addBuiltin("JSONSet", []*Type{JSONType, StringType, AnyType}, []*Type{})

	// Struct/JSON conversion functions
	a.addBuiltin("StructToJSON", []*Type{AnyType}, []*Type{JSONType})
	a.addBuiltin("JSONToStruct", []*Type{JSONType, AnyType}, []*Type{AnyType})
}

func (a *Analyzer) addBuiltin(name string, params []*Type, returns []*Type) {
	var symType *Type
	if len(returns) > 0 {
		symType = NewFunctionType(params, returns)
	} else {
		symType = NewSubType(params)
	}

	sym := &Symbol{
		Name: name,
		Kind: SymFunction,
		Type: symType,
	}
	a.symbols.DefineGlobal(sym)
}

func (a *Analyzer) addVariadicBuiltin(name string, params []*Type, returns []*Type) {
	var symType *Type
	if len(returns) > 0 {
		symType = NewVariadicFunctionType(params, returns)
	} else {
		symType = NewVariadicSubType(params)
	}

	sym := &Symbol{
		Name: name,
		Kind: SymFunction,
		Type: symType,
	}
	a.symbols.DefineGlobal(sym)
}

// SetSource sets the source code for error context
func (a *Analyzer) SetSource(source string) {
	a.lines = strings.Split(source, "\n")
}

// getSourceLine returns the source line at the given line number (1-indexed)
func (a *Analyzer) getSourceLine(lineNum int) string {
	if lineNum < 1 || lineNum > len(a.lines) {
		return ""
	}
	return a.lines[lineNum-1]
}

// Analyze performs semantic analysis on the program
func (a *Analyzer) Analyze(program *parser.Program) (*SymbolTable, []string) {
	a.program = program

	// First pass: collect all imports (must be before type declarations)
	for _, stmt := range program.Statements {
		if is, ok := stmt.(*parser.ImportStatement); ok {
			a.symbols.AddImport(is.Package, is.Alias)
		}
	}

	// Second pass: collect all type definitions
	for _, stmt := range program.Statements {
		if ts, ok := stmt.(*parser.TypeStatement); ok {
			a.declareType(ts)
		}
	}

	// Third pass: collect all function/sub/method declarations
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *parser.SubStatement:
			a.declareSubOrFunction(s.Name.Value, s.Params, nil, s)
		case *parser.FunctionStatement:
			a.declareSubOrFunction(s.Name.Value, s.Params, s.ReturnTypes, s)
		case *parser.MethodStatement:
			a.declareMethod(s)
		}
	}

	// Fourth pass: declare global DIM statements (so they're available in all functions)
	for _, stmt := range program.Statements {
		if ds, ok := stmt.(*parser.DimStatement); ok {
			varType := a.resolveTypeSpec(ds.Type)
			sym := &Symbol{
				Name: ds.Name.Value,
				Kind: SymVariable,
				Type: varType,
			}
			a.symbols.DefineGlobal(sym)
		}
	}

	// Fifth pass: analyze all statements
	for _, stmt := range program.Statements {
		a.analyzeStatement(stmt)
	}

	return a.symbols, a.errors
}

// TypeRegistry returns the type registry
func (a *Analyzer) TypeRegistry() *TypeRegistry {
	return a.types
}

// Errors returns the list of errors
func (a *Analyzer) Errors() []string {
	return a.errors
}

// SymbolTable returns the symbol table
func (a *Analyzer) SymbolTable() *SymbolTable {
	return a.symbols
}

func (a *Analyzer) error(line int, format string, args ...interface{}) {
	a.errorWithHint(line, format, "", args...)
}

func (a *Analyzer) errorWithHint(line int, format string, hint string, args ...interface{}) {
	var sb strings.Builder
	msg := fmt.Sprintf(format, args...)

	if line > 0 {
		sb.WriteString(fmt.Sprintf("semantic error at line %d: %s\n", line, msg))

		// Show source line with context
		sourceLine := a.getSourceLine(line)
		if sourceLine != "" {
			sb.WriteString(fmt.Sprintf("  %d | %s\n", line, sourceLine))
		}
	} else {
		sb.WriteString(fmt.Sprintf("semantic error: %s\n", msg))
	}

	if hint != "" {
		sb.WriteString(fmt.Sprintf("  hint: %s\n", hint))
	}

	a.errors = append(a.errors, sb.String())
}

func (a *Analyzer) declareType(stmt *parser.TypeStatement) {
	var fields []*StructField
	for _, f := range stmt.Fields {
		fieldType := a.resolveTypeSpec(f.Type)
		fields = append(fields, &StructField{
			Name: f.Name.Value,
			Type: fieldType,
		})
	}

	structType := NewStructType(stmt.Name.Value, fields)
	structType.Implements = stmt.Implements // Copy interface info
	a.types.Register(stmt.Name.Value, structType)
}

func (a *Analyzer) declareMethod(stmt *parser.MethodStatement) {
	// Get the receiver type name
	var receiverTypeName string
	if stmt.ReceiverType.IsPointer {
		receiverTypeName = stmt.ReceiverType.ElementType.Name
	} else {
		receiverTypeName = stmt.ReceiverType.Name
	}

	// Method name is TypeName.MethodName for symbol table
	methodName := receiverTypeName + "." + stmt.Name.Value

	var paramTypes []*Type
	for _, p := range stmt.Params {
		paramTypes = append(paramTypes, a.resolveTypeSpec(p.Type))
	}

	var retTypes []*Type
	for _, rt := range stmt.ReturnTypes {
		retTypes = append(retTypes, a.resolveTypeSpec(rt))
	}

	symType := NewFunctionType(paramTypes, retTypes)

	sym := &Symbol{
		Name: methodName,
		Kind: SymFunction,
		Type: symType,
		Node: stmt,
	}

	if err := a.symbols.DefineGlobal(sym); err != nil {
		a.error(stmt.Token.Line, "duplicate method definition: %s", methodName)
	}
}

func (a *Analyzer) declareSubOrFunction(name string, params []*parser.Parameter, returnTypes []*parser.TypeSpec, node parser.Node) {
	var paramTypes []*Type
	for _, p := range params {
		paramTypes = append(paramTypes, a.resolveTypeSpec(p.Type))
	}

	var retTypes []*Type
	for _, rt := range returnTypes {
		retTypes = append(retTypes, a.resolveTypeSpec(rt))
	}

	var symType *Type
	var symKind SymbolKind

	if len(returnTypes) > 0 {
		symType = NewFunctionType(paramTypes, retTypes)
		symKind = SymFunction
	} else {
		symType = NewSubType(paramTypes)
		symKind = SymSub
	}

	sym := &Symbol{
		Name: name,
		Kind: symKind,
		Type: symType,
		Node: node,
	}

	if err := a.symbols.DefineGlobal(sym); err != nil {
		a.error(0, "duplicate definition: %s", name)
	}
}

func (a *Analyzer) resolveTypeSpec(spec *parser.TypeSpec) *Type {
	if spec == nil {
		return VoidType
	}

	if spec.IsPointer {
		elemType := a.resolveTypeSpec(spec.ElementType)
		return NewPointerType(elemType)
	}

	if spec.IsChannel {
		elemType := a.resolveTypeSpec(spec.ElementType)
		return NewChannelType(elemType)
	}

	// Handle slice/array types with []TYPE syntax
	if spec.IsArray {
		var elemType *Type
		if spec.ElementType != nil {
			// New syntax: []TYPE - element type is in ElementType
			elemType = a.resolveTypeSpec(spec.ElementType)
		} else {
			// Legacy: TYPE() syntax - element type comes from Name
			elemType = TypeFromName(spec.Name)
			if elemType == nil {
				elemType = a.types.Lookup(spec.Name)
			}
			if elemType == nil {
				a.error(spec.Token.Line, "unknown type: %s", spec.Name)
				return AnyType
			}
		}
		return NewSliceType(elemType)
	}

	// Check for external Go type (package.Type syntax)
	if strings.Contains(spec.Name, ".") {
		parts := strings.SplitN(spec.Name, ".", 2)
		alias := parts[0]
		typeName := parts[1]

		// Verify the import exists
		importInfo := a.symbols.GetImport(alias)
		if importInfo == nil {
			a.error(spec.Token.Line, "unknown package: %s", alias)
			return AnyType
		}

		return NewExternalType(alias, typeName, importInfo.Path)
	}

	// Try built-in types first
	baseType := TypeFromName(spec.Name)
	if baseType == nil {
		// Try user-defined types
		baseType = a.types.Lookup(spec.Name)
	}
	if baseType == nil {
		a.error(spec.Token.Line, "unknown type: %s", spec.Name)
		return AnyType
	}

	return baseType
}

func (a *Analyzer) analyzeStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.ImportStatement:
		// Already handled in first pass
	case *parser.DimStatement:
		a.analyzeDimStatement(s)
	case *parser.LetStatement:
		a.analyzeLetStatement(s)
	case *parser.ConstStatement:
		a.analyzeConstStatement(s)
	case *parser.AssignmentStatement:
		a.analyzeAssignmentStatement(s)
	case *parser.MultiAssignmentStatement:
		a.analyzeMultiAssignmentStatement(s)
	case *parser.PrintStatement:
		a.analyzePrintStatement(s)
	case *parser.InputStatement:
		a.analyzeInputStatement(s)
	case *parser.IfStatement:
		a.analyzeIfStatement(s)
	case *parser.ForStatement:
		a.analyzeForStatement(s)
	case *parser.WhileStatement:
		a.analyzeWhileStatement(s)
	case *parser.DoLoopStatement:
		a.analyzeDoLoopStatement(s)
	case *parser.SelectStatement:
		a.analyzeSelectStatement(s)
	case *parser.TypeStatement:
		// Already handled in first pass
	case *parser.SubStatement:
		a.analyzeSubStatement(s)
	case *parser.FunctionStatement:
		a.analyzeFunctionStatement(s)
	case *parser.MethodStatement:
		a.analyzeMethodStatement(s)
	case *parser.ReturnStatement:
		a.analyzeReturnStatement(s)
	case *parser.ExitStatement:
		// Valid exit types are checked by parser
	case *parser.GotoStatement:
		// Label resolution is done later
	case *parser.LabelStatement:
		a.analyzeLabelStatement(s)
	case *parser.SpawnStatement:
		a.analyzeSpawnStatement(s)
	case *parser.SendStatement:
		a.analyzeSendStatement(s)
	case *parser.ReceiveStatement:
		a.analyzeReceiveStatement(s)
	case *parser.ExpressionStatement:
		if s.Expression != nil {
			a.analyzeExpression(s.Expression)
		}
	}
}

func (a *Analyzer) analyzeDimStatement(stmt *parser.DimStatement) {
	varType := a.resolveTypeSpec(stmt.Type)

	// Skip defining global variables (they're already defined in the earlier pass)
	if !a.symbols.IsGlobalScope() || a.symbols.Resolve(stmt.Name.Value) == nil {
		sym := &Symbol{
			Name: stmt.Name.Value,
			Kind: SymVariable,
			Type: varType,
			Node: stmt,
		}

		if err := a.symbols.Define(sym); err != nil {
			a.error(stmt.Token.Line, err.Error())
		}
	}

	if stmt.Value != nil {
		valueType := a.analyzeExpression(stmt.Value)
		if !varType.IsCompatibleWith(valueType) {
			a.error(stmt.Token.Line, "type mismatch: cannot assign %s to %s",
				valueType.String(), varType.String())
		}
	}
}

func (a *Analyzer) analyzeLetStatement(stmt *parser.LetStatement) {
	// Infer type from the value expression
	valueType := a.analyzeExpression(stmt.Value)

	sym := &Symbol{
		Name: stmt.Name.Value,
		Kind: SymVariable,
		Type: valueType,
		Node: stmt,
	}

	if err := a.symbols.Define(sym); err != nil {
		a.error(stmt.Token.Line, err.Error())
	}
}

func (a *Analyzer) analyzeConstStatement(stmt *parser.ConstStatement) {
	constType := a.resolveTypeSpec(stmt.Type)

	sym := &Symbol{
		Name: stmt.Name.Value,
		Kind: SymConstant,
		Type: constType,
		Node: stmt,
	}

	if err := a.symbols.Define(sym); err != nil {
		a.error(stmt.Token.Line, err.Error())
	}

	if stmt.Value != nil {
		valueType := a.analyzeExpression(stmt.Value)
		if !constType.IsCompatibleWith(valueType) {
			a.error(stmt.Token.Line, "type mismatch in constant declaration")
		}
	}
}

func (a *Analyzer) analyzeAssignmentStatement(stmt *parser.AssignmentStatement) {
	leftType := a.analyzeExpression(stmt.Left)
	rightType := a.analyzeExpression(stmt.Value)

	if !leftType.IsCompatibleWith(rightType) {
		a.error(stmt.Token.Line, "type mismatch in assignment: cannot assign %s to %s",
			rightType.String(), leftType.String())
	}
}

func (a *Analyzer) analyzeMultiAssignmentStatement(stmt *parser.MultiAssignmentStatement) {
	// Check for type assertion with ok pattern: value, ok = expr.(Type)
	if typeAssert, ok := stmt.Value.(*parser.TypeAssertionExpression); ok {
		if len(stmt.Targets) != 2 {
			a.error(stmt.Token.Line, "type assertion with ok pattern requires exactly 2 targets (value, ok)")
			return
		}
		// Analyze the type assertion value
		a.analyzeExpression(typeAssert.Value)
		// Analyze both targets
		for _, target := range stmt.Targets {
			a.analyzeExpression(target)
		}
		return
	}

	// Get the types of the right-hand side (should be a function call)
	call, ok := stmt.Value.(*parser.CallExpression)
	if !ok {
		a.error(stmt.Token.Line, "multiple assignment requires function call or type assertion on right side")
		return
	}

	// Check if this is an external Go method call - skip validation
	if _, isMember := call.Function.(*parser.MemberExpression); isMember {
		// External Go function call - analyze arguments and targets but don't validate return count
		for _, arg := range call.Arguments {
			a.analyzeExpression(arg)
		}
		for _, target := range stmt.Targets {
			a.analyzeExpression(target)
		}
		return
	}

	funcSym := a.resolveFunctionCall(call)
	if funcSym == nil || funcSym.Type == nil {
		return
	}

	if len(funcSym.Type.ReturnTypes) != len(stmt.Targets) {
		a.error(stmt.Token.Line, "wrong number of values in multiple assignment: expected %d, got %d",
			len(funcSym.Type.ReturnTypes), len(stmt.Targets))
		return
	}

	for i, target := range stmt.Targets {
		targetType := a.analyzeExpression(target)
		if !targetType.IsCompatibleWith(funcSym.Type.ReturnTypes[i]) {
			a.error(stmt.Token.Line, "type mismatch in multiple assignment at position %d", i+1)
		}
	}
}

func (a *Analyzer) analyzePrintStatement(stmt *parser.PrintStatement) {
	for _, val := range stmt.Values {
		a.analyzeExpression(val)
	}
}

func (a *Analyzer) analyzeInputStatement(stmt *parser.InputStatement) {
	if stmt.Prompt != nil {
		a.analyzeExpression(stmt.Prompt)
	}
	// Check that variable exists
	sym := a.symbols.Resolve(stmt.Variable.Value)
	if sym == nil {
		a.error(stmt.Token.Line, "undefined variable: %s", stmt.Variable.Value)
	}
}

func (a *Analyzer) analyzeIfStatement(stmt *parser.IfStatement) {
	condType := a.analyzeExpression(stmt.Condition)
	if condType.Kind != TypeBoolean && condType.Kind != TypeAny {
		a.error(stmt.Token.Line, "IF condition must be boolean, got %s", condType.String())
	}

	a.analyzeBlockStatement(stmt.Consequence)

	for _, elseif := range stmt.ElseIfs {
		condType := a.analyzeExpression(elseif.Condition)
		if condType.Kind != TypeBoolean && condType.Kind != TypeAny {
			a.error(elseif.Token.Line, "ELSEIF condition must be boolean")
		}
		a.analyzeBlockStatement(elseif.Consequence)
	}

	if stmt.Alternative != nil {
		a.analyzeBlockStatement(stmt.Alternative)
	}
}

func (a *Analyzer) analyzeForStatement(stmt *parser.ForStatement) {
	a.symbols.EnterScope("for")
	defer a.symbols.ExitScope()

	// Define loop variable
	sym := &Symbol{
		Name: stmt.Variable.Value,
		Kind: SymVariable,
		Type: IntegerType, // FOR loop variables are integers
		Node: stmt,
	}
	a.symbols.Define(sym)

	startType := a.analyzeExpression(stmt.Start)
	endType := a.analyzeExpression(stmt.End)

	if !startType.IsNumeric() || !endType.IsNumeric() {
		a.error(stmt.Token.Line, "FOR loop bounds must be numeric")
	}

	if stmt.Step != nil {
		stepType := a.analyzeExpression(stmt.Step)
		if !stepType.IsNumeric() {
			a.error(stmt.Token.Line, "FOR loop step must be numeric")
		}
	}

	a.analyzeBlockStatement(stmt.Body)
}

func (a *Analyzer) analyzeWhileStatement(stmt *parser.WhileStatement) {
	condType := a.analyzeExpression(stmt.Condition)
	if condType.Kind != TypeBoolean && condType.Kind != TypeAny {
		a.error(stmt.Token.Line, "WHILE condition must be boolean")
	}

	a.symbols.EnterScope("while")
	a.analyzeBlockStatement(stmt.Body)
	a.symbols.ExitScope()
}

func (a *Analyzer) analyzeDoLoopStatement(stmt *parser.DoLoopStatement) {
	if stmt.Condition != nil {
		condType := a.analyzeExpression(stmt.Condition)
		if condType.Kind != TypeBoolean && condType.Kind != TypeAny {
			a.error(stmt.Token.Line, "DO/LOOP condition must be boolean")
		}
	}

	a.symbols.EnterScope("do")
	a.analyzeBlockStatement(stmt.Body)
	a.symbols.ExitScope()
}

func (a *Analyzer) analyzeSelectStatement(stmt *parser.SelectStatement) {
	testType := a.analyzeExpression(stmt.TestExpr)

	for _, caseClause := range stmt.Cases {
		for _, val := range caseClause.Values {
			caseType := a.analyzeExpression(val)
			if !testType.IsCompatibleWith(caseType) {
				a.error(caseClause.Token.Line, "case value type mismatch")
			}
		}
		a.analyzeBlockStatement(caseClause.Body)
	}

	if stmt.Default != nil {
		a.analyzeBlockStatement(stmt.Default)
	}
}

func (a *Analyzer) analyzeSubStatement(stmt *parser.SubStatement) {
	a.symbols.EnterScope(stmt.Name.Value)
	defer a.symbols.ExitScope()

	// Define parameters
	for _, param := range stmt.Params {
		paramType := a.resolveTypeSpec(param.Type)
		sym := &Symbol{
			Name:    param.Name.Value,
			Kind:    SymParameter,
			Type:    paramType,
			IsByRef: param.ByRef,
		}
		a.symbols.Define(sym)
	}

	a.analyzeBlockStatement(stmt.Body)
}

func (a *Analyzer) analyzeFunctionStatement(stmt *parser.FunctionStatement) {
	a.symbols.EnterScope(stmt.Name.Value)
	defer a.symbols.ExitScope()

	// Define parameters
	for _, param := range stmt.Params {
		paramType := a.resolveTypeSpec(param.Type)
		sym := &Symbol{
			Name:    param.Name.Value,
			Kind:    SymParameter,
			Type:    paramType,
			IsByRef: param.ByRef,
		}
		a.symbols.Define(sym)
	}

	a.analyzeBlockStatement(stmt.Body)
}

func (a *Analyzer) analyzeMethodStatement(stmt *parser.MethodStatement) {
	// Get the receiver type name for scope naming
	var receiverTypeName string
	if stmt.ReceiverType.IsPointer {
		receiverTypeName = stmt.ReceiverType.ElementType.Name
	} else {
		receiverTypeName = stmt.ReceiverType.Name
	}

	scopeName := receiverTypeName + "." + stmt.Name.Value
	a.symbols.EnterScope(scopeName)
	defer a.symbols.ExitScope()

	// Define the receiver as a parameter
	receiverType := a.resolveTypeSpec(stmt.ReceiverType)
	receiverSym := &Symbol{
		Name: stmt.ReceiverName.Value,
		Kind: SymParameter,
		Type: receiverType,
	}
	a.symbols.Define(receiverSym)

	// Define parameters
	for _, param := range stmt.Params {
		paramType := a.resolveTypeSpec(param.Type)
		sym := &Symbol{
			Name:    param.Name.Value,
			Kind:    SymParameter,
			Type:    paramType,
			IsByRef: param.ByRef,
		}
		a.symbols.Define(sym)
	}

	a.analyzeBlockStatement(stmt.Body)
}

func (a *Analyzer) analyzeReturnStatement(stmt *parser.ReturnStatement) {
	for _, val := range stmt.Values {
		a.analyzeExpression(val)
	}
	// TODO: Check return types match function signature
}

func (a *Analyzer) analyzeLabelStatement(stmt *parser.LabelStatement) {
	sym := &Symbol{
		Name: stmt.Name,
		Kind: SymLabel,
		Node: stmt,
	}
	if err := a.symbols.CurrentScope.DefineLabel(stmt.Name, sym); err != nil {
		a.error(stmt.Token.Line, err.Error())
	}
}

func (a *Analyzer) analyzeSpawnStatement(stmt *parser.SpawnStatement) {
	a.analyzeExpression(stmt.Call)
}

func (a *Analyzer) analyzeSendStatement(stmt *parser.SendStatement) {
	valType := a.analyzeExpression(stmt.Value)
	chanType := a.analyzeExpression(stmt.Channel)

	if chanType.Kind != TypeChannel {
		a.error(stmt.Token.Line, "SEND target must be a channel")
		return
	}

	if !chanType.ElementType.IsCompatibleWith(valType) {
		a.error(stmt.Token.Line, "cannot send %s to channel of %s",
			valType.String(), chanType.ElementType.String())
	}
}

func (a *Analyzer) analyzeReceiveStatement(stmt *parser.ReceiveStatement) {
	varType := a.analyzeExpression(stmt.Variable)
	chanType := a.analyzeExpression(stmt.Channel)

	if chanType.Kind != TypeChannel {
		a.error(stmt.Token.Line, "RECEIVE source must be a channel")
		return
	}

	if !varType.IsCompatibleWith(chanType.ElementType) {
		a.error(stmt.Token.Line, "cannot receive %s from channel of %s",
			chanType.ElementType.String(), varType.String())
	}
}

func (a *Analyzer) analyzeBlockStatement(block *parser.BlockStatement) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		a.analyzeStatement(stmt)
	}
}

func (a *Analyzer) analyzeExpression(expr parser.Expression) *Type {
	if expr == nil {
		return VoidType
	}

	switch e := expr.(type) {
	case *parser.Identifier:
		return a.analyzeIdentifier(e)
	case *parser.IntegerLiteral:
		return IntegerType
	case *parser.FloatLiteral:
		return DoubleType
	case *parser.StringLiteral:
		return StringType
	case *parser.BooleanLiteral:
		return BooleanType
	case *parser.NilLiteral:
		return AnyType
	case *parser.JSONLiteral:
		return JSONType
	case *parser.ArrayLiteral:
		return a.analyzeArrayLiteral(e)
	case *parser.PrefixExpression:
		return a.analyzePrefixExpression(e)
	case *parser.InfixExpression:
		return a.analyzeInfixExpression(e)
	case *parser.CallExpression:
		return a.analyzeCallExpression(e)
	case *parser.IndexExpression:
		return a.analyzeIndexExpression(e)
	case *parser.MemberExpression:
		return a.analyzeMemberExpression(e)
	case *parser.AddressOfExpression:
		innerType := a.analyzeExpression(e.Value)
		return NewPointerType(innerType)
	case *parser.DereferenceExpression:
		innerType := a.analyzeExpression(e.Value)
		if innerType.Kind != TypePointer {
			a.error(e.Token.Line, "cannot dereference non-pointer type")
			return AnyType
		}
		return innerType.ElementType
	case *parser.MakeChanExpression:
		elemType := a.resolveTypeSpec(e.ChannelType)
		return NewChannelType(elemType)
	case *parser.ReceiveExpression:
		chanType := a.analyzeExpression(e.Channel)
		if chanType.Kind != TypeChannel {
			a.error(e.Token.Line, "cannot receive from non-channel type")
			return AnyType
		}
		return chanType.ElementType
	case *parser.TypeAssertionExpression:
		// Analyze the value being asserted
		a.analyzeExpression(e.Value)
		// Return the target type
		return a.resolveTypeSpec(e.TargetType)
	case *parser.StructLiteral:
		// Analyze struct literal fields
		for _, fieldExpr := range e.Fields {
			a.analyzeExpression(fieldExpr)
		}
		// Look up the struct type
		if a.types != nil {
			if t := a.types.Lookup(e.TypeName); t != nil {
				return t
			}
		}
		// Return a placeholder struct type
		return &Type{Kind: TypeStruct, Name: e.TypeName}
	default:
		return AnyType
	}
}

func (a *Analyzer) analyzeIdentifier(ident *parser.Identifier) *Type {
	sym := a.symbols.Resolve(ident.Value)
	if sym == nil {
		// Check if it's an imported package
		if a.symbols.GetImport(ident.Value) != nil {
			return AnyType // Package reference
		}
		a.error(ident.Token.Line, "undefined: %s", ident.Value)
		return AnyType
	}
	return sym.Type
}

func (a *Analyzer) analyzeArrayLiteral(arr *parser.ArrayLiteral) *Type {
	if len(arr.Elements) == 0 {
		return NewSliceType(AnyType)
	}

	elemType := a.analyzeExpression(arr.Elements[0])
	for _, elem := range arr.Elements[1:] {
		t := a.analyzeExpression(elem)
		if !elemType.IsCompatibleWith(t) {
			a.error(arr.Token.Line, "inconsistent array element types")
		}
	}

	return NewSliceType(elemType)
}

func (a *Analyzer) analyzePrefixExpression(expr *parser.PrefixExpression) *Type {
	rightType := a.analyzeExpression(expr.Right)

	switch expr.Operator {
	case "-":
		if !rightType.IsNumeric() {
			a.error(expr.Token.Line, "cannot negate non-numeric type")
		}
		return rightType
	case "NOT":
		if rightType.Kind != TypeBoolean {
			a.error(expr.Token.Line, "NOT requires boolean operand")
		}
		return BooleanType
	default:
		return rightType
	}
}

func (a *Analyzer) analyzeInfixExpression(expr *parser.InfixExpression) *Type {
	leftType := a.analyzeExpression(expr.Left)
	rightType := a.analyzeExpression(expr.Right)

	switch expr.Operator {
	case "+", "-", "*", "/", "\\":
		// Allow AnyType for external constants (e.g., walk.MsgBoxYesNo + walk.MsgBoxIconQuestion)
		if leftType.Kind == TypeAny || rightType.Kind == TypeAny {
			return AnyType
		}
		if !leftType.IsNumeric() || !rightType.IsNumeric() {
			// String concatenation
			if expr.Operator == "+" && leftType.Kind == TypeString && rightType.Kind == TypeString {
				return StringType
			}
			a.error(expr.Token.Line, "arithmetic operators require numeric operands")
			return AnyType
		}
		return PromoteNumeric(leftType, rightType)

	case "&":
		// String concatenation
		return StringType

	case "MOD":
		if !leftType.IsInteger() || !rightType.IsInteger() {
			a.error(expr.Token.Line, "MOD requires integer operands")
		}
		return IntegerType

	case "^":
		if !leftType.IsNumeric() || !rightType.IsNumeric() {
			a.error(expr.Token.Line, "exponentiation requires numeric operands")
		}
		return DoubleType

	case "=", "<>", "<", ">", "<=", ">=":
		return BooleanType

	case "AND", "OR", "XOR":
		if leftType.Kind != TypeBoolean || rightType.Kind != TypeBoolean {
			a.error(expr.Token.Line, "logical operators require boolean operands")
		}
		return BooleanType

	default:
		return AnyType
	}
}

func (a *Analyzer) analyzeCallExpression(call *parser.CallExpression) *Type {
	// Check if this is an external Go package function call
	if _, ok := call.Function.(*parser.MemberExpression); ok {
		// External Go function call - analyze arguments but don't check types
		for _, arg := range call.Arguments {
			a.analyzeExpression(arg)
		}
		return AnyType
	}

	// Check for Go builtin functions that need special handling
	if ident, ok := call.Function.(*parser.Identifier); ok {
		switch strings.ToUpper(ident.Value) {
		case "APPEND":
			// APPEND(slice, element...) returns the same slice type
			if len(call.Arguments) >= 1 {
				sliceType := a.analyzeExpression(call.Arguments[0])
				for _, arg := range call.Arguments[1:] {
					a.analyzeExpression(arg)
				}
				return sliceType
			}
			return AnyType
		case "LEN":
			// LEN works on strings, slices, arrays, maps, channels
			if len(call.Arguments) == 1 {
				argType := a.analyzeExpression(call.Arguments[0])
				switch argType.Kind {
				case TypeString, TypeSlice, TypeArray, TypeJSON, TypeChannel, TypeBytes:
					return IntegerType
				}
			}
			// Fall through to regular Len builtin for strings
		case "CAP":
			// CAP works on slices, arrays, channels
			if len(call.Arguments) == 1 {
				a.analyzeExpression(call.Arguments[0])
				return IntegerType
			}
			return AnyType
		case "MAKE":
			// MAKE(type, len, cap) - type determines return
			for _, arg := range call.Arguments {
				a.analyzeExpression(arg)
			}
			return AnyType
		case "COPY":
			// COPY(dst, src) returns int
			for _, arg := range call.Arguments {
				a.analyzeExpression(arg)
			}
			return IntegerType
		case "DELETE":
			// DELETE(map, key) returns nothing
			for _, arg := range call.Arguments {
				a.analyzeExpression(arg)
			}
			return VoidType
		case "CLOSE":
			// CLOSE(channel) returns nothing
			for _, arg := range call.Arguments {
				a.analyzeExpression(arg)
			}
			return VoidType
		case "PANIC", "RECOVER", "STRING", "RUNE", "BYTE":
			// Handle other Go builtins
			for _, arg := range call.Arguments {
				a.analyzeExpression(arg)
			}
			return AnyType
		case "NEW":
			// NEW takes a type name, not a variable expression
			// The argument should be resolved as a type, not analyzed as an expression
			if len(call.Arguments) == 1 {
				if ident, ok := call.Arguments[0].(*parser.Identifier); ok {
					// Check if it's a known type
					if t := a.types.Lookup(ident.Value); t != nil {
						return NewPointerType(t)
					}
					// Check if it's a built-in type
					if t := TypeFromName(ident.Value); t != nil {
						return NewPointerType(t)
					}
					// Otherwise, assume it's an external type
					return AnyType
				}
			}
			return AnyType
		}
	}

	sym := a.resolveFunctionCall(call)
	if sym == nil {
		return AnyType
	}

	if sym.Type == nil {
		return AnyType
	}

	// Check argument count
	if sym.Type.Variadic {
		// Variadic functions require at least the defined params
		if len(call.Arguments) < len(sym.Type.ParamTypes) {
			a.error(call.Token.Line, "wrong number of arguments: expected at least %d, got %d",
				len(sym.Type.ParamTypes), len(call.Arguments))
		}
	} else {
		// Non-variadic functions require exact match
		if len(call.Arguments) != len(sym.Type.ParamTypes) {
			a.error(call.Token.Line, "wrong number of arguments: expected %d, got %d",
				len(sym.Type.ParamTypes), len(call.Arguments))
		}
	}

	// Check argument types (only for defined params, not variadic args)
	for i, arg := range call.Arguments {
		if i >= len(sym.Type.ParamTypes) {
			// For variadic functions, still analyze extra args but don't type-check them
			if sym.Type.Variadic {
				a.analyzeExpression(arg)
			}
			break
		}
		argType := a.analyzeExpression(arg)
		if !sym.Type.ParamTypes[i].IsCompatibleWith(argType) {
			a.error(call.Token.Line, "argument %d type mismatch", i+1)
		}
	}

	if len(sym.Type.ReturnTypes) > 0 {
		return sym.Type.ReturnTypes[0]
	}
	return VoidType
}

func (a *Analyzer) resolveFunctionCall(call *parser.CallExpression) *Symbol {
	switch fn := call.Function.(type) {
	case *parser.Identifier:
		sym := a.symbols.Resolve(fn.Value)
		if sym == nil {
			a.error(call.Token.Line, "undefined function: %s", fn.Value)
			return nil
		}
		return sym
	case *parser.MemberExpression:
		// Package.Function call
		// For Go package calls, we return a placeholder
		return &Symbol{
			Name: fn.Member.Value,
			Kind: SymFunction,
			Type: NewFunctionType(nil, []*Type{AnyType}),
		}
	default:
		return nil
	}
}

func (a *Analyzer) analyzeIndexExpression(expr *parser.IndexExpression) *Type {
	leftType := a.analyzeExpression(expr.Left)

	// Analyze index if present
	if expr.Index != nil {
		indexType := a.analyzeExpression(expr.Index)
		if !indexType.IsInteger() {
			a.error(expr.Token.Line, "array index must be integer")
		}
	}

	// Analyze end index if present (for slice operations)
	if expr.End != nil {
		endType := a.analyzeExpression(expr.End)
		if !endType.IsInteger() {
			a.error(expr.Token.Line, "slice end index must be integer")
		}
	}

	// For slice operations [start:end], return the same slice/array type
	if expr.IsSlice {
		switch leftType.Kind {
		case TypeArray, TypeSlice:
			return NewSliceType(leftType.ElementType)
		case TypeString:
			return StringType
		case TypeBytes:
			return BytesType
		default:
			a.error(expr.Token.Line, "cannot slice type %s", leftType.String())
			return AnyType
		}
	}

	// For regular indexing [index], return the element type
	switch leftType.Kind {
	case TypeArray, TypeSlice:
		return leftType.ElementType
	case TypeString:
		return IntegerType // character code
	case TypeBytes:
		return IntegerType // byte value
	case TypeJSON:
		return AnyType
	default:
		a.error(expr.Token.Line, "cannot index type %s", leftType.String())
		return AnyType
	}
}

func (a *Analyzer) analyzeMemberExpression(expr *parser.MemberExpression) *Type {
	objType := a.analyzeExpression(expr.Object)

	// Check for package access
	if ident, ok := expr.Object.(*parser.Identifier); ok {
		if a.symbols.GetImport(ident.Value) != nil {
			// This is a package member access
			return AnyType
		}
	}

	if objType.Kind == TypeJSON {
		return AnyType
	}

	// Handle struct field access
	if objType.Kind == TypeStruct {
		for _, field := range objType.Fields {
			if strings.EqualFold(field.Name, expr.Member.Value) {
				return field.Type
			}
		}
		a.error(expr.Token.Line, "type %s has no field %s", objType.Name, expr.Member.Value)
		return AnyType
	}

	// Handle pointer to struct field access
	if objType.Kind == TypePointer && objType.ElementType != nil && objType.ElementType.Kind == TypeStruct {
		structType := objType.ElementType
		for _, field := range structType.Fields {
			if strings.EqualFold(field.Name, expr.Member.Value) {
				return field.Type
			}
		}
		a.error(expr.Token.Line, "type %s has no field %s", structType.Name, expr.Member.Value)
		return AnyType
	}

	// Handle external Go type field/method access
	// We can't validate these at compile time, let Go handle it
	if objType.Kind == TypeExternal {
		return AnyType
	}

	// Handle pointer to external type field access
	if objType.Kind == TypePointer && objType.ElementType != nil && objType.ElementType.Kind == TypeExternal {
		return AnyType
	}

	// Allow member access on AnyType (for chained access on external types)
	if objType.Kind == TypeAny {
		return AnyType
	}

	a.error(expr.Token.Line, "cannot access member of type %s", objType.String())
	return AnyType
}

// GetAllFunctions returns all declared functions and subs
func (a *Analyzer) GetAllFunctions() []*Symbol {
	var funcs []*Symbol
	for _, sym := range a.symbols.GlobalScope.AllSymbols() {
		if sym.Kind == SymFunction || sym.Kind == SymSub {
			funcs = append(funcs, sym)
		}
	}
	return funcs
}

// GetMainSub returns the Main sub if it exists
func (a *Analyzer) GetMainSub() *Symbol {
	sym := a.symbols.GlobalScope.Resolve("MAIN")
	if sym != nil && sym.Kind == SymSub {
		return sym
	}
	return nil
}

// HasMain checks if the program has a Main sub
func (a *Analyzer) HasMain() bool {
	return a.GetMainSub() != nil
}
