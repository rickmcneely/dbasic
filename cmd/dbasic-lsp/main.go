// dbasic-lsp is a Language Server Protocol implementation for DBasic.
// It listens on stdin/stdout for JSON-RPC messages and provides
// real-time compile diagnostics, hover, document symbols, and
// go-to-definition for any editor that speaks LSP — VS Code, Neovim,
// Helix, Emacs, JetBrains.
//
// Currently supported:
//   - initialize / initialized / shutdown / exit handshake
//   - textDocument/didOpen / didChange / didSave / didClose
//   - textDocument/publishDiagnostics on parse and analyze errors
//   - textDocument/documentSymbol (file outline)
//   - textDocument/hover (signature info on identifiers)
//   - textDocument/definition (jump to top-level def)
//
// To wire it up to an editor, point the editor at the dbasic-lsp
// binary as an LSP server for files matching `*.dbas`. Example for
// Neovim with nvim-lspconfig:
//
//	require'lspconfig.configs'.dbasic = {
//	  default_config = {
//	    cmd = { 'dbasic-lsp' },
//	    filetypes = { 'dbasic' },
//	    root_dir = function(fname) return vim.fn.getcwd() end,
//	  },
//	}
//	require'lspconfig'.dbasic.setup{}
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/zditech/dbasic/pkg/analyzer"
	"github.com/zditech/dbasic/pkg/lexer"
	"github.com/zditech/dbasic/pkg/parser"
)

// --- LSP type definitions (just what we need) -------------------------------

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type rangePos struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type diagnostic struct {
	Range    rangePos `json:"range"`
	Severity int      `json:"severity"` // 1=Error, 2=Warning
	Source   string   `json:"source"`
	Message  string   `json:"message"`
}

type publishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     int          `json:"version,omitempty"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// --- server state -----------------------------------------------------------

type server struct {
	mu      sync.Mutex
	docs    map[string]string // uri -> content
	out     io.Writer
	outLock sync.Mutex
}

func main() {
	s := &server{
		docs: map[string]string{},
		out:  os.Stdout,
	}
	s.run(bufio.NewReader(os.Stdin))
}

func (s *server) run(r *bufio.Reader) {
	for {
		raw, err := readMessage(r)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "dbasic-lsp: read error: %v\n", err)
			return
		}
		s.handle(raw)
	}
}

// readMessage reads one LSP-framed JSON-RPC payload:
//
//	Content-Length: N\r\n
//	\r\n
//	<N bytes of JSON>
func readMessage(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if cl, ok := strings.CutPrefix(line, "Content-Length: "); ok {
			n, err := strconv.Atoi(cl)
			if err != nil {
				return nil, fmt.Errorf("bad Content-Length: %q", cl)
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *server) handle(raw []byte) {
	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}

	switch req.Method {
	case "initialize":
		s.respond(req.ID, map[string]any{
			"capabilities": map[string]any{
				// 1 = full document sync (we re-receive the entire file on each change)
				"textDocumentSync":       1,
				"documentSymbolProvider": true,
				"hoverProvider":          true,
				"definitionProvider":     true,
				"referencesProvider":     true,
				"renameProvider":         true,
				"completionProvider": map[string]any{
					"triggerCharacters": []string{"."},
				},
				"signatureHelpProvider": map[string]any{
					"triggerCharacters": []string{"(", ","},
				},
			},
			"serverInfo": map[string]string{
				"name":    "dbasic-lsp",
				"version": "0.4",
			},
		})

	case "initialized":
		// no-op

	case "shutdown":
		s.respond(req.ID, nil)

	case "exit":
		os.Exit(0)

	case "textDocument/didOpen":
		var p struct {
			TextDocument struct {
				URI     string `json:"uri"`
				Version int    `json:"version"`
				Text    string `json:"text"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return
		}
		s.setDoc(p.TextDocument.URI, p.TextDocument.Text)
		s.publishDiagnostics(p.TextDocument.URI, p.TextDocument.Version)

	case "textDocument/didChange":
		var p struct {
			TextDocument struct {
				URI     string `json:"uri"`
				Version int    `json:"version"`
			} `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return
		}
		// With textDocumentSync=1 (full), each change has the whole file as Text.
		if len(p.ContentChanges) > 0 {
			s.setDoc(p.TextDocument.URI, p.ContentChanges[len(p.ContentChanges)-1].Text)
			s.publishDiagnostics(p.TextDocument.URI, p.TextDocument.Version)
		}

	case "textDocument/didSave":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return
		}
		if p.Text != "" {
			s.setDoc(p.TextDocument.URI, p.Text)
		}
		s.publishDiagnostics(p.TextDocument.URI, 0)

	case "textDocument/documentSymbol":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			s.respond(req.ID, []docSymbol{})
			return
		}
		s.respond(req.ID, s.documentSymbols(p.TextDocument.URI))

	case "textDocument/hover":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Position position `json:"position"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			s.respond(req.ID, nil)
			return
		}
		s.respond(req.ID, s.hover(p.TextDocument.URI, p.Position))

	case "textDocument/definition":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Position position `json:"position"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			s.respond(req.ID, nil)
			return
		}
		s.respond(req.ID, s.definition(p.TextDocument.URI, p.Position))

	case "textDocument/completion":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Position position `json:"position"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			s.respond(req.ID, []completionItem{})
			return
		}
		s.respond(req.ID, s.completion(p.TextDocument.URI, p.Position))

	case "textDocument/references":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Position position `json:"position"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			s.respond(req.ID, []location{})
			return
		}
		s.respond(req.ID, s.references(p.TextDocument.URI, p.Position))

	case "textDocument/signatureHelp":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Position position `json:"position"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			s.respond(req.ID, nil)
			return
		}
		s.respond(req.ID, s.signatureHelp(p.TextDocument.URI, p.Position))

	case "textDocument/rename":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Position position `json:"position"`
			NewName  string   `json:"newName"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			s.respond(req.ID, nil)
			return
		}
		s.respond(req.ID, s.rename(p.TextDocument.URI, p.Position, p.NewName))

	case "textDocument/didClose":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return
		}
		s.deleteDoc(p.TextDocument.URI)
		// Clear stale diagnostics on close.
		s.send(notification{
			JSONRPC: "2.0",
			Method:  "textDocument/publishDiagnostics",
			Params: publishDiagnosticsParams{
				URI:         p.TextDocument.URI,
				Diagnostics: []diagnostic{},
			},
		})

	default:
		// Unknown methods: return method-not-found for requests, ignore notifications.
		if len(req.ID) > 0 {
			s.respondError(req.ID, -32601, "method not found: "+req.Method)
		}
	}
}

// --- document tracking ------------------------------------------------------

func (s *server) setDoc(uri, text string) {
	s.mu.Lock()
	s.docs[uri] = text
	s.mu.Unlock()
}

func (s *server) deleteDoc(uri string) {
	s.mu.Lock()
	delete(s.docs, uri)
	s.mu.Unlock()
}

func (s *server) getDoc(uri string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.docs[uri]
	return t, ok
}

// --- diagnostics ------------------------------------------------------------

func (s *server) publishDiagnostics(uri string, version int) {
	text, ok := s.getDoc(uri)
	if !ok {
		return
	}
	diags := analyze(text)
	s.send(notification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: publishDiagnosticsParams{
			URI:         uri,
			Version:     version,
			Diagnostics: diags,
		},
	})
}

// analyze runs the lexer/parser/analyzer pipeline on text and returns
// LSP diagnostics. Errors come back as pre-formatted multi-line strings
// like "parse error at line 5, column 12: ...\n  5 | ...\n   ^\n hint: ...".
// We split on those, regex-extract the line/column, and use the first
// line of each error as the diagnostic message.
func analyze(text string) []diagnostic {
	var diags []diagnostic

	l := lexer.New(text)
	p := parser.New(l)
	prog := p.ParseProgram()

	for _, errMsg := range p.Errors() {
		if d, ok := parseErrorMessage(errMsg, "parser"); ok {
			diags = append(diags, d)
		}
	}

	// Only run analyzer if parsing succeeded enough to produce a program.
	if prog != nil {
		a := analyzer.New()
		_, errs := a.Analyze(prog)
		for _, errMsg := range errs {
			if d, ok := parseErrorMessage(errMsg, "analyzer"); ok {
				diags = append(diags, d)
			}
		}
	}
	return diags
}

// errLineRE pulls "line N" and optional ", column M" from the first
// line of a parser/analyzer error string.
var errLineRE = regexp.MustCompile(`(?m)^(?:parse|semantic) error at line (\d+)(?:, column (\d+))?:\s*(.*?)$`)

func parseErrorMessage(raw, source string) (diagnostic, bool) {
	m := errLineRE.FindStringSubmatch(raw)
	if m == nil {
		return diagnostic{
			Range:    rangePos{},
			Severity: 1,
			Source:   "dbasic-" + source,
			Message:  strings.TrimSpace(strings.SplitN(raw, "\n", 2)[0]),
		}, true
	}
	line, _ := strconv.Atoi(m[1])
	col := 0
	if m[2] != "" {
		col, _ = strconv.Atoi(m[2])
	}
	// LSP uses 0-based positions; DBasic errors are 1-based.
	startLine := line - 1
	startChar := 0
	endChar := 100
	if col > 0 {
		startChar = col - 1
		endChar = startChar + 1
	}
	return diagnostic{
		Range: rangePos{
			Start: position{Line: startLine, Character: startChar},
			End:   position{Line: startLine, Character: endChar},
		},
		Severity: 1,
		Source:   "dbasic-" + source,
		Message:  strings.TrimSpace(m[3]),
	}, true
}

// --- transport -------------------------------------------------------------

func (s *server) send(v any) {
	s.outLock.Lock()
	defer s.outLock.Unlock()
	body, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n", len(body))
	s.out.Write(body)
}

func (s *server) respond(id json.RawMessage, result any) {
	s.send(response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *server) respondError(id json.RawMessage, code int, message string) {
	s.send(response{JSONRPC: "2.0", ID: id, Error: &responseError{Code: code, Message: message}})
}

// uriToPath strips a "file://..." prefix and percent-decodes the path.
// (Reserved for future use — the current diagnostics-only path doesn't
// need to read disk; we work entirely from the in-memory document text.)
func uriToPath(uri string) string {
	if u, err := url.Parse(uri); err == nil && u.Scheme == "file" {
		return u.Path
	}
	return uri
}

// --- LSP capability types ---------------------------------------------------

type docSymbol struct {
	Name           string      `json:"name"`
	Detail         string      `json:"detail,omitempty"`
	Kind           int         `json:"kind"`
	Range          rangePos    `json:"range"`
	SelectionRange rangePos    `json:"selectionRange"`
	Children       []docSymbol `json:"children,omitempty"`
}

type hoverResult struct {
	Contents markupContent `json:"contents"`
	Range    *rangePos     `json:"range,omitempty"`
}

type markupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type location struct {
	URI   string   `json:"uri"`
	Range rangePos `json:"range"`
}

type textEdit struct {
	Range   rangePos `json:"range"`
	NewText string   `json:"newText"`
}

type workspaceEdit struct {
	Changes map[string][]textEdit `json:"changes"`
}

type signatureHelpResult struct {
	Signatures      []signatureInformation `json:"signatures"`
	ActiveSignature int                    `json:"activeSignature"`
	ActiveParameter int                    `json:"activeParameter"`
}

type signatureInformation struct {
	Label         string                 `json:"label"`
	Documentation string                 `json:"documentation,omitempty"`
	Parameters    []parameterInformation `json:"parameters"`
}

type parameterInformation struct {
	Label string `json:"label"`
}

type completionItem struct {
	Label         string `json:"label"`
	Kind          int    `json:"kind"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	InsertText    string `json:"insertText,omitempty"`
}

