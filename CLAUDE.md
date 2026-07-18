# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Goals - THESE ARE ABSOLUTES!!!

1. DBasic is a Cross-Platform BASIC language transpiler, written entirely in Go.
2. DBasic maintains the simplicity of BASIC with the versatility and speed of Go.
3. DBasic is able to use ALL Go Packages.
4. VDBTerm is a clone of Visual Basic DOS Pro.
5. VDBTerm is written entirely in DBasic using github.com/charmbracelet packages for the TUI implementation.
6. VDBTerm will be able to write Cross-Platform TUI programs that are programmed in DBasic and have all of the benefits of DBasic.

## Build Commands

```bash
# Build the DBasic compiler itself (from repo root)
go build -o dbasic ./cmd/dbasic
go install ./cmd/dbasic

# Build the LSP server (for editor diagnostics)
go build -o dbasic-lsp ./cmd/dbasic-lsp
go install ./cmd/dbasic-lsp

# Check DBasic source for errors (no compilation)
dbasic check myfile.dbas

# Format DBasic source (prints to stdout; -w writes back, -l lists files needing formatting)
dbasic fmt myfile.dbas
dbasic fmt -w examples/*.dbas

# Generate Markdown API docs from a .dbas file (top-level decls + leading ' comments)
dbasic doc myfile.dbas              # prints Markdown to stdout
dbasic doc -o api.md myfile.dbas    # writes to api.md

# Run *_test.dbas files (subs named Test*); each runs in a recover() wrapper
dbasic test                          # current dir, recursive
dbasic test ./tests
dbasic test mypkg_test.dbas          # single file

# Build a DBasic program to executable
dbasic build myfile.dbas -o myprogram

# Compile and run immediately
dbasic run myfile.dbas

# View generated Go code (useful for debugging transpilation)
dbasic emit myfile.dbas

# Cross-compile (auto-disables CGO; .exe added for windows targets)
dbasic build myfile.dbas --target windows/amd64
dbasic build myfile.dbas --target linux/arm64

# Build using only cached Go modules (when proxy.golang.org is unreachable)
dbasic build myfile.dbas --offline

# Run compiler tests
go test ./...
```

## Compilation Chain

```
.dbas source → DBasic Compiler (lexer → parser → analyzer → codegen) → Go source → go build → native binary
```

The DBasic compiler transpiles `.dbas` files to Go, writes a temporary `main.go`, runs `go build`, and produces a native executable. Each `.dbas` program needs a `SUB Main()` entry point.

## Compiler Architecture (pkg/)

