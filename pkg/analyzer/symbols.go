package analyzer

import (
	"strings"

	"github.com/zditech/dbasic/pkg/parser"
)

// SymbolKind represents the kind of symbol
type SymbolKind int

const (
	SymVariable SymbolKind = iota
	SymConstant
	SymFunction
	SymSub
	SymParameter
	SymLabel
	SymImport
)

// Symbol represents a symbol in the symbol table
type Symbol struct {
	Name       string
	Kind       SymbolKind
	Type       *Type
	Node       parser.Node   // The AST node that defined this symbol
	Scope      *Scope
	IsByRef    bool          // For parameters passed by reference
	IsExported bool          // For Go package interop
	GoName     string        // The Go identifier name (for imports)
}

// Scope represents a scope in the symbol table
type Scope struct {
	Name    string
	Parent  *Scope
	symbols map[string]*Symbol
	labels  map[string]*Symbol
}

// NewScope creates a new scope
func NewScope(name string, parent *Scope) *Scope {
	return &Scope{
		Name:    name,
		Parent:  parent,
		symbols: make(map[string]*Symbol),
		labels:  make(map[string]*Symbol),
	}
}

// Define adds a symbol to the scope
func (s *Scope) Define(sym *Symbol) error {
	name := strings.ToUpper(sym.Name)
	if _, exists := s.symbols[name]; exists {
		return &SymbolError{
			Message: "symbol already defined: " + sym.Name,
		}
	}
	sym.Scope = s
	s.symbols[name] = sym
	return nil
}

// DefineLabel adds a label to the scope
func (s *Scope) DefineLabel(name string, sym *Symbol) error {
	upperName := strings.ToUpper(name)
	if _, exists := s.labels[upperName]; exists {
		return &SymbolError{
			Message: "label already defined: " + name,
		}
	}
	s.labels[upperName] = sym
	return nil
}

// Resolve looks up a symbol in the current scope and parent scopes
func (s *Scope) Resolve(name string) *Symbol {
	upperName := strings.ToUpper(name)
	if sym, ok := s.symbols[upperName]; ok {
		return sym
	}
	if s.Parent != nil {
		return s.Parent.Resolve(name)
	}
	return nil
}

// ResolveLocal looks up a symbol only in the current scope
func (s *Scope) ResolveLocal(name string) *Symbol {
	return s.symbols[strings.ToUpper(name)]
}

// ResolveLabel looks up a label in the current scope
func (s *Scope) ResolveLabel(name string) *Symbol {
	upperName := strings.ToUpper(name)
	if sym, ok := s.labels[upperName]; ok {
		return sym
	}
	// Labels are only valid in their defining scope
	return nil
}

// AllSymbols returns all symbols in this scope
func (s *Scope) AllSymbols() []*Symbol {
	result := make([]*Symbol, 0, len(s.symbols))
	for _, sym := range s.symbols {
		result = append(result, sym)
	}
	return result
}

// SymbolTable manages all scopes and symbols
type SymbolTable struct {
	GlobalScope  *Scope
	CurrentScope *Scope
	imports      map[string]*ImportInfo
}

// ImportInfo stores information about an imported package
type ImportInfo struct {
	Path  string
	Alias string
}

// NewSymbolTable creates a new symbol table
func NewSymbolTable() *SymbolTable {
	global := NewScope("global", nil)
	return &SymbolTable{
		GlobalScope:  global,
		CurrentScope: global,
		imports:      make(map[string]*ImportInfo),
	}
}

// EnterScope creates and enters a new scope
func (st *SymbolTable) EnterScope(name string) *Scope {
	newScope := NewScope(name, st.CurrentScope)
	st.CurrentScope = newScope
	return newScope
}

// ExitScope returns to the parent scope
func (st *SymbolTable) ExitScope() {
	if st.CurrentScope.Parent != nil {
		st.CurrentScope = st.CurrentScope.Parent
	}
}

// Define adds a symbol to the current scope
func (st *SymbolTable) Define(sym *Symbol) error {
	return st.CurrentScope.Define(sym)
}

// DefineGlobal adds a symbol to the global scope
func (st *SymbolTable) DefineGlobal(sym *Symbol) error {
	return st.GlobalScope.Define(sym)
}

// Resolve looks up a symbol
func (st *SymbolTable) Resolve(name string) *Symbol {
	return st.CurrentScope.Resolve(name)
}

// AddImport adds an import to the symbol table
func (st *SymbolTable) AddImport(path, alias string) {
	key := path
	if alias != "" {
		key = alias
	} else {
		// Use the last part of the path as the default alias
		parts := strings.Split(path, "/")
		key = parts[len(parts)-1]
	}
	st.imports[key] = &ImportInfo{Path: path, Alias: alias}
}

// GetImport returns import info for the given alias/name
func (st *SymbolTable) GetImport(name string) *ImportInfo {
	return st.imports[name]
}

// AllImports returns all imports
func (st *SymbolTable) AllImports() map[string]*ImportInfo {
	return st.imports
}

// IsGlobalScope returns true if currently in global scope
func (st *SymbolTable) IsGlobalScope() bool {
	return st.CurrentScope == st.GlobalScope
}

// SymbolError represents a symbol-related error
type SymbolError struct {
	Message string
	Line    int
	Column  int
}

func (e *SymbolError) Error() string {
	if e.Line > 0 {
		return e.Message
	}
	return e.Message
}