// LSP CompletionItemKind subset.
const (
	ckText     int = 1
	ckMethod   int = 2
	ckFunction int = 3
	ckField    int = 5
	ckVariable int = 6
	ckClass    int = 7
	ckProperty int = 10
	ckKeyword  int = 14
	ckConstant int = 21
	ckStruct   int = 22
)

// LSP SymbolKind values we emit (subset).
const (
	skVariable int = 13
	skConstant int = 14
	skFunction int = 12
	skMethod   int = 6
	skStruct   int = 23
	skField    int = 8
)

// defInfo records a top-level definition's location and presentable signature.
type defInfo struct {
	name    string
	kind    int
	sig     string
	line    int // 1-based
	col     int // 1-based start of name
	nameLen int
	endLine int // 1-based end (last line of the definition); falls back to line
	endCol  int
	fields  []defInfo
}

// programIndex bundles top-level defs in two forms: hierarchical (for
// documentSymbol) and a flat case-insensitive lookup (for hover/definition).
type programIndex struct {
	tree []defInfo
	flat map[string]defInfo
}

func (s *server) parseDoc(uri string) (*parser.Program, string, bool) {
	text, ok := s.getDoc(uri)
	if !ok {
		return nil, "", false
	}
	l := lexer.New(text)
	p := parser.New(l)
	prog := p.ParseProgram()
	return prog, text, true
}

func indexProgram(prog *parser.Program) programIndex {
	idx := programIndex{flat: map[string]defInfo{}}
	if prog == nil {
		return idx
	}
	for _, st := range prog.Statements {
		switch n := st.(type) {
		case *parser.SubStatement:
			d := defInfo{
				name: n.Name.Value, kind: skFunction,
				sig:  "SUB " + n.Name.Value + paramSig(n.Params),
				line: n.Name.Token.Line, col: n.Name.Token.Column,
				nameLen: len(n.Name.Value),
			}
			idx.tree = append(idx.tree, d)
			idx.flat[strings.ToUpper(d.name)] = d
		case *parser.FunctionStatement:
			d := defInfo{
				name: n.Name.Value, kind: skFunction,
				sig:  "FUNCTION " + n.Name.Value + paramSig(n.Params) + retSig(n.ReturnTypes),
				line: n.Name.Token.Line, col: n.Name.Token.Column,
				nameLen: len(n.Name.Value),
			}
			idx.tree = append(idx.tree, d)
			idx.flat[strings.ToUpper(d.name)] = d
		case *parser.MethodStatement:
			recv := typeStr(n.ReceiverType)
			d := defInfo{
				name: n.Name.Value, kind: skMethod,
				sig:  fmt.Sprintf("FUNCTION (%s AS %s) %s%s%s", n.ReceiverName.Value, recv, n.Name.Value, paramSig(n.Params), retSig(n.ReturnTypes)),
				line: n.Name.Token.Line, col: n.Name.Token.Column,
				nameLen: len(n.Name.Value),
			}
			idx.tree = append(idx.tree, d)
			idx.flat[strings.ToUpper(d.name)] = d
		case *parser.TypeStatement:
			td := defInfo{
				name: n.Name.Value, kind: skStruct,
				sig:  "TYPE " + n.Name.Value,
				line: n.Name.Token.Line, col: n.Name.Token.Column,
				nameLen: len(n.Name.Value),
			}
			for _, f := range n.Fields {
				fd := defInfo{
					name: f.Name.Value, kind: skField,
					sig:  "DIM " + f.Name.Value + " AS " + typeStr(f.Type),
					line: f.Name.Token.Line, col: f.Name.Token.Column,
					nameLen: len(f.Name.Value),
				}
				td.fields = append(td.fields, fd)
				if _, exists := idx.flat[strings.ToUpper(fd.name)]; !exists {
					idx.flat[strings.ToUpper(fd.name)] = fd
				}
			}
			idx.tree = append(idx.tree, td)
			idx.flat[strings.ToUpper(td.name)] = td
		case *parser.ConstStatement:
			d := defInfo{
				name: n.Name.Value, kind: skConstant,
				sig:  "CONST " + n.Name.Value + " AS " + typeStr(n.Type),
				line: n.Name.Token.Line, col: n.Name.Token.Column,
				nameLen: len(n.Name.Value),
			}
			idx.tree = append(idx.tree, d)
			idx.flat[strings.ToUpper(d.name)] = d
		case *parser.DimStatement:
			d := defInfo{
				name: n.Name.Value, kind: skVariable,
				sig:  "DIM " + n.Name.Value + " AS " + typeStr(n.Type),
				line: n.Name.Token.Line, col: n.Name.Token.Column,
				nameLen: len(n.Name.Value),
			}
			idx.tree = append(idx.tree, d)
			idx.flat[strings.ToUpper(d.name)] = d
		}
	}
	return idx
}

