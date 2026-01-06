package parser

import (
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

// TypeSpec represents a type specification
type TypeSpec struct {
	Token       lexer.Token // The type token
	Name        string      // Base type name (INTEGER, STRING, etc.)
	IsPointer   bool        // POINTER TO X
	IsChannel   bool        // CHAN OF X
	ElementType *TypeSpec   // For POINTER TO and CHAN OF
	IsArray     bool        // Array type
	ArraySize   Expression  // Array size expression (can be nil for dynamic)
}

func (t *TypeSpec) TokenLiteral() string { return t.Token.Literal }
func (t *TypeSpec) String() string {
	if t.IsPointer {
		return "POINTER TO " + t.ElementType.String()
	}
	if t.IsChannel {
		return "CHAN OF " + t.ElementType.String()
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

// SpawnStatement represents a SPAWN statement (goroutine)
type SpawnStatement struct {
	Token lexer.Token
	Call  *CallExpression
}

func (ss *SpawnStatement) statementNode()       {}
func (ss *SpawnStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SpawnStatement) String() string       { return "SPAWN " + ss.Call.String() }

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

// ReceiveStatement represents a RECEIVE ... FROM ... statement
type ReceiveStatement struct {
	Token    lexer.Token
	Variable Expression
	Channel  Expression
}

func (rs *ReceiveStatement) statementNode()       {}
func (rs *ReceiveStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReceiveStatement) String() string {
	return "RECEIVE " + rs.Variable.String() + " FROM " + rs.Channel.String()
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

// IndexExpression represents array/slice indexing
type IndexExpression struct {
	Token lexer.Token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
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
