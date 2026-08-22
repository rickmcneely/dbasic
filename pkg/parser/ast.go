package parser

import (
	"strconv"
	"strings"

	"github.com/zditech/dbasic/pkg/lexer"
)

// Node is the interface for all AST nodes
type Node interface {
	TokenLiteral() string
	String() string
}

// Statement is the interface for all statement nodes
type Statement interface {
	Node
	statementNode()
}

// Expression is the interface for all expression nodes
type Expression interface {
	Node
	expressionNode()
}

// Program is the root node of the AST
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var sb strings.Builder
	for _, s := range p.Statements {
		sb.WriteString(s.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

// Channel direction for a CHAN OF type. A bare `CHAN OF T` is
// bidirectional; `RECEIVE CHAN OF T` is receive-only (Go `<-chan T`) and
// `SEND CHAN OF T` is send-only (Go `chan<- T`). These matter for interop
// with Go APIs that hand back directional channels (e.g. a window-size
// channel typed `<-chan Window`).
const (
	ChanBoth = iota // CHAN OF T        -> chan T
	ChanRecv        // RECEIVE CHAN OF T -> <-chan T
	ChanSend        // SEND CHAN OF T    -> chan<- T
)

// TypeSpec represents a type specification
type TypeSpec struct {
	Token       lexer.Token // The type token
	Name        string      // Base type name (INTEGER, STRING, etc.)
	IsPointer   bool        // POINTER TO X
	IsChannel   bool        // CHAN OF X
	ChanDir     int         // Channel direction (ChanBoth/ChanRecv/ChanSend)
	ElementType *TypeSpec   // For POINTER TO and CHAN OF (also map value type when IsMap)
	IsArray     bool        // Array type
	ArraySize   Expression  // Array size expression (can be nil for dynamic)
	IsMap       bool        // MAP OF K TO V
	KeyType     *TypeSpec   // Map key type (when IsMap)
}

func (t *TypeSpec) TokenLiteral() string { return t.Token.Literal }
func (t *TypeSpec) String() string {
	if t.IsPointer {
		return "POINTER TO " + t.ElementType.String()
	}
	if t.IsChannel {
		switch t.ChanDir {
		case ChanRecv:
			return "RECEIVE CHAN OF " + t.ElementType.String()
		case ChanSend:
			return "SEND CHAN OF " + t.ElementType.String()
		}
		return "CHAN OF " + t.ElementType.String()
	}
	if t.IsMap {
		return "MAP OF " + t.KeyType.String() + " TO " + t.ElementType.String()
	}
	if t.IsArray {
		if t.ArraySize != nil {
			return t.Name + "(" + t.ArraySize.String() + ")"
		}
		return t.Name + "()"
	}
	return t.Name
}

// --- Statements ---

// ImportStatement represents an IMPORT statement
type ImportStatement struct {
	Token   lexer.Token // IMPORT token
	Package string      // Package path
	Alias   string      // Optional alias
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStatement) String() string {
	if is.Alias != "" {
		return "IMPORT " + is.Package + " AS " + is.Alias
	}
	return "IMPORT " + is.Package
}

// DimStatement represents a DIM variable declaration
type DimStatement struct {
	Token     lexer.Token // DIM token
	Name      *Identifier
	Type      *TypeSpec
	Value     Expression // Optional initial value
	ArraySize Expression // For array declarations
	IsStatic  bool       // true for `STATIC x AS T`: persists across sub calls
}

// WithStatement represents `WITH expr ... END WITH`. Inside the body,
// `.field` is shorthand for `expr.field`. The receiver expression is
// stored once and assigned to a synthesized local at codegen time so
// side-effecting receivers aren't re-evaluated per shorthand reference.
type WithStatement struct {
	Token     lexer.Token
	Receiver  Expression  // The expression after WITH
	Synthetic *Identifier // Auto-generated local name used as the receiver inside the block
	Body      *BlockStatement
}

func (ws *WithStatement) statementNode()       {}
func (ws *WithStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WithStatement) String() string {
	return "WITH " + ws.Receiver.String() + "\n" + ws.Body.String() + "END WITH"
}

// LineInputStatement represents `LINE INPUT [prompt$,] var$`. Reads a
// whole line from stdin including embedded spaces. The prompt is optional.
type LineInputStatement struct {
	Token  lexer.Token
	Prompt Expression  // optional string literal or expression
	Var    *Identifier
}

func (li *LineInputStatement) statementNode()       {}
func (li *LineInputStatement) TokenLiteral() string { return li.Token.Literal }
func (li *LineInputStatement) String() string {
	if li.Prompt != nil {
		return "LINE INPUT " + li.Prompt.String() + ", " + li.Var.String()
	}
	return "LINE INPUT " + li.Var.String()
}

// OptionStatement represents an `OPTION` pragma at file scope.
// Currently supports OPTION EXPLICIT (no-op since DBasic always
// requires DIM) and OPTION BASE 0|1 (sets default array lower bound).
type OptionStatement struct {
	Token lexer.Token
	Kind  string // "EXPLICIT" or "BASE"
	Value int    // for OPTION BASE: 0 or 1
}

func (os *OptionStatement) statementNode()       {}
func (os *OptionStatement) TokenLiteral() string { return os.Token.Literal }
func (os *OptionStatement) String() string {
	if os.Kind == "BASE" {
		return "OPTION BASE " + strconv.Itoa(os.Value)
	}
	return "OPTION " + os.Kind
}

// SharedStatement represents `SHARED name1, name2, ...` inside a sub or
// function. In QB this opted the listed module-level names into the sub's
// scope. DBasic already resolves identifiers up the scope chain, so this
// statement is parsed and accepted but generates no Go code.
type SharedStatement struct {
	Token lexer.Token
	Names []*Identifier
}

func (ss *SharedStatement) statementNode()       {}
func (ss *SharedStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SharedStatement) String() string {
	parts := make([]string, len(ss.Names))
	for i, n := range ss.Names {
		parts[i] = n.String()
	}
	return "SHARED " + strings.Join(parts, ", ")
}

// ReDimStatement represents `REDIM [PRESERVE] x(n) AS T`, which resizes
// an existing slice. Without PRESERVE the slice's contents are discarded;
// with PRESERVE the existing elements are copied into the new slice.
type ReDimStatement struct {
	Token     lexer.Token
	Name      *Identifier
	ArraySize Expression
	Type      *TypeSpec
	Preserve  bool
}

func (rd *ReDimStatement) statementNode()       {}
func (rd *ReDimStatement) TokenLiteral() string { return rd.Token.Literal }
func (rd *ReDimStatement) String() string {
	var sb strings.Builder
	sb.WriteString("REDIM ")
	if rd.Preserve {
		sb.WriteString("PRESERVE ")
	}
	sb.WriteString(rd.Name.String())
	sb.WriteString("(")
	sb.WriteString(rd.ArraySize.String())
	sb.WriteString(") AS ")
	sb.WriteString(rd.Type.String())
	return sb.String()
}

func (ds *DimStatement) statementNode()       {}
func (ds *DimStatement) TokenLiteral() string { return ds.Token.Literal }
func (ds *DimStatement) String() string {
	var sb strings.Builder
	sb.WriteString("DIM ")
	sb.WriteString(ds.Name.String())
	if ds.ArraySize != nil {
		sb.WriteString("(")
		sb.WriteString(ds.ArraySize.String())
		sb.WriteString(")")
	}
	sb.WriteString(" AS ")
	sb.WriteString(ds.Type.String())
	if ds.Value != nil {
		sb.WriteString(" = ")
		sb.WriteString(ds.Value.String())
	}
	return sb.String()
}

// LetStatement represents a LET statement with type inference
type LetStatement struct {
	Token lexer.Token // LET token
	Name  *Identifier
	Value Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	return "LET " + ls.Name.String() + " = " + ls.Value.String()
}

// ConstStatement represents a CONST declaration
type ConstStatement struct {
	Token lexer.Token
	Name  *Identifier
	Type  *TypeSpec
	Value Expression
}

func (cs *ConstStatement) statementNode()       {}
func (cs *ConstStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStatement) String() string {
	return "CONST " + cs.Name.String() + " AS " + cs.Type.String() + " = " + cs.Value.String()
}

// AssignmentStatement represents a variable assignment
type AssignmentStatement struct {
	Token lexer.Token
	Left  Expression // Can be Identifier, IndexExpression, or MemberExpression
	Value Expression
}

func (as *AssignmentStatement) statementNode()       {}
func (as *AssignmentStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignmentStatement) String() string {
	return as.Left.String() + " = " + as.Value.String()
}

// MultiAssignmentStatement represents multiple return value assignment
type MultiAssignmentStatement struct {
	Token   lexer.Token
	Targets []Expression
	Value   Expression // Usually a CallExpression
	// LastTargetIsError is set by the analyzer when the final target
	// resolves to type ERROR. Codegen uses it to emit ONERR auto-checks.
	LastTargetIsError bool
}

func (ms *MultiAssignmentStatement) statementNode()       {}
func (ms *MultiAssignmentStatement) TokenLiteral() string { return ms.Token.Literal }
func (ms *MultiAssignmentStatement) String() string {
	var targets []string
	for _, t := range ms.Targets {
		targets = append(targets, t.String())
	}
	return strings.Join(targets, ", ") + " = " + ms.Value.String()
}

// PrintStatement represents a PRINT statement
type PrintStatement struct {
	Token      lexer.Token
	Values     []Expression
	Separators []string // ";" or "," between values
}

func (ps *PrintStatement) statementNode()       {}
func (ps *PrintStatement) TokenLiteral() string { return ps.Token.Literal }
func (ps *PrintStatement) String() string {
	var sb strings.Builder
	sb.WriteString("PRINT ")
	for i, v := range ps.Values {
		sb.WriteString(v.String())
		if i < len(ps.Separators) {
			sb.WriteString(ps.Separators[i])
		}
	}
	return sb.String()
}

// InputStatement represents an INPUT statement
type InputStatement struct {
	Token    lexer.Token
	Prompt   Expression // Optional prompt string
	Variable *Identifier
}

func (is *InputStatement) statementNode()       {}
func (is *InputStatement) TokenLiteral() string { return is.Token.Literal }
func (is *InputStatement) String() string {
	if is.Prompt != nil {
		return "INPUT " + is.Prompt.String() + "; " + is.Variable.String()
	}
	return "INPUT " + is.Variable.String()
}

// IfStatement represents an IF/THEN/ELSE/ENDIF block
type IfStatement struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockStatement
	ElseIfs     []*ElseIfClause
	Alternative *BlockStatement // ELSE block
}

type ElseIfClause struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockStatement
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IfStatement) String() string {
	var sb strings.Builder
	sb.WriteString("IF ")
	sb.WriteString(is.Condition.String())
	sb.WriteString(" THEN\n")
	sb.WriteString(is.Consequence.String())
	for _, elif := range is.ElseIfs {
		sb.WriteString("ELSEIF ")
		sb.WriteString(elif.Condition.String())
		sb.WriteString(" THEN\n")
		sb.WriteString(elif.Consequence.String())
	}
	if is.Alternative != nil {
		sb.WriteString("ELSE\n")
		sb.WriteString(is.Alternative.String())
	}
	sb.WriteString("ENDIF")
	return sb.String()
}