func (s *server) documentSymbols(uri string) []docSymbol {
	prog, _, ok := s.parseDoc(uri)
	if !ok {
		return []docSymbol{}
	}
	idx := indexProgram(prog)
	out := make([]docSymbol, 0, len(idx.tree))
	for _, d := range idx.tree {
		out = append(out, defToDocSymbol(d))
	}
	return out
}

func defToDocSymbol(d defInfo) docSymbol {
	r := identRange(d)
	sym := docSymbol{
		Name:           d.name,
		Detail:         d.sig,
		Kind:           d.kind,
		Range:          r,
		SelectionRange: r,
	}
	for _, f := range d.fields {
		sym.Children = append(sym.Children, defToDocSymbol(f))
	}
	return sym
}

func identRange(d defInfo) rangePos {
	startLine := d.line - 1
	// The lexer reports an identifier's Column as the position *after* the
	// last char (it gets reassigned post-readIdentifier). Walk back by the
	// name length to get the true start column.
	startCh := d.col - 1 - d.nameLen
	if startCh < 0 {
		startCh = 0
	}
	return rangePos{
		Start: position{Line: startLine, Character: startCh},
		End:   position{Line: startLine, Character: startCh + d.nameLen},
	}
}

func (s *server) hover(uri string, pos position) any {
	prog, text, ok := s.parseDoc(uri)
	if !ok {
		return nil
	}
	word, found := wordAt(text, pos)
	if !found {
		return nil
	}
	idx := indexProgram(prog)
	d, ok := idx.flat[strings.ToUpper(word)]
	if !ok {
		return nil
	}
	return hoverResult{
		Contents: markupContent{
			Kind:  "markdown",
			Value: "```dbasic\n" + d.sig + "\n```",
		},
	}
}

// signatureHelp returns parameter info for the function call surrounding
// the cursor. We walk backwards through the current line, tracking paren
// depth, until we find the unbalanced `(` that opens the enclosing call.
// The identifier just before that `(` names the function; commas at depth
// zero between the `(` and the cursor determine the active parameter.
//
// Multi-line calls are not handled — we look at the current line only.
func (s *server) signatureHelp(uri string, pos position) any {
	prog, text, ok := s.parseDoc(uri)
	if !ok {
		return nil
	}
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return nil
	}
	line := lines[pos.Line]
	if pos.Character > len(line) {
		pos.Character = len(line)
	}
	// Strip strings/comments first so parens inside them don't fool us.
	stripped := stripCommentsAndStrings(line)
	// Walk back from the cursor, balancing parens.
	depth := 0
	commas := 0
	openParen := -1
	for i := pos.Character - 1; i >= 0; i-- {
		c := stripped[i]
		switch c {
		case ')', ']':
			depth++
		case '(':
			if depth == 0 {
				openParen = i
				break
			}
			depth--
		case '[':
			if depth == 0 {
				return nil
			}
			depth--
		case ',':
			if depth == 0 {
				commas++
			}
		}
		if openParen >= 0 {
			break
		}
	}
	if openParen < 0 {
		return nil
	}
	// Identifier just before openParen.
	end := openParen
	start := end
	for start > 0 {
		c := line[start-1]
		isPart := isIdentByte(c) || c == '.'
		if !isPart {
			break
		}
		start--
	}
	if start >= end {
		return nil
	}
	callName := line[start:end]
	// Resolve to a known def. For dotted names like `pkg.Func` we look up
	// just the last segment, since member info from external packages is
	// out of scope.
	if dot := strings.LastIndex(callName, "."); dot >= 0 {
		callName = callName[dot+1:]
	}
	idx := indexProgram(prog)
	d, ok := idx.flat[strings.ToUpper(callName)]
	if !ok {
		return nil
	}
	// Build SignatureInformation by parsing the recorded sig string.
	sigInfo := buildSignatureInfo(d.sig)
	active := commas
	if active >= len(sigInfo.Parameters) && len(sigInfo.Parameters) > 0 {
		active = len(sigInfo.Parameters) - 1
	}
	return signatureHelpResult{
		Signatures:      []signatureInformation{sigInfo},
		ActiveSignature: 0,
		ActiveParameter: active,
	}
}

