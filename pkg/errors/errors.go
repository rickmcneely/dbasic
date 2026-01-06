package errors

import (
	"fmt"
	"strings"
)

// CompileError represents a compilation error with source context
type CompileError struct {
	Phase    string // "lexer", "parser", "analyzer"
	Line     int
	Column   int
	Message  string
	Source   string // The source line where the error occurred
	Hint     string // Optional hint for fixing the error
}

func (e *CompileError) Error() string {
	var sb strings.Builder

	// Main error message with location
	if e.Line > 0 {
		sb.WriteString(fmt.Sprintf("%s error at line %d", e.Phase, e.Line))
		if e.Column > 0 {
			sb.WriteString(fmt.Sprintf(", column %d", e.Column))
		}
		sb.WriteString(": ")
	} else {
		sb.WriteString(fmt.Sprintf("%s error: ", e.Phase))
	}
	sb.WriteString(e.Message)
	sb.WriteString("\n")

	// Show source line with caret pointing to error location
	if e.Source != "" && e.Line > 0 {
		sb.WriteString(fmt.Sprintf("  %d | %s\n", e.Line, e.Source))
		if e.Column > 0 {
			// Calculate padding for the caret
			lineNumWidth := len(fmt.Sprintf("%d", e.Line))
			padding := lineNumWidth + 3 + e.Column - 1 // " | " = 3 chars
			sb.WriteString(strings.Repeat(" ", padding))
			sb.WriteString("^\n")
		}
	}

	// Optional hint
	if e.Hint != "" {
		sb.WriteString(fmt.Sprintf("  hint: %s\n", e.Hint))
	}

	return sb.String()
}

// SourceContext holds the source code for error reporting
type SourceContext struct {
	lines []string
}

// NewSourceContext creates a new SourceContext from source code
func NewSourceContext(source string) *SourceContext {
	return &SourceContext{
		lines: strings.Split(source, "\n"),
	}
}

// GetLine returns the source line at the given line number (1-indexed)
func (sc *SourceContext) GetLine(lineNum int) string {
	if lineNum < 1 || lineNum > len(sc.lines) {
		return ""
	}
	return sc.lines[lineNum-1]
}

// GetLines returns the total number of lines
func (sc *SourceContext) GetLines() int {
	return len(sc.lines)
}

// FormatError creates a formatted CompileError with source context
func (sc *SourceContext) FormatError(phase string, line, column int, message string, hint string) *CompileError {
	return &CompileError{
		Phase:   phase,
		Line:    line,
		Column:  column,
		Message: message,
		Source:  sc.GetLine(line),
		Hint:    hint,
	}
}

// ErrorList is a collection of compile errors
type ErrorList struct {
	Errors  []*CompileError
	context *SourceContext
	phase   string
}

// NewErrorList creates a new ErrorList
func NewErrorList(source string, phase string) *ErrorList {
	return &ErrorList{
		Errors:  []*CompileError{},
		context: NewSourceContext(source),
		phase:   phase,
	}
}

// Add adds an error to the list
func (el *ErrorList) Add(line, column int, message string) {
	el.AddWithHint(line, column, message, "")
}

// AddWithHint adds an error with a hint to the list
func (el *ErrorList) AddWithHint(line, column int, message, hint string) {
	err := el.context.FormatError(el.phase, line, column, message, hint)
	el.Errors = append(el.Errors, err)
}

// HasErrors returns true if there are any errors
func (el *ErrorList) HasErrors() bool {
	return len(el.Errors) > 0
}

// Messages returns all error messages as strings
func (el *ErrorList) Messages() []string {
	msgs := make([]string, len(el.Errors))
	for i, err := range el.Errors {
		msgs[i] = err.Error()
	}
	return msgs
}

// String returns all errors as a single string
func (el *ErrorList) String() string {
	var sb strings.Builder
	for _, err := range el.Errors {
		sb.WriteString(err.Error())
	}
	return sb.String()
}