// ForStatement represents a FOR/TO/STEP/NEXT loop
type ForStatement struct {
	Token    lexer.Token
	Variable *Identifier
	Start    Expression
	End      Expression
	Step     Expression // Optional, defaults to 1
	Body     *BlockStatement
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) String() string {
	var sb strings.Builder
	sb.WriteString("FOR ")
	sb.WriteString(fs.Variable.String())
	sb.WriteString(" = ")
	sb.WriteString(fs.Start.String())
	sb.WriteString(" TO ")
	sb.WriteString(fs.End.String())
	if fs.Step != nil {
		sb.WriteString(" STEP ")
		sb.WriteString(fs.Step.String())
	}
	sb.WriteString("\n")
	sb.WriteString(fs.Body.String())
	sb.WriteString("NEXT")
	return sb.String()
}

// WhileStatement represents a WHILE/WEND loop
type WhileStatement struct {
	Token     lexer.Token
	Condition Expression
	Body      *BlockStatement
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStatement) String() string {
	var sb strings.Builder
	sb.WriteString("WHILE ")
	sb.WriteString(ws.Condition.String())
	sb.WriteString("\n")
	sb.WriteString(ws.Body.String())
	sb.WriteString("WEND")
	return sb.String()
}

// DoLoopStatement represents a DO/LOOP with optional WHILE/UNTIL
type DoLoopStatement struct {
	Token         lexer.Token
	Condition     Expression
	Body          *BlockStatement
	IsWhile       bool // true for WHILE, false for UNTIL
	IsPreCondition bool // true if condition is at DO, false if at LOOP
}