// buildSignatureInfo turns "FUNCTION Foo(a AS INT, b AS STRING) AS T" into
// a structured signatureInformation with one parameterInformation per
// comma-separated entry inside the parentheses.
func buildSignatureInfo(sig string) signatureInformation {
	info := signatureInformation{Label: sig}
	open := strings.Index(sig, "(")
	close := strings.LastIndex(sig, ")")
	if open < 0 || close <= open {
		return info
	}
	inner := sig[open+1 : close]
	if strings.TrimSpace(inner) == "" {
		return info
	}
	for _, p := range strings.Split(inner, ",") {
		info.Parameters = append(info.Parameters, parameterInformation{
			Label: strings.TrimSpace(p),
		})
	}
	return info
}

// references returns every occurrence of the identifier under the cursor.
// Scope rules: if the cursor is inside a sub/function and the name is
// declared as a parameter or local DIM/STATIC of that sub, references
// are limited to that sub's body. Otherwise (top-level symbol or
// reference into a top-level symbol from anywhere), references span the
// whole document.
func (s *server) references(uri string, pos position) []location {
	prog, text, ok := s.parseDoc(uri)
	if !ok {
		return []location{}
	}
	word, ok := wordAt(text, pos)
	if !ok {
		return []location{}
	}
	return scopedRefs(prog, text, uri, word, pos)
}

// rename builds a WorkspaceEdit replacing each scoped reference of the
// identifier under the cursor with newName.
func (s *server) rename(uri string, pos position, newName string) any {
	prog, text, ok := s.parseDoc(uri)
	if !ok {
		return nil
	}
	word, ok := wordAt(text, pos)
	if !ok || newName == "" {
		return nil
	}
	locs := scopedRefs(prog, text, uri, word, pos)
	if len(locs) == 0 {
		return nil
	}
	edits := make([]textEdit, len(locs))
	for i, l := range locs {
		edits[i] = textEdit{Range: l.Range, NewText: newName}
	}
	return workspaceEdit{
		Changes: map[string][]textEdit{uri: edits},
	}
}

// scopedRefs filters findIdentRefs by the sub-local scope, when applicable.
// If `word` resolves to a local of the sub containing `pos`, only refs
// within that sub's line range are returned.
func scopedRefs(prog *parser.Program, text, uri, word string, pos position) []location {
	all := findIdentRefs(text, uri, word)
	if prog == nil {
		return all
	}
	subs := collectSubRanges(prog)
	var containing *subRange
	for i := range subs {
		if pos.Line >= subs[i].startLine && pos.Line <= subs[i].endLine {
			containing = &subs[i]
			break
		}
	}
	if containing == nil {
		return all
	}
	if !containing.locals[strings.ToUpper(word)] {
		return all
	}
	filtered := make([]location, 0, len(all))
	for _, l := range all {
		if l.Range.Start.Line >= containing.startLine && l.Range.Start.Line <= containing.endLine {
			filtered = append(filtered, l)
		}
	}
	if filtered == nil {
		return []location{}
	}
	return filtered
}

type subRange struct {
	name      string
	startLine int // 0-based
	endLine   int // 0-based, inclusive
	locals    map[string]bool
}

// collectSubRanges walks top-level sub/function/method statements and
// returns each one's line span and declared-local names. End line is
// estimated as the start line of the next top-level def minus 1, or
// the end of file for the last def.
func collectSubRanges(prog *parser.Program) []subRange {
	type item struct {
		name      string
		startLine int
		body      *parser.BlockStatement
		params    []*parser.Parameter
		recvName  string
	}
	var items []item
	maxLine := 0
	for _, st := range prog.Statements {
		switch s := st.(type) {
		case *parser.SubStatement:
			items = append(items, item{name: s.Name.Value, startLine: s.Token.Line - 1, body: s.Body, params: s.Params})
		case *parser.FunctionStatement:
			items = append(items, item{name: s.Name.Value, startLine: s.Token.Line - 1, body: s.Body, params: s.Params})
		case *parser.MethodStatement:
			recvName := ""
			if s.ReceiverName != nil {
				recvName = s.ReceiverName.Value
			}
			items = append(items, item{name: s.Name.Value, startLine: s.Token.Line - 1, body: s.Body, params: s.Params, recvName: recvName})
		}
		if l := topLineOf(st); l > maxLine {
			maxLine = l
		}
	}
	// Estimate end lines: just before the next item's start, or maxLine for the last.
	subs := make([]subRange, len(items))
	for i, it := range items {
		end := maxLine + 200 // generous fallback in case body extends beyond what we tracked
		if i+1 < len(items) {
			end = items[i+1].startLine - 1
		}
		locals := map[string]bool{}
		if it.recvName != "" {
			locals[strings.ToUpper(it.recvName)] = true
		}
		for _, p := range it.params {
			locals[strings.ToUpper(p.Name.Value)] = true
		}
		collectLocalDims(it.body, locals)
		subs[i] = subRange{name: it.name, startLine: it.startLine, endLine: end, locals: locals}
	}
	return subs
}

// topLineOf returns a generous line number for st; used only to bound
// the last sub's end line.
func topLineOf(st parser.Statement) int {
	switch s := st.(type) {
	case *parser.SubStatement:
		return s.Token.Line + 100
	case *parser.FunctionStatement:
		return s.Token.Line + 100
	case *parser.MethodStatement:
		return s.Token.Line + 100
	case *parser.TypeStatement:
		return s.Token.Line + 50
	}
	return 0
}

