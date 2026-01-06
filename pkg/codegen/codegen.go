package codegen

import (
	"fmt"
	"strings"

	"github.com/zditech/dbasic/pkg/analyzer"
	"github.com/zditech/dbasic/pkg/parser"
)

// Generator generates Go code from a DBasic AST
type Generator struct {
	program    *parser.Program
	symbols    *analyzer.SymbolTable
	output     strings.Builder
	indent     int
	imports    map[string]bool
	hasMain    bool
	labelCount int
	debugMode  bool
	sourceFile string
}

// New creates a new code generator
func New(program *parser.Program, symbols *analyzer.SymbolTable) *Generator {
	return &Generator{
		program: program,
		symbols: symbols,
		imports: make(map[string]bool),
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

// Generate generates Go source code
func (g *Generator) Generate() string {
	// Collect imports from explicit IMPORT statements
	g.collectImports()

	// Pre-scan for additional required imports
	g.scanForRequiredImports()

	// Check for Main sub
	mainSym := g.symbols.GlobalScope.Resolve("Main")
	g.hasMain = mainSym != nil

	// Generate package declaration
	g.writeLine("package main")
	g.writeLine("")

	// Generate imports
	g.generateImports()

	// Generate global variables
	g.generateGlobalVariables()

	// Generate functions and subs
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
		g.imports["bufio"] = true
		g.imports["os"] = true
	}
}

func (g *Generator) scanBlockForImports(block *parser.BlockStatement) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *parser.InputStatement:
			g.imports["bufio"] = true
			g.imports["os"] = true
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
			g.imports["math"] = true
		}
	case *parser.ExpressionStatement:
		if s.Expression != nil && g.exprNeedsMath(s.Expression) {
			g.imports["math"] = true
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
		return g.exprNeedsMath(e.Left) || g.exprNeedsMath(e.Index)
	}
	return false
}

func (g *Generator) collectImports() {
	// Always include fmt for PRINT
	g.imports["fmt"] = true

	// Add user imports
	for _, imp := range g.symbols.AllImports() {
		g.imports[imp.Path] = true
	}
}

func (g *Generator) generateImports() {
	if len(g.imports) == 0 {
		return
	}

	g.writeLine("import (")
	g.indent++
	for imp := range g.imports {
		g.writeLine(fmt.Sprintf(`"%s"`, imp))
	}
	g.indent--
	g.writeLine(")")
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
	g.generateBlockStatement(stmt.Body)
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
	g.generateBlockStatement(stmt.Body)
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
	g.imports["bufio"] = true
	g.imports["os"] = true

	varName := g.toGoIdent(stmt.Variable.Value)

	if stmt.Prompt != nil {
		g.writeLine(fmt.Sprintf("fmt.Print(%s)", g.exprToGo(stmt.Prompt)))
	}

	g.writeLine("_reader := bufio.NewReader(os.Stdin)")
	g.writeLine(fmt.Sprintf("%s, _ = _reader.ReadString('\\n')", varName))
	g.writeLine(fmt.Sprintf("%s = %s[:len(%s)-1]", varName, varName, varName))
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
	case *parser.PrefixExpression:
		return g.prefixExprToGo(e)
	case *parser.InfixExpression:
		return g.infixExprToGo(e)
	case *parser.CallExpression:
		return g.callExprToGo(e)
	case *parser.IndexExpression:
		return fmt.Sprintf("%s[%s]", g.exprToGo(e.Left), g.exprToGo(e.Index))
	case *parser.MemberExpression:
		return fmt.Sprintf("%s.%s", g.exprToGo(e.Object), g.toGoIdent(e.Member.Value))
	case *parser.AddressOfExpression:
		return fmt.Sprintf("&%s", g.exprToGo(e.Value))
	case *parser.DereferenceExpression:
		return fmt.Sprintf("*%s", g.exprToGo(e.Value))
	case *parser.MakeChanExpression:
		chanType := g.typeSpecToGo(e.ChannelType)
		if e.Size != nil {
			return fmt.Sprintf("make(chan %s, %s)", chanType, g.exprToGo(e.Size))
		}
		return fmt.Sprintf("make(chan %s)", chanType)
	case *parser.ReceiveExpression:
		return fmt.Sprintf("<-%s", g.exprToGo(e.Channel))
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

	var elems []string
	for _, e := range lit.Elements {
		elems = append(elems, g.exprToGo(e))
	}
	return fmt.Sprintf("[]interface{}{%s}", strings.Join(elems, ", "))
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
		g.imports["math"] = true
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

	switch strings.ToUpper(spec.Name) {
	case "INTEGER":
		return "int32"
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
	default:
		return "interface{}"
	}
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