func (dl *DoLoopStatement) statementNode()       {}
func (dl *DoLoopStatement) TokenLiteral() string { return dl.Token.Literal }
func (dl *DoLoopStatement) String() string {
	var sb strings.Builder
	sb.WriteString("DO")
	if dl.IsPreCondition && dl.Condition != nil {
		if dl.IsWhile {
			sb.WriteString(" WHILE ")
		} else {
			sb.WriteString(" UNTIL ")
		}
		sb.WriteString(dl.Condition.String())
	}
	sb.WriteString("\n")
	sb.WriteString(dl.Body.String())
	sb.WriteString("LOOP")
	if !dl.IsPreCondition && dl.Condition != nil {
		if dl.IsWhile {
			sb.WriteString(" WHILE ")
		} else {
			sb.WriteString(" UNTIL ")
		}
		sb.WriteString(dl.Condition.String())
	}
	return sb.String()
}

// SelectStatement represents a SELECT CASE statement
type SelectStatement struct {
	Token    lexer.Token
	TestExpr Expression
	Cases    []*CaseClause
	Default  *BlockStatement
}

type CaseClause struct {
	Token  lexer.Token
	Values []Expression
	Body   *BlockStatement
}

func (ss *SelectStatement) statementNode()       {}
func (ss *SelectStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SelectStatement) String() string {
	var sb strings.Builder
	sb.WriteString("SELECT CASE ")
	sb.WriteString(ss.TestExpr.String())
	sb.WriteString("\n")
	for _, c := range ss.Cases {
		sb.WriteString("CASE ")
		for i, v := range c.Values {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(v.String())
		}
		sb.WriteString("\n")
		sb.WriteString(c.Body.String())
	}
	if ss.Default != nil {
		sb.WriteString("CASE ELSE\n")
		sb.WriteString(ss.Default.String())
	}
	sb.WriteString("END SELECT")
	return sb.String()
}