// collectLocalDims walks a block (and nested blocks) gathering names
// declared by DIM, STATIC, or LET, plus FOR loop variables.
func collectLocalDims(block *parser.BlockStatement, out map[string]bool) {
	if block == nil {
		return
	}
	for _, st := range block.Statements {
		switch s := st.(type) {
		case *parser.DimStatement:
			if s.Name != nil {
				out[strings.ToUpper(s.Name.Value)] = true
			}
		case *parser.LetStatement:
			if s.Name != nil {
				out[strings.ToUpper(s.Name.Value)] = true
			}
		case *parser.ForStatement:
			if s.Variable != nil {
				out[strings.ToUpper(s.Variable.Value)] = true
			}
			collectLocalDims(s.Body, out)
		case *parser.WhileStatement:
			collectLocalDims(s.Body, out)
		case *parser.DoLoopStatement:
			collectLocalDims(s.Body, out)
		case *parser.IfStatement:
			collectLocalDims(s.Consequence, out)
			for _, ei := range s.ElseIfs {
				collectLocalDims(ei.Consequence, out)
			}
			collectLocalDims(s.Alternative, out)
		case *parser.SelectStatement:
			for _, c := range s.Cases {
				collectLocalDims(c.Body, out)
			}
			collectLocalDims(s.Default, out)
		}
	}
}

// findIdentRefs scans the source text line-by-line for word-boundary,
// case-insensitive matches of `word`. Lines are stripped of string
// literals and comments first so matches inside `"Point"` or `' Point`
// are ignored. We use a text scan rather than the lexer because the
// lexer reports identifier Token positions as end-of-token, and resets
// to column 0 when the identifier ends on a newline — making both the
// reported line and column wrong for end-of-line identifiers.
func findIdentRefs(text, uri, word string) []location {
	target := strings.ToLower(word)
	tlen := len(target)
	if tlen == 0 {
		return []location{}
	}
	// Filter out matches when the word is a DBasic keyword — keywords
	// aren't user identifiers, so refs/rename on them is meaningless.
	if _, isKw := lexer.Keywords[strings.ToUpper(word)]; isKw {
		return []location{}
	}
	var locs []location
	lines := strings.Split(text, "\n")
	for lineNum, line := range lines {
		stripped := stripCommentsAndStrings(line)
		lcStripped := strings.ToLower(stripped)
		off := 0
		for off+tlen <= len(lcStripped) {
			j := strings.Index(lcStripped[off:], target)
			if j < 0 {
				break
			}
			j += off
			// Word-boundary check on the original line.
			before := byte(0)
			if j > 0 {
				before = line[j-1]
			}
			after := byte(0)
			if j+tlen < len(line) {
				after = line[j+tlen]
			}
			if isIdentByte(before) || isIdentByte(after) {
				off = j + 1
				continue
			}
			locs = append(locs, location{
				URI: uri,
				Range: rangePos{
					Start: position{Line: lineNum, Character: j},
					End:   position{Line: lineNum, Character: j + tlen},
				},
			})
			off = j + tlen
		}
	}
	if locs == nil {
		return []location{}
	}
	return locs
}

// stripCommentsAndStrings replaces every byte inside a string literal or a
// `'`-comment on the line with a space, preserving column alignment so
// caller indices into the line still match the original text.
func stripCommentsAndStrings(line string) string {
	out := make([]byte, len(line))
	copy(out, line)
	inString := false
	for i := 0; i < len(out); i++ {
		c := out[i]
		if inString {
			if c == '"' {
				inString = false
			}
			out[i] = ' '
			continue
		}
		if c == '"' {
			inString = true
			out[i] = ' '
			continue
		}
		if c == '\'' {
			for k := i; k < len(out); k++ {
				out[k] = ' '
			}
			break
		}
	}
	return string(out)
}

func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '_'
}

// completion returns completion items for the given cursor position.
// Two contexts are supported:
//   - After `.`: emit known field/method names. We don't yet resolve the
//     receiver's type, so we return all field/method names known across the
//     program. Editors will filter as the user types.
//   - Anywhere else: emit DBasic keywords plus all top-level def names.
func (s *server) completion(uri string, pos position) []completionItem {
	prog, text, ok := s.parseDoc(uri)
	if !ok {
		return []completionItem{}
	}

	if afterDot(text, pos) {
		return s.memberCompletions(prog, text, pos)
	}
	return s.identCompletions(prog)
}

func afterDot(text string, pos position) bool {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return false
	}
	line := lines[pos.Line]
	// Walk back over the partial identifier the user is typing, then check
	// for a `.` immediately before it.
	end := pos.Character
	if end > len(line) {
		end = len(line)
	}
	i := end
	for i > 0 {
		c := line[i-1]
		isIdent := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_'
		if !isIdent {
			break
		}
		i--
	}
	return i > 0 && line[i-1] == '.'
}

// memberCompletions returns field/method names. If the receiver's type
// can be resolved, results are narrowed to that type; otherwise we return
// the union of all known field/method names and let the editor filter.
func (s *server) memberCompletions(prog *parser.Program, text string, pos position) []completionItem {
	if recvType := resolveReceiverType(prog, text, pos); recvType != "" {
		if narrowed := completionsForType(prog, recvType); len(narrowed) > 0 {
			return narrowed
		}
	}
	return allMemberCompletions(prog)
}

