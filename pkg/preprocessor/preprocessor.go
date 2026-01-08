// Package preprocessor handles file inclusion and other preprocessing directives.
package preprocessor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SourceMapping tracks the original source file and line for preprocessed code.
type SourceMapping struct {
	File string
	Line int
}

// Result contains the preprocessed source and metadata.
type Result struct {
	Source      string
	LineMap     []SourceMapping // Maps output line number to original file:line
	MainFile    string
	IncludedFiles []string
}

// Preprocessor handles INCLUDE directives and other preprocessing.
type Preprocessor struct {
	baseDir      string
	includedSet  map[string]bool // Track included files to prevent circular includes
	includedList []string        // Ordered list of included files
	lineMap      []SourceMapping
	errors       []string
}

// New creates a new preprocessor with the given base directory.
func New(baseDir string) *Preprocessor {
	return &Preprocessor{
		baseDir:     baseDir,
		includedSet: make(map[string]bool),
	}
}

// Process preprocesses the source file, expanding INCLUDE directives.
func (p *Preprocessor) Process(filename string) (*Result, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve path '%s': %v", filename, err)
	}

	// Set base directory from the main file's location
	p.baseDir = filepath.Dir(absPath)

	source, err := p.processFile(absPath, 0)
	if err != nil {
		return nil, err
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("preprocessing errors:\n%s", strings.Join(p.errors, "\n"))
	}

	return &Result{
		Source:        source,
		LineMap:       p.lineMap,
		MainFile:      absPath,
		IncludedFiles: p.includedList,
	}, nil
}

// processFile reads and processes a single file, recursively handling includes.
func (p *Preprocessor) processFile(filename string, depth int) (string, error) {
	const maxDepth = 100 // Prevent infinite recursion

	if depth > maxDepth {
		return "", fmt.Errorf("include depth exceeded %d (possible circular include)", maxDepth)
	}

	// Normalize the path
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path '%s': %v", filename, err)
	}

	// Check for circular includes
	if p.includedSet[absPath] {
		return "", fmt.Errorf("circular include detected: %s", absPath)
	}
	p.includedSet[absPath] = true
	p.includedList = append(p.includedList, absPath)

	// Read the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("cannot read file '%s': %v", filename, err)
	}

	// Get the directory of this file for resolving relative includes
	fileDir := filepath.Dir(absPath)
	baseName := filepath.Base(absPath)

	// Process line by line
	var output strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNum := 0

	// Regex to match INCLUDE "filename" (case insensitive)
	includeRe := regexp.MustCompile(`(?i)^\s*INCLUDE\s+"([^"]+)"\s*(?:'.*)?$`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check for INCLUDE directive
		if matches := includeRe.FindStringSubmatch(line); matches != nil {
			includePath := matches[1]

			// Resolve the include path relative to the current file
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(fileDir, includePath)
			}

			// Add a comment showing where the include came from (useful for debugging)
			p.lineMap = append(p.lineMap, SourceMapping{File: baseName, Line: lineNum})
			output.WriteString(fmt.Sprintf("' >>> BEGIN INCLUDE: %s (from %s:%d)\n",
				filepath.Base(includePath), baseName, lineNum))

			// Process the included file
			includedSource, err := p.processFile(includePath, depth+1)
			if err != nil {
				p.errors = append(p.errors, fmt.Sprintf("%s:%d: %v", baseName, lineNum, err))
				// Continue processing to find all errors
				continue
			}

			output.WriteString(includedSource)

			// Add end marker
			p.lineMap = append(p.lineMap, SourceMapping{File: baseName, Line: lineNum})
			output.WriteString(fmt.Sprintf("' <<< END INCLUDE: %s\n", filepath.Base(includePath)))
		} else {
			// Regular line - add to output and track source mapping
			p.lineMap = append(p.lineMap, SourceMapping{File: baseName, Line: lineNum})
			output.WriteString(line)
			output.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading '%s': %v", filename, err)
	}

	return output.String(), nil
}

// Errors returns any errors encountered during preprocessing.
func (p *Preprocessor) Errors() []string {
	return p.errors
}

// GetOriginalLocation returns the original file and line for a preprocessed line number.
// Line numbers are 1-based.
func (r *Result) GetOriginalLocation(line int) (string, int) {
	if line < 1 || line > len(r.LineMap) {
		return "", 0
	}
	mapping := r.LineMap[line-1]
	return mapping.File, mapping.Line
}