// OnErrAction is the kind of handler installed by an ONERR statement.
type OnErrAction int

const (
	OnErrGoto   OnErrAction = iota // ONERR GOTO label
	OnErrGoFunc                    // ONERR GOFUNC funcName
	OnErrClear                     // ONERR GOTO 0  (disable handler)
)

// OnErrStatement installs (or clears) a per-function error-handling
// directive. After this statement, subsequent multi-return assignments
// whose final value is of type ERROR have an automatic
//
//	if err != nil { goto Label }                          // OnErrGoto
//	if err != nil { Handler(err); return [zero values] }  // OnErrGoFunc
//
// emitted by codegen. ONERR is purely lexical/per-function — it does not
// cross sub/function boundaries and does not introduce any defer/recover
// machinery (no try/catch). `ONERR GOTO 0` clears the active handler.
type OnErrStatement struct {
	Token  lexer.Token
	Action OnErrAction
	Target *Identifier // label name (Goto), function name (GoFunc), nil for Clear
}

func (oe *OnErrStatement) statementNode()       {}
func (oe *OnErrStatement) TokenLiteral() string { return oe.Token.Literal }
func (oe *OnErrStatement) String() string {
	switch oe.Action {
	case OnErrGoto:
		return "ONERR GOTO " + oe.Target.Value
	case OnErrGoFunc:
		return "ONERR GOFUNC " + oe.Target.Value
	case OnErrClear:
		return "ONERR GOTO 0"
	}
	return "ONERR"
}

// GotoStatement represents a GOTO statement
type GotoStatement struct {
	Token lexer.Token
	Label string
}

func (gs *GotoStatement) statementNode()       {}
func (gs *GotoStatement) TokenLiteral() string { return gs.Token.Literal }
func (gs *GotoStatement) String() string       { return "GOTO " + gs.Label }

// LabelStatement represents a label definition
type LabelStatement struct {
	Token lexer.Token
	Name  string
}

func (ls *LabelStatement) statementNode()       {}
func (ls *LabelStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LabelStatement) String() string       { return ls.Name + ":" }