// allMemberCompletions returns the union of every field/method name in the
// program — used as fallback when receiver type can't be resolved.
func allMemberCompletions(prog *parser.Program) []completionItem {
	idx := indexProgram(prog)
	seen := map[string]bool{}
	out := []completionItem{}
	for _, d := range idx.tree {
		if d.kind == skMethod {
			if !seen[d.name] {
				seen[d.name] = true
				out = append(out, completionItem{Label: d.name, Kind: ckMethod, Detail: d.sig})
			}
		}
		for _, f := range d.fields {
			if !seen[f.name] {
				seen[f.name] = true
				out = append(out, completionItem{Label: f.name, Kind: ckField, Detail: f.sig})
			}
		}
	}
	return out
}

// completionsForType returns field/method completions for the given DBasic
// type name (case-insensitive). Methods are matched by receiver type
// stripping any leading `POINTER TO`.
func completionsForType(prog *parser.Program, typeName string) []completionItem {
	upper := strings.ToUpper(typeName)
	var out []completionItem
	for _, st := range prog.Statements {
		switch s := st.(type) {
		case *parser.TypeStatement:
			if strings.ToUpper(s.Name.Value) != upper {
				continue
			}
			for _, f := range s.Fields {
				out = append(out, completionItem{
					Label:  f.Name.Value,
					Kind:   ckField,
					Detail: "DIM " + f.Name.Value + " AS " + typeStr(f.Type),
				})
			}
		case *parser.MethodStatement:
			if matchesReceiver(s.ReceiverType, upper) {
				out = append(out, completionItem{
					Label:  s.Name.Value,
					Kind:   ckMethod,
					Detail: methodSig(s),
				})
			}
		}
	}
	return out
}

func methodSig(m *parser.MethodStatement) string {
	recv := typeStr(m.ReceiverType)
	recvName := ""
	if m.ReceiverName != nil {
		recvName = m.ReceiverName.Value
	}
	return "FUNCTION (" + recvName + " AS " + recv + ") " + m.Name.Value + paramSig(m.Params) + retSig(m.ReturnTypes)
}

func matchesReceiver(ts *parser.TypeSpec, upperName string) bool {
	if ts == nil {
		return false
	}
	if ts.IsPointer {
		return matchesReceiver(ts.ElementType, upperName)
	}
	return strings.ToUpper(ts.Name) == upperName
}

// resolveReceiverType inspects the source line up to the cursor, finds the
// identifier just before the trigger `.`, and returns its declared DBasic
// type name. Looks at the containing sub's parameters and local DIMs
// first, then top-level DIMs and CONSTs. Returns "" if it can't resolve.
func resolveReceiverType(prog *parser.Program, text string, pos position) string {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	end := pos.Character
	if end > len(line) {
		end = len(line)
	}
	// Skip the partial identifier currently being typed.
	i := end
	for i > 0 && isIdentByte(line[i-1]) {
		i--
	}
	if i == 0 || line[i-1] != '.' {
		return ""
	}
	dotPos := i - 1
	// Receiver identifier just before the dot.
	j := dotPos
	for j > 0 && isIdentByte(line[j-1]) {
		j--
	}
	if j >= dotPos {
		return ""
	}
	recv := line[j:dotPos]
	return findIdentDeclaredType(prog, recv, pos)
}

func findIdentDeclaredType(prog *parser.Program, name string, pos position) string {
	upper := strings.ToUpper(name)
	// 1. Containing sub's parameters and locals.
	for _, st := range prog.Statements {
		body, params, recvName, recvType, startLine := extractSubInfo(st)
		if body == nil {
			continue
		}
		// Naïve range check: only sub bodies starting before pos.Line and
		// not a different earlier sub. We accept any sub that starts at or
		// before pos.Line, since collectSubRanges already iterates in order.
		if startLine > pos.Line {
			continue
		}
		// Receiver
		if recvName != "" && strings.EqualFold(recvName, name) {
			return typeStr(recvType)
		}
		// Params
		for _, p := range params {
			if strings.EqualFold(p.Name.Value, name) {
				return typeStr(p.Type)
			}
		}
		// Locals (recursive)
		if t := findLocalDimType(body, upper); t != "" {
			return t
		}
	}
	// 2. Top-level DIMs and CONSTs.
	for _, st := range prog.Statements {
		switch s := st.(type) {
		case *parser.DimStatement:
			if strings.EqualFold(s.Name.Value, name) {
				return typeStr(s.Type)
			}
		case *parser.ConstStatement:
			if strings.EqualFold(s.Name.Value, name) {
				return typeStr(s.Type)
			}
		}
	}
	return ""
}

func extractSubInfo(st parser.Statement) (body *parser.BlockStatement, params []*parser.Parameter, recvName string, recvType *parser.TypeSpec, startLine int) {
	switch s := st.(type) {
	case *parser.SubStatement:
		return s.Body, s.Params, "", nil, s.Token.Line - 1
	case *parser.FunctionStatement:
		return s.Body, s.Params, "", nil, s.Token.Line - 1
	case *parser.MethodStatement:
		rn := ""
		if s.ReceiverName != nil {
			rn = s.ReceiverName.Value
		}
		return s.Body, s.Params, rn, s.ReceiverType, s.Token.Line - 1
	}
	return nil, nil, "", nil, 0
}

