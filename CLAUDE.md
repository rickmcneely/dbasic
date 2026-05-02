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

# Check DBasic source for errors (no compilation)
dbasic check myfile.dbas

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
12. `new_features.dbas` - Slices, APPEND, struct literals

Project examples: `gorillas/` (terminal game), `contacts/` (SQLite app), `edit/` (DOS editor clone), `tictactoe/` (web server), `curl_example/` (HTTP client), `pacman/` (Ebitengine game with audio/multiplayer), `keen3/` (Ebitengine platformer)

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
