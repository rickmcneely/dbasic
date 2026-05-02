// dbasic-lsp is a minimal Language Server Protocol implementation for
// DBasic. It listens on stdin/stdout for JSON-RPC messages and provides
// real-time compile diagnostics (parse errors, semantic errors) for any
// editor that speaks LSP — VS Code, Neovim, Helix, Emacs, JetBrains.
//
// Currently supported:
//   - initialize / initialized / shutdown / exit handshake
//   - textDocument/didOpen / didChange / didSave / didClose
//   - textDocument/publishDiagnostics on parse and analyze errors
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
				"textDocumentSync": 1,
			},
			"serverInfo": map[string]string{
				"name":    "dbasic-lsp",
				"version": "0.1",
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