// findLocalDimType walks a block (and nested control-flow bodies)
// searching for a DIM/LET of `upper` and returning its declared type.
func findLocalDimType(block *parser.BlockStatement, upper string) string {
	if block == nil {
		return ""
	}
	for _, st := range block.Statements {
		switch s := st.(type) {
		case *parser.DimStatement:
			if s.Name != nil && strings.ToUpper(s.Name.Value) == upper {
				return typeStr(s.Type)
			}
		case *parser.LetStatement:
			if s.Name != nil && strings.ToUpper(s.Name.Value) == upper {
				return ""
			}
		case *parser.ForStatement:
			if t := findLocalDimType(s.Body, upper); t != "" {
				return t
			}
		case *parser.WhileStatement:
			if t := findLocalDimType(s.Body, upper); t != "" {
				return t
			}
		case *parser.DoLoopStatement:
			if t := findLocalDimType(s.Body, upper); t != "" {
				return t
			}
		case *parser.IfStatement:
			if t := findLocalDimType(s.Consequence, upper); t != "" {
				return t
			}
			for _, ei := range s.ElseIfs {
				if t := findLocalDimType(ei.Consequence, upper); t != "" {
					return t
				}
			}
			if t := findLocalDimType(s.Alternative, upper); t != "" {
				return t
			}
		case *parser.SelectStatement:
			for _, c := range s.Cases {
				if t := findLocalDimType(c.Body, upper); t != "" {
					return t
				}
			}
			if t := findLocalDimType(s.Default, upper); t != "" {
				return t
			}
		}
	}
	return ""
}

// identCompletions returns keywords plus top-level identifiers.
func (s *server) identCompletions(prog *parser.Program) []completionItem {
	out := dbasicKeywordCompletions()
	idx := indexProgram(prog)
	for _, d := range idx.tree {
		out = append(out, completionItem{
			Label:  d.name,
			Kind:   defKindToCompletionKind(d.kind),
			Detail: d.sig,
		})
	}
	return out
}

func defKindToCompletionKind(k int) int {
	switch k {
	case skFunction:
		return ckFunction
	case skMethod:
		return ckMethod
	case skVariable:
		return ckVariable
	case skConstant:
		return ckConstant
	case skStruct:
		return ckClass
	case skField:
		return ckField
	}
	return ckText
}

// dbasicKeywordCompletions returns the canonical-cased keyword list.
// We pull from lexer.Keywords and filter out the lowercase aliases the
// lexer keeps for legacy reasons (Keywords is a map; iteration order is
// already non-deterministic, so we sort for stable output).
func dbasicKeywordCompletions() []completionItem {
	out := make([]completionItem, 0, len(lexer.Keywords))
	for kw := range lexer.Keywords {
		out = append(out, completionItem{
			Label: kw,
			Kind:  ckKeyword,
		})
	}
	return out
}

func (s *server) definition(uri string, pos position) any {
	prog, text, ok := s.parseDoc(uri)
	if !ok {
		return nil
	}
	word, found := wordAt(text, pos)
	if !found {
		return nil
	}
	idx := indexProgram(prog)
	d, ok := idx.flat[strings.ToUpper(word)]
	if !ok {
		return nil
	}
	return location{URI: uri, Range: identRange(d)}
}

// wordAt scans the source line at the LSP position (0-based line/char) and
// returns the identifier (letters/digits/underscore) under the cursor. We
// scan the raw text rather than relying on lexer Token columns, which are
// reported as end-positions for identifiers rather than start-positions.
func wordAt(text string, pos position) (string, bool) {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return "", false
	}
	line := lines[pos.Line]
	if pos.Character < 0 || pos.Character > len(line) {
		return "", false
	}

	isIdent := func(b byte) bool {
		return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '_'
	}

	start := pos.Character
	end := pos.Character

	// If we're past the end of an identifier (cursor right after last char),
	// step back one to land inside it.
	if start == len(line) || !isIdent(line[start]) {
		if start > 0 && isIdent(line[start-1]) {
			start--
		} else {
			return "", false
		}
	}
	for start > 0 && isIdent(line[start-1]) {
		start--
	}
	for end < len(line) && isIdent(line[end]) {
		end++
	}
	if start >= end {
		return "", false
	}
	return line[start:end], true
}

func paramSig(params []*parser.Parameter) string {
	parts := make([]string, 0, len(params))
	for _, p := range params {
		s := p.Name.Value + " AS " + typeStr(p.Type)
		if p.ByRef {
			s = "BYREF " + s
		}
		parts = append(parts, s)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func retSig(rets []*parser.TypeSpec) string {
	if len(rets) == 0 {
		return ""
	}
	if len(rets) == 1 {
		return " AS " + typeStr(rets[0])
	}
	parts := make([]string, len(rets))
	for i, r := range rets {
		parts[i] = typeStr(r)
	}
	return " AS (" + strings.Join(parts, ", ") + ")"
}

func typeStr(ts *parser.TypeSpec) string {
	if ts == nil {
		return ""
	}
	return ts.String()
}