// ReturnStatement represents a RETURN statement
type ReturnStatement struct {
	Token  lexer.Token
	Values []Expression // Multiple return values
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	if len(rs.Values) == 0 {
		return "RETURN"
	}
	var vals []string
	for _, v := range rs.Values {
		vals = append(vals, v.String())
	}
	return "RETURN " + strings.Join(vals, ", ")
}

// ExitStatement represents an EXIT statement
type ExitStatement struct {
	Token    lexer.Token
	ExitType string // FOR, WHILE, DO, SUB, FUNCTION
}

func (es *ExitStatement) statementNode()       {}
func (es *ExitStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExitStatement) String() string       { return "EXIT " + es.ExitType }

// ContinueStatement represents a CONTINUE statement, which skips the rest of
// the current loop iteration and goes straight on to the next one.
//
// The loop kind (CONTINUE FOR / WHILE / DO) is optional and, as with EXIT,
// purely documentation: CONTINUE always affects the innermost enclosing loop.
type ContinueStatement struct {
	Token    lexer.Token
	LoopType string // FOR, WHILE, DO, or "" when written bare
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string {
	if cs.LoopType == "" {
		return "CONTINUE"
	}
	return "CONTINUE " + cs.LoopType
}

// SubStatement represents a SUB definition
type SubStatement struct {
	Token  lexer.Token
	Name   *Identifier
	Params []*Parameter
	Body   *BlockStatement
}

type Parameter struct {
	Name   *Identifier
	Type   *TypeSpec
	ByRef  bool // Pass by reference
}

func (ss *SubStatement) statementNode()       {}
func (ss *SubStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SubStatement) String() string {
	var sb strings.Builder
	sb.WriteString("SUB ")
	sb.WriteString(ss.Name.String())
	sb.WriteString("(")
	for i, p := range ss.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		if p.ByRef {
			sb.WriteString("BYREF ")
		}
		sb.WriteString(p.Name.String())
		sb.WriteString(" AS ")
		sb.WriteString(p.Type.String())
	}
	sb.WriteString(")\n")
	sb.WriteString(ss.Body.String())
	sb.WriteString("END SUB")
	return sb.String()
}

// FunctionStatement represents a FUNCTION definition
type FunctionStatement struct {
	Token       lexer.Token
	Name        *Identifier
	Params      []*Parameter
	ReturnTypes []*TypeSpec // Multiple return types
	Body        *BlockStatement
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *FunctionStatement) String() string {
	var sb strings.Builder
	sb.WriteString("FUNCTION ")
	sb.WriteString(fs.Name.String())
	sb.WriteString("(")
	for i, p := range fs.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		if p.ByRef {
			sb.WriteString("BYREF ")
		}
		sb.WriteString(p.Name.String())
		sb.WriteString(" AS ")
		sb.WriteString(p.Type.String())
	}
	sb.WriteString(") AS ")
	if len(fs.ReturnTypes) > 1 {
		sb.WriteString("(")
		for i, t := range fs.ReturnTypes {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(t.String())
		}
		sb.WriteString(")")
	} else if len(fs.ReturnTypes) == 1 {
		sb.WriteString(fs.ReturnTypes[0].String())
	}
	sb.WriteString("\n")
	sb.WriteString(fs.Body.String())
	sb.WriteString("END FUNCTION")
	return sb.String()
}

// MethodStatement represents a method definition with receiver
// FUNCTION (recv AS POINTER TO Type) Name(params) AS ReturnType
type MethodStatement struct {
	Token        lexer.Token
	ReceiverName *Identifier
	ReceiverType *TypeSpec
	Name         *Identifier
	Params       []*Parameter
	ReturnTypes  []*TypeSpec
	Body         *BlockStatement
}

func (ms *MethodStatement) statementNode()       {}
func (ms *MethodStatement) TokenLiteral() string { return ms.Token.Literal }
func (ms *MethodStatement) String() string {
	var sb strings.Builder
	sb.WriteString("FUNCTION (")
	sb.WriteString(ms.ReceiverName.String())
	sb.WriteString(" AS ")
	sb.WriteString(ms.ReceiverType.String())
	sb.WriteString(") ")
	sb.WriteString(ms.Name.String())
	sb.WriteString("(")
	for i, p := range ms.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.Name.String())
		sb.WriteString(" AS ")
		sb.WriteString(p.Type.String())
	}
	sb.WriteString(")")
	if len(ms.ReturnTypes) > 0 {
		sb.WriteString(" AS ")
		if len(ms.ReturnTypes) > 1 {
			sb.WriteString("(")
			for i, t := range ms.ReturnTypes {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(t.String())
			}
			sb.WriteString(")")
		} else {
			sb.WriteString(ms.ReturnTypes[0].String())
		}
	}
	sb.WriteString("\n")
	sb.WriteString(ms.Body.String())
	sb.WriteString("END FUNCTION")
	return sb.String()
}