- **lexer/** - Tokenizes `.dbas` source into tokens. `token.go` defines all token types (keywords, operators, literals). Reserved words like `STEP`, `BYTES`, `STRING` cannot be used as identifiers.
- **parser/** - Parses tokens into an AST. `ast.go` defines all AST node types. Supports BASIC syntax (DIM, IF/THEN/ENDIF, FOR/NEXT, etc.) plus Go-inspired features (slices, struct literals, SPAWN).
- **analyzer/** - Semantic analysis: type checking, symbol resolution, scope validation. Enforces rules like "AND/OR require boolean operands" and "no DIM redeclaration in same scope."
- **preprocessor/** - Handles `INCLUDE` directives by inlining referenced files before parsing.
- **codegen/** - Generates Go source from the AST. Injects runtime helpers (Mid, Sng, RndInt, etc.) into the output. Each DBasic type maps to a Go type (INTEGER→int, DOUBLE→float64, BYTES→[]byte, etc.).
- **runtime/** - Runtime helper functions injected into generated code.
- **errors/** - Error reporting with source location tracking.

## DBasic Language Pitfalls

These are non-obvious behaviors that cause build failures. Always follow these rules when writing `.dbas` code:

1. **AND/OR/XOR/NOT accept boolean OR numeric operands.** Booleans are canonical; numeric operands use truthiness (`0` is false, non-zero is true). This applies to identifiers, literals, and arithmetic expressions. Function calls and struct field access are *not* auto-wrapped — Go's strict bool semantics still apply there. Use a comparison (`foo() = TRUE`, `foo() <> 0`) when you need to be explicit.

2. **STEP, BYTES, STRING, TYPE are reserved words** in identifier positions. Never use them as variable or field names. Use alternatives like `PX_STEP`, `EType`, `GmMode`. They *are* allowed as method names after a dot (`obj.String()`, `obj.Bytes()`, `obj.Type()`, `err.Error()`).

3. **DIM rebind is allowed only with the same type.** `DIM x AS INTEGER` then later `DIM x AS INTEGER = 5` is fine — it compiles to an assignment. Type-changing rebinds (`DIM x AS INTEGER` then `DIM x AS STRING`) still error.

4. **Sng() required for float32 arguments.** All Ebitengine `vector.DrawFilledRect`/`DrawFilledCircle` calls need `Sng()` wrapping for integer→float32 conversion.

5. **Unused variables cause Go build errors.** If you DIM a variable, you must use it. The generated Go code will fail `go build` otherwise.

6. **Mid() is 1-based.** `Mid(layout[y], x + 1, 1)` gets the character at 0-based position `x`.

7. **APPEND has two forms.** As an expression: `names = APPEND(names, "Alice")`. As a statement: `APPEND(names, "Alice")` — mutates in place.

8. **`DEFER` requires a function call**, not a statement. Wrap inline cleanup in a small helper sub if needed (`SUB Cleanup(label) ... END SUB`, then `DEFER Cleanup("...")`).

9. **`MAP OF K TO V` is auto-initialized.** `DIM m AS MAP OF STRING TO INTEGER` produces a writable empty map — no explicit `MAKE` call needed.

10. **`FOR i = 1 TO 10` auto-declares `i`** if it is not already in scope. Pre-declaring with `DIM i AS INTEGER` is still allowed (and useful when you want the loop variable to retain its post-loop value in the enclosing scope).

11. **Cross-compiling: use `dbasic build --target os/arch`.** Auto-disables CGO and appends `.exe` for windows targets. Examples: `--target windows/amd64`, `--target linux/arm64`, `--target darwin/arm64`. The legacy `dbasic emit` + manual `go build` path still works.

12. **Build offline: use `dbasic build --offline`** when proxy.golang.org is unreachable but you have the modules cached in `$GOMODCACHE`. Pins each non-stdlib import to the latest cached version compatible with the project's Go directive, then runs `go mod tidy` and `go build` with `GOPROXY=off GOSUMDB=off`.

13. **Go-build error messages reference your `.dbas` source.** Codegen emits `//line` directives, so `cannot use X as Y` and similar Go errors point to the original `.dbas:line`, not a temp `main.go`.

14. **Delve sees `.dbas` source directly.** Because of the `//line` directives, the standard Go debugger picks up DBasic source. Build, then `dlv exec ./prog` and set breakpoints with `break foo.dbas:42`. Run dlv from the directory containing the `.dbas` source.

15. **`dbasic-lsp` provides editor language support.** The LSP binary at `cmd/dbasic-lsp` runs over stdio and publishes: parse + analyze diagnostics; document symbols (file outline); hover (signature for top-level identifiers); go-to-definition; completions (keywords + top-level idents, member completions after `.`); find-references; rename. Wire it up in any LSP-aware editor (VS Code, Neovim, Helix, etc.). Note: rename and find-references are document-wide and do not honor scope — a local variable and a top-level symbol with the same name are treated as the same. Workable for top-level symbols (subs, functions, types, constants).

16. **`STEP`/`BYTES`/`STRING`/`TYPE` may now be used as identifiers** in declaration positions (DIM/LET/CONST/parameter/field/sub/function/method names) AND in expression contexts. The parser disambiguates statement-start `TYPE` based on what follows (`TYPE Foo` is still a type-decl; `TYPE = 5` and `TYPE.X()` are assignments/calls on a variable named TYPE).

17. **Explicit Go generic instantiation:** use `pkg.Func(OF T1, T2)(args)` when type inference can't resolve the type parameters. Example: `slices.Sort(OF []INTEGER)(nums)` emits `slices.Sort[[]int](nums)`.

18. **`dbasic fmt` is a token-stream formatter.** Comment-preserving, idempotent. Default prints to stdout; `-w` writes back; `-l` lists files whose formatting differs from the formatter's output.

19. **Classic-BASIC keywords supported.** REM is a comment-to-EOL alternative to `'`. REDIM and REDIM PRESERVE resize slices (`REDIM xs(N) AS T`). STATIC inside a SUB/FUNCTION declares a variable that persists across calls (codegen hoists it to a uniquified package-level var). SHARED is parsed and accepted as a no-op since DBasic already resolves identifiers up the scope chain. OPTION EXPLICIT is a no-op (already enforced); OPTION BASE 0|1 sets the array lower bound and rewrites every `xs[i]` access to `xs[i-1]` when set to 1 — don't combine with map-style indexing under OPTION BASE 1. DECLARE SUB/FUNCTION is parsed and discarded since forward references already auto-resolve. CALL is an optional prefix on call statements (stripped by the parser). LINE INPUT reads a whole line including spaces; the prompt is optional: `LINE INPUT "Name: ", n` or `LINE INPUT n`.

20. **`WITH obj ... END WITH`** allows `.field` shorthand inside the block. Parser pushes the receiver expression onto a WITH stack; `.field` parses as `receiver.field` directly (QB-style — no synthetic copy, so assignments through `.field` reach the original). Nested WITHs are supported via stack push/pop. Single-line `IF .. THEN x ELSE y` also parses now.

21. **`dbasic test [path]`** runs `*_test.dbas` files. Any `SUB Test*` is invoked in a `recover()` wrapper; failures (panics, including those raised by `PANIC()`) are caught and reported as FAIL with the panic message. Exits non-zero if any test fails.

22. **VS Code extension** lives at `vscode-dbasic/`. Provides syntax highlighting, snippets, and LSP integration. Build with `cd vscode-dbasic && npm install && npx vsce package`. Requires `dbasic-lsp` on PATH (or set `dbasic.lspPath` in settings; set `dbasic.lspEnabled` to `false` for grammar-only mode).

23. **Error handling is Go-idiomatic; there is NO try/catch.** The default pattern is multi-return + explicit check: `data, err = os.ReadFile("x"); IF err <> NIL THEN ...`. As sugar, `ONERR GOTO Label` / `ONERR GOFUNC Handler` instructs the compiler to *automatically* emit `if err != nil { goto Label }` / `if err != nil { Handler(err); return }` after every multi-assignment whose final value is of type ERROR. `ONERR GOTO 0` clears the active handler. ONERR is per-function and lexical — no defer/recover, no exceptions. When a function uses `ONERR GOTO`, codegen automatically hoists every non-`STATIC` `DIM` to the function top so Go's "goto cannot jump over a declaration" rule never bites — you can write DIMs anywhere in the body. `ONERR GOFUNC` emits a bare `return` after the handler, so it fits SUBs and FUNCTIONs whose signature accepts a bare return; for FUNCTIONs that must return concrete values, prefer ONERR GOTO. The full demo lives at `examples/onerr.dbas`; the formal language reference is in `docs/LANGUAGE_REFERENCE.md`.

24. **`INCLUDE` is confined to the project directory.** The preprocessor (`pkg/preprocessor`) rejects INCLUDE targets that escape the directory of the top-level `.dbas` file — absolute paths and `../` traversal both fail with "escapes the project directory". This is a path-traversal guard so `check`/`fmt`/`doc`/`emit` on an untrusted file can't read arbitrary files. All in-tree examples (DBtui, `examples/include`) use only in-project relative includes, so they're unaffected. The `--allow-external-includes` build/check flag (wired via `pp.SetAllowExternal`) opts out. The security posture of the whole toolchain is documented in the README's "Security / Trust Model" section.

25. **Directional channels, `<-ch` operator, and comma-ok RECEIVE.** `CHAN OF T` is bidirectional (`chan T`); prefix with `RECEIVE`/`SEND` for `RECEIVE CHAN OF T` (`<-chan T`) and `SEND CHAN OF T` (`chan<- T`). Naming the direction is required to capture a Go API that returns a directional channel — a bidirectional `CHAN OF T` variable is *not* assignable from a `<-chan T` (Go rejects it), and multi-assign emits `=` (no `:=` inference), so the receiving variable must be pre-declared with the right direction: `DIM winCh AS RECEIVE CHAN OF ssh.Window`. Receiving has two spellings: the `RECEIVE v FROM ch` / `RECEIVE v, ok FROM ch` statements, and the `<-ch` operator expression (`v = <-ch`, `v, ok = <-ch`, `foo(<-ch)`, `(<-ch) + 1`). The comma-ok flag goes FALSE once the channel is closed and drained. Lexer note: `<-` is tokenized whenever the `<` and `-` are adjacent (as in Go), so a comparison against a negative needs a space — write `x < -5`, not `x<-5`. See `examples/termserve/termserve.dbas` (`watchResize`) for real interop use.

26. **Variadic spread `xs...`.** Follow the final argument of a call with `...` to spread a slice into a variadic parameter, exactly like Go: `fmt.Println(xs...)`, `wish.NewServer(opts...)`. Only the last argument may be spread. This is the way to forward a runtime-built argument list (e.g. conditionally `APPEND`-ed options) to a variadic Go function. Note: `[]ANY{...}` composite literals are *not* accepted (the `ANY` keyword isn't a valid slice-literal element type) — build `[]ANY` slices with `DIM xs AS []ANY` + `APPEND` instead.

## VDBTerm (examples/vdbterm/)

The IDE itself, written in DBasic. Multi-file project using INCLUDE directives:

- `vdbterm.dbas` - Main entry point, Bubble Tea model
- `vdbterm_const.dbas` - Constants (control types, dialog modes)
- `vdbterm_types.dbas` - Type definitions (VDBProject, VDBForm, VDBControl)
- `vdbterm_codegen.dbas` - Code generation: transforms .vdbp projects to .dbas
- `vdbterm_project.dbas` - Project loading/saving
- `vdbterm_input.dbas` - Keyboard and mouse input handling
- `vdbterm_render.dbas` - UI rendering
- `vdbterm_helpers.dbas` - Utility functions
- `vdbterm_plugins.dbas` - Plugin system for custom widgets

```bash
# Build vdbterm
cd examples/vdbterm && dbasic build vdbterm.dbas -o vdbterm

# Run with a project
./vdbterm examples/file_dialog.vdbp
```

### VDBTerm Code Generation

`TransformEventCode()` in `vdbterm_codegen.dbas` transforms user event handler code, adding `m.` prefixes to control references. Each control type has specific property mappings. **Order of transformations matters** — some can corrupt others (e.g., `.Selected` matching within `.SelectedFile`).

## Example Programs (examples/)

All examples are commented for beginning BASIC programmers. Tutorial progression:

1. `hello.dbas` - Hello World, PRINT, INPUT
2. `variables.dbas` - All data types and operators
3. `control_flow.dbas` - IF, FOR, WHILE, DO-LOOP, SELECT CASE
4. `functions.dbas` - Functions, multiple return values, recursion
5. `arrays.dbas` - Array declaration, iteration, algorithms
6. `structs.dbas` - TYPE definitions, methods, pointer receivers
7. `pointers.dbas` - Address-of (@), dereference (^)
8. `bytes.dbas` - BYTES type, binary data
9. `errors.dbas` - Error handling, WrapError
10. `json.dbas` / `json_advanced.dbas` - JSON operations
11. `goroutines.dbas` - SPAWN, channels, SEND/RECEIVE
12. `channels.dbas` - Channels ELI5: SEND/RECEIVE + `<-` operator, directional channels, pipeline & worker-pool
13. `new_features.dbas` - Slices, APPEND, struct literals

Project examples: `gorillas/` (terminal game), `contacts/` (SQLite app), `edit/` (DOS editor clone), `tictactoe/` (web server), `curl_example/` (HTTP client), `pacman/` (Ebitengine game with audio/multiplayer), `keen3/` (Ebitengine platformer), `termserve/` (**TermApp** host — serves any DBasic TUI over authenticated SSH, written in DBasic; see `docs/REMOTE_SERVING.md`)

## Ebitengine Game Pattern

Games using `github.com/hajimehoshi/ebiten/v2` follow this pattern:

```basic
TYPE Game
    ' ... game state fields ...
END TYPE

FUNCTION (g AS POINTER TO Game) Update() AS ERROR
    ' game logic (called 60x/sec via SetTPS)
    RETURN NIL
END FUNCTION

SUB (g AS POINTER TO Game) Draw(screen AS POINTER TO ebiten.Image)
    ' rendering
END SUB

FUNCTION (g AS POINTER TO Game) Layout(ow AS INTEGER, oh AS INTEGER) AS (INTEGER, INTEGER)
    RETURN screenWidth, screenHeight
END FUNCTION

SUB Main()
    ebiten.SetTPS(60)
    DIM g AS Game = NewGame()
    ebiten.RunGame(@g)
END SUB
```

**Performance:** Pre-render static elements to offscreen `ebiten.NewImage()` to avoid redrawing every frame. Use `FALSE` for antialiasing on WSL/software renderers.

**Audio:** Use `audio.NewContext(44100)` (only ONE per program). Generate PCM data as strings via `Chr()`, feed to `audio.NewPlayer` through `strings.NewReader`. Cannot call `.Bytes()` or `.String()` due to keyword conflicts.

**Cross-platform speed:** Use accumulator-based movement (add speed% per tick, move when >=100) rather than frame-dependent speeds. Lock tick rate with `ebiten.SetTPS(60)`.

## Dependencies

- **Compiler:** Pure Go, no external dependencies beyond Go stdlib
- **VDBTerm:** github.com/charmbracelet/bubbletea, lipgloss, bubbles; github.com/atotto/clipboard
- **Games (pacman, keen3):** github.com/hajimehoshi/ebiten/v2 (requires system libs: libxrandr-dev, libgl1-mesa-dev, libasound2-dev on Linux)