// TypeStatement represents a TYPE definition (struct)
// EmbeddedDeclaration represents an embedded type in a TYPE definition
type EmbeddedDeclaration struct {
	Token    lexer.Token
	TypeName string // e.g., "walk.TableModelBase"
}

type TypeStatement struct {
	Token      lexer.Token
	Name       *Identifier
	Implements string // Go interface this type implements (e.g., "tea.Model")
	Embedded   []*EmbeddedDeclaration
	Fields     []*FieldDeclaration
}

func (ts *TypeStatement) statementNode()       {}
func (ts *TypeStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *TypeStatement) String() string {
	var sb strings.Builder
	sb.WriteString("TYPE ")
	sb.WriteString(ts.Name.String())
	if ts.Implements != "" {
		sb.WriteString(" IMPLEMENTS ")
		sb.WriteString(ts.Implements)
	}
	sb.WriteString("\n")
	for _, e := range ts.Embedded {
		sb.WriteString("    EMBED ")
		sb.WriteString(e.TypeName)
		sb.WriteString("\n")
	}
	for _, f := range ts.Fields {
		sb.WriteString("    ")
		sb.WriteString(f.String())
		sb.WriteString("\n")
	}
	sb.WriteString("END TYPE")
	return sb.String()
}

// FieldDeclaration represents a field in a TYPE definition
type FieldDeclaration struct {
	Token lexer.Token
	Name  *Identifier
	Type  *TypeSpec
}

func (fd *FieldDeclaration) statementNode()       {}
func (fd *FieldDeclaration) TokenLiteral() string { return fd.Token.Literal }
func (fd *FieldDeclaration) String() string {
	return "DIM " + fd.Name.String() + " AS " + fd.Type.String()
}

// SpawnStatement represents a SPAWN statement (goroutine)
type SpawnStatement struct {
	Token lexer.Token
	Call  *CallExpression
}

func (ss *SpawnStatement) statementNode()       {}
func (ss *SpawnStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SpawnStatement) String() string       { return "SPAWN " + ss.Call.String() }

// DeferStatement represents a DEFER statement (deferred call, maps to Go's defer)
type DeferStatement struct {
	Token lexer.Token
	Call  *CallExpression
}

func (ds *DeferStatement) statementNode()       {}
func (ds *DeferStatement) TokenLiteral() string { return ds.Token.Literal }
func (ds *DeferStatement) String() string       { return "DEFER " + ds.Call.String() }

// FunctionLiteral represents an anonymous function expression:
//   FUNCTION(args) AS Type ... END FUNCTION
//   SUB(args) ... END SUB
// Maps to a Go func literal. Captures variables from enclosing scopes naturally
// (Go closures handle the capture).
type FunctionLiteral struct {
	Token       lexer.Token  // FUNCTION or SUB
	IsSub       bool         // true if SUB (no return type), false if FUNCTION
	Params      []*Parameter
	ReturnTypes []*TypeSpec
	Body        *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var sb strings.Builder
	if fl.IsSub {
		sb.WriteString("SUB(")
	} else {
		sb.WriteString("FUNCTION(")
	}
	for i, p := range fl.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.Name.String())
		sb.WriteString(" AS ")
		sb.WriteString(p.Type.String())
	}
	sb.WriteString(")")
	if !fl.IsSub && len(fl.ReturnTypes) > 0 {
		sb.WriteString(" AS ")
		if len(fl.ReturnTypes) > 1 {
			sb.WriteString("(")
		}
		for i, t := range fl.ReturnTypes {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(t.String())
		}
		if len(fl.ReturnTypes) > 1 {
			sb.WriteString(")")
		}
	}
	sb.WriteString(" ... END")
	return sb.String()
}

// SendStatement represents a SEND ... TO ... statement
type SendStatement struct {
	Token   lexer.Token
	Value   Expression
	Channel Expression
}

func (ss *SendStatement) statementNode()       {}
func (ss *SendStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SendStatement) String() string {
	return "SEND " + ss.Value.String() + " TO " + ss.Channel.String()
}

// ReceiveStatement represents a RECEIVE ... FROM ... statement. The
// optional second target (`RECEIVE v, ok FROM ch`) captures the
// comma-ok flag, which is false once the channel is closed and drained.
type ReceiveStatement struct {
	Token    lexer.Token
	Variable Expression
	OkVar    Expression // optional; the ", ok" comma-ok target
	Channel  Expression
}

func (rs *ReceiveStatement) statementNode()       {}
func (rs *ReceiveStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReceiveStatement) String() string {
	lhs := rs.Variable.String()
	if rs.OkVar != nil {
		lhs += ", " + rs.OkVar.String()
	}
	return "RECEIVE " + lhs + " FROM " + rs.Channel.String()
}

// ExpressionStatement wraps an expression as a statement
type ExpressionStatement struct {
	Token      lexer.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement represents a block of statements
type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var sb strings.Builder
	for _, s := range bs.Statements {
		sb.WriteString("  ")
		sb.WriteString(s.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

// --- Expressions ---

// Identifier represents an identifier
type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral represents an integer literal
type IntegerLiteral struct {
	Token lexer.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral represents a floating-point literal
type FloatLiteral struct {
	Token lexer.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral represents a string literal
type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return "\"" + sl.Value + "\"" }

// ByteStringLiteral represents a byte string literal (B"...")
type ByteStringLiteral struct {
	Token lexer.Token
	Value string
}

func (bs *ByteStringLiteral) expressionNode()      {}
func (bs *ByteStringLiteral) TokenLiteral() string { return bs.Token.Literal }
func (bs *ByteStringLiteral) String() string       { return "B\"" + bs.Value + "\"" }

// BooleanLiteral represents a boolean literal
type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string {
	if bl.Value {
		return "TRUE"
	}
	return "FALSE"
}

// NilLiteral represents NIL
type NilLiteral struct {
	Token lexer.Token
}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NilLiteral) String() string       { return "NIL" }

// JSONLiteral represents a JSON literal
type JSONLiteral struct {
	Token lexer.Token
	Pairs map[string]Expression
	// Keys in the order they were written. Ranging over a Go map is
	// deliberately randomised, so emitting from Pairs alone would produce
	// different output on every run; this keeps the generated code stable
	// and in the order the programmer wrote.
	Order []string
}

func (jl *JSONLiteral) expressionNode()      {}
func (jl *JSONLiteral) TokenLiteral() string { return jl.Token.Literal }
func (jl *JSONLiteral) String() string {
	var pairs []string
	for k, v := range jl.Pairs {
		pairs = append(pairs, "\""+k+"\": "+v.String())
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// ArrayLiteral represents an array literal
type ArrayLiteral struct {
	Token    lexer.Token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var elements []string
	for _, e := range al.Elements {
		elements = append(elements, e.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

// StructLiteral represents a struct literal (TypeName{field: value, ...})
type StructLiteral struct {
	Token    lexer.Token
	TypeName string                // The struct type name
	Fields   map[string]Expression // field: value pairs
	// Field names in the order they were written; see JSONLiteral.Order.
	Order []string
}

func (sl *StructLiteral) expressionNode()      {}
func (sl *StructLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StructLiteral) String() string {
	var pairs []string
	for k, v := range sl.Fields {
		pairs = append(pairs, k+": "+v.String())
	}
	return sl.TypeName + "{" + strings.Join(pairs, ", ") + "}"
}

// SliceLiteral represents a slice literal like []Type{elem1, elem2}
type SliceLiteral struct {
	Token       lexer.Token
	ElementType string       // The element type name
	Elements    []Expression // The elements
}

func (sl *SliceLiteral) expressionNode()      {}
func (sl *SliceLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *SliceLiteral) String() string {
	var elems []string
	for _, e := range sl.Elements {
		elems = append(elems, e.String())
	}
	return "[]" + sl.ElementType + "{" + strings.Join(elems, ", ") + "}"
}

// PrefixExpression represents a prefix expression (NOT, -, @, ^)
type PrefixExpression struct {
	Token    lexer.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	return "(" + pe.Operator + pe.Right.String() + ")"
}

// InfixExpression represents an infix expression
type InfixExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	return "(" + ie.Left.String() + " " + ie.Operator + " " + ie.Right.String() + ")"
}

// CallExpression represents a function/sub call
type CallExpression struct {
	Token     lexer.Token
	Function  Expression // Identifier or MemberExpression
	Arguments []Expression
	TypeArgs  []*TypeSpec // Explicit generic instantiation: pkg.Func(OF T1, T2)(args)
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var args []string
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	return ce.Function.String() + "(" + strings.Join(args, ", ") + ")"
}

// IndexExpression represents array/slice indexing or slicing
type IndexExpression struct {
	Token   lexer.Token
	Left    Expression
	Index   Expression // For simple indexing or slice start
	End     Expression // For slice end (nil if simple indexing)
	IsSlice bool       // True if this is a slice operation [start:end]
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	if ie.IsSlice {
		start := ""
		end := ""
		if ie.Index != nil {
			start = ie.Index.String()
		}
		if ie.End != nil {
			end = ie.End.String()
		}
		return "(" + ie.Left.String() + "[" + start + ":" + end + "])"
	}
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}

// MemberExpression represents member access (dot notation)
type MemberExpression struct {
	Token  lexer.Token
	Object Expression
	Member *Identifier
}

func (me *MemberExpression) expressionNode()      {}
func (me *MemberExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MemberExpression) String() string {
	return me.Object.String() + "." + me.Member.String()
}

// TypeAssertionExpression represents a type assertion: value.(Type)
type TypeAssertionExpression struct {
	Token      lexer.Token // The '.' token
	Value      Expression  // The expression being asserted
	TargetType *TypeSpec   // The target type
}

func (ta *TypeAssertionExpression) expressionNode()      {}
func (ta *TypeAssertionExpression) TokenLiteral() string { return ta.Token.Literal }
func (ta *TypeAssertionExpression) String() string {
	return ta.Value.String() + ".(" + ta.TargetType.String() + ")"
}

// MakeChanExpression represents MAKE_CHAN(TYPE, size)
type MakeChanExpression struct {
	Token       lexer.Token
	ChannelType *TypeSpec
	Size        Expression // Buffer size (optional)
}

func (mc *MakeChanExpression) expressionNode()      {}
func (mc *MakeChanExpression) TokenLiteral() string { return mc.Token.Literal }
func (mc *MakeChanExpression) String() string {
	if mc.Size != nil {
		return "MAKE_CHAN(" + mc.ChannelType.String() + ", " + mc.Size.String() + ")"
	}
	return "MAKE_CHAN(" + mc.ChannelType.String() + ")"
}

// SpreadExpression represents a variadic spread argument in a call:
// `f(a, xs...)`. It wraps the slice expression being spread into the
// callee's variadic parameter (Go `f(a, xs...)`).
type SpreadExpression struct {
	Token lexer.Token // the '...' token
	Value Expression  // the slice being spread
}

func (se *SpreadExpression) expressionNode()      {}
func (se *SpreadExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SpreadExpression) String() string       { return se.Value.String() + "..." }

// ReceiveExpression represents receiving from a channel as an expression
type ReceiveExpression struct {
	Token   lexer.Token
	Channel Expression
}

func (re *ReceiveExpression) expressionNode()      {}
func (re *ReceiveExpression) TokenLiteral() string { return re.Token.Literal }
func (re *ReceiveExpression) String() string {
	return "<-" + re.Channel.String()
}

// AddressOfExpression represents @variable
type AddressOfExpression struct {
	Token lexer.Token
	Value Expression
}

func (ae *AddressOfExpression) expressionNode()      {}
func (ae *AddressOfExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AddressOfExpression) String() string {
	return "@" + ae.Value.String()
}

// DereferenceExpression represents ^pointer
type DereferenceExpression struct {
	Token lexer.Token
	Value Expression
}

func (de *DereferenceExpression) expressionNode()      {}
func (de *DereferenceExpression) TokenLiteral() string { return de.Token.Literal }
func (de *DereferenceExpression) String() string {
	return "^" + de.Value.String()
}
