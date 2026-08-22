# DBasic

A modern BASIC-to-Go transpiler with native JSON support, goroutines, channels, and Go package integration.

## Features

- **BASIC Syntax**: Familiar BASIC-style programming with labels instead of line numbers
- **Go Transpilation**: Compiles to Go source code for cross-platform executables
- **Cross-Compilation**: `dbasic build --target windows/amd64`, `linux/arm64`, `darwin/arm64`, etc.
- **Type System**: Strong typing with INTEGER, LONG, SINGLE, DOUBLE, STRING, BOOLEAN, JSON, BYTES, MAP
- **Slices**: Go-style dynamic arrays with `[]TYPE` syntax, APPEND, REDIM/PRESERVE, slice operations
- **Maps**: Typed `MAP OF K TO V` with auto-initialization
- **Structs**: User-defined types with TYPE/END TYPE, methods, embedding, and struct literals
- **Functions**: SUB and FUNCTION with multi-return, function literals, closures, DEFER
- **Pointers**: Go-style pointer operations with `@` (address-of) and `^` (dereference)
- **Concurrency**: Goroutines via `SPAWN`, channels with `SEND` and `RECEIVE`
- **JSON Support**: Native JSON type with dot notation access and helper functions
- **HTTP / Shell**: Built-in HTTP client (CurlGet/Post/Fetch) and shell execution (Shell/Exec/Start)
- **Go Integration**: Import and use any Go package; explicit generic instantiation via `(OF T)`
- **Classic BASIC**: REM, REDIM/PRESERVE, STATIC, SHARED, OPTION EXPLICIT/BASE, DECLARE, CALL, LINE INPUT, WITH, single-line IF/ELSE
- **Tooling**: `dbasic fmt` formatter, `dbasic doc` Markdown generator, `dbasic test` runner with recover()
- **Editor Support**: `dbasic-lsp` provides diagnostics, hover, completions, definition, references, rename, signatureHelp, document outline
- **VS Code Extension**: Syntax highlighting, snippets, and LSP integration in `vscode-dbasic/`
- **Debugger**: Delve sees `.dbas:line` directly via `//line` source-map directives

## Installation

```bash
# Clone the repository
git clone https://github.com/zditech/dbasic.git
cd dbasic

# Build the compiler
go build -o dbasic ./cmd/dbasic

# Optionally, install to your Go bin
go install ./cmd/dbasic
```

## Quick Start

Create a file `hello.dbas`:

```basic
SUB Main()
    PRINT "Hello, World!"

    DIM name AS STRING
    INPUT "What is your name? "; name
    PRINT "Hello, "; name; "!"
END SUB
```

Run it:

```bash
dbasic run hello.dbas
```

Build an executable:

```bash
dbasic build hello.dbas
./hello
```

## Usage

```
dbasic <command> [options] [arguments]

Commands:
  build <file.dbas>     Compile to executable
  run <file.dbas>       Compile and run immediately
  emit <file.dbas>      Output generated Go code to stdout
  check <file.dbas>     Check for errors without compiling
  fmt <file.dbas>...    Reformat source (use -w to write, -l to list)
  doc <file.dbas>       Emit Markdown API docs (use -o to write to file)
  test [path]           Run *_test.dbas files (subs named Test*)
  version               Print version
  help                  Print help

Options:
  -o <file>             Output file name (for build, doc)
  -debug                Include source line comments in output
  -v                    Verbose output
  --target os/arch      Cross-compile (e.g. windows/amd64, linux/arm64)
  --offline             Build using only cached Go modules
  --allow-external-includes
                        Permit INCLUDE outside the project dir (off by default)
  -w                    Write formatted output back to file (fmt only)
  -l                    List files needing formatting (fmt only)
```

## Security / Trust Model

DBasic is a compiler-and-runtime, so treat a `.dbas` file the way you would treat a shell script: **only build or run sources you trust.** The security properties are:

- **`build`, `run`, and `test` execute code.** They transpile to Go, run `go mod tidy` (which may **download the Go modules the program imports** from the network), and then execute the result. A hostile `.dbas` can import any Go package and run arbitrary commands — this is inherent to the language's goal of using *all* Go packages, not a bug. Do not `dbasic run` a file from an untrusted source.
- **`check`, `fmt`, `emit`, and `doc` do not execute your program** and do not fetch modules. They only read the source (and any files it `INCLUDE`s). The `dbasic-lsp` language server likewise never executes code, fetches modules, or follows `INCLUDE` — it analyzes the open document in memory. These are safe to run on files you have not vetted.
- **`INCLUDE` is confined to the project directory.** By default a `.dbas` file may only `INCLUDE` files at or below the directory of the top-level file passed on the command line. Absolute paths and `../` traversal (e.g. `INCLUDE "/etc/passwd"`) are rejected, so merely *checking* or *formatting* an untrusted file cannot be used to read arbitrary files off your disk. Pass `--allow-external-includes` to opt out when you have a legitimate need to include files from outside the project. (Confinement is lexical; it does not resolve symlinks that already live inside the project.)
- **No shell injection in the toolchain.** The compiler invokes `go` with explicit arguments (never a shell string), writes generated code to a private `0700` temp directory, and applies cross-compile targets via `GOOS`/`GOARCH` environment variables. Generated Go string literals are escaped, so source text cannot break out into generated code.
- **`Shell()` / `ShellExec()` are your program's responsibility.** These language built-ins run commands on behalf of the *compiled program* (via `/bin/sh -c` for `Shell`). If your program passes untrusted data to them, quote/validate it yourself — `ShellExec` splits arguments on whitespace and does no quoting.

### Editor Support

`dbasic-lsp` is a Language Server (over stdio) for any LSP-aware editor. It provides:

- Live parse and analyze diagnostics
- Document outline (file symbols)
- Hover with signatures for top-level identifiers
- Go-to-definition for top-level decls
- Find references and rename (scope-aware: locals stay within their sub)
- Completions (keywords + top-level + type-narrowed members after `.`)
- Signature help inside function calls

Build it with `go install ./cmd/dbasic-lsp`. See `vscode-dbasic/` for the VS Code package.

## Language Overview

### Variable Declarations

```basic
' Explicit type declaration
DIM count AS INTEGER = 0
DIM name AS STRING = "DBasic"
DIM price AS DOUBLE = 19.99
DIM active AS BOOLEAN = TRUE

' Type inference with LET
LET x = 42          ' Inferred as INTEGER
LET message = "Hi"  ' Inferred as STRING
```

### Data Types

| DBasic Type | Go Type | Description |
|-------------|---------|-------------|
| INTEGER | int | Platform integer |
| LONG | int64 | 64-bit integer |
| SINGLE | float32 | 32-bit float |
| DOUBLE | float64 | 64-bit float |
| STRING | string | Text string |
| BOOLEAN | bool | TRUE or FALSE |
| JSON | map[string]interface{} | JSON object |
| []X | []X | Slice (dynamic array) |
| POINTER TO X | *X | Pointer type |
| CHAN OF X | chan X | Channel type |

### Control Flow

```basic
' IF statement
IF score >= 90 THEN
    PRINT "A"
ELSEIF score >= 80 THEN
    PRINT "B"
ELSE
    PRINT "C"
ENDIF

' FOR loop
FOR i = 1 TO 10 STEP 2
    PRINT i
NEXT

' WHILE loop
WHILE x > 0
    x = x - 1
WEND

' DO-LOOP
DO
    counter = counter + 1
LOOP WHILE counter < 10

' SELECT CASE
SELECT CASE day
CASE 1
    PRINT "Monday"
CASE 6, 7
    PRINT "Weekend"
CASE ELSE
    PRINT "Weekday"
END SELECT
```

### Functions and Subroutines

```basic
' Subroutine (no return value)
SUB Greet(name AS STRING)
    PRINT "Hello, "; name; "!"
END SUB

' Function with return value
FUNCTION Square(n AS INTEGER) AS INTEGER
    RETURN n * n
END FUNCTION

' Multiple return values
FUNCTION Divide(a AS INTEGER, b AS INTEGER) AS (INTEGER, BOOLEAN)
    IF b = 0 THEN
        RETURN 0, FALSE
    ENDIF
    RETURN a / b, TRUE
END FUNCTION

' Using multiple returns
DIM result AS INTEGER
DIM ok AS BOOLEAN
result, ok = Divide(10, 2)
```

### Pointers

```basic
DIM x AS INTEGER = 42
DIM ptr AS POINTER TO INTEGER = @x  ' Address-of

PRINT ^ptr      ' Dereference: prints 42
^ptr = 100      ' Modify through pointer
PRINT x         ' Prints 100
```

### Goroutines and Channels

```basic
' Create a channel
DIM ch AS CHAN OF INTEGER = MAKE_CHAN(INTEGER, 10)

' Send to channel
SEND 42 TO ch

' Receive from channel
DIM value AS INTEGER
RECEIVE value FROM ch

' Start a goroutine
SPAWN Worker(ch)
```

### Slices and Structs

```basic
' Define a struct type
TYPE Person
    DIM Name AS STRING
    DIM Age AS INTEGER
END TYPE

' Slice declaration
DIM names AS []STRING
names = ["Alice", "Bob", "Charlie"]

' APPEND to slice
names = APPEND(names, "David")

' Slice operations
DIM first2 AS []STRING
first2 = names[0:2]     ' ["Alice", "Bob"]

' Struct literals
DIM p AS Person
p = Person{Name: "John", Age: 30}

' Slice of structs
DIM people AS []Person
people = APPEND(people, Person{Name: "Alice", Age: 25})
people = APPEND(people, Person{Name: "Bob", Age: 35})
PRINT people[0].Name    ' "Alice"
```

### JSON Support

```basic
DIM config AS JSON = {name: "app", version: 1, enabled: TRUE}

' Access with dot notation
PRINT config.name
PRINT config.version

' Modify values
config.enabled = FALSE

' JSON helper functions
DIM data AS JSON = JSONParse("{\"key\": \"value\"}")
DIM value AS ANY = JSONGet(data, "key")
JSONSet(data, "newKey", 123)
PRINT JSONPretty(data)

' Convert between structs and JSON
DIM jsonData AS JSON = StructToJSON(myStruct)
myStruct = JSONToStruct(jsonData, @myStruct)
```

### Shell Commands

Execute external programs and shell commands:

```basic
' Run a shell command (via /bin/sh -c)
DIM output AS STRING
DIM exitCode AS INTEGER
DIM errMsg AS STRING

output, exitCode, errMsg = Shell("ls -la")
IF exitCode = 0 THEN
    PRINT output
ELSE
    PRINT "Error: " & errMsg
ENDIF

' Execute a program directly with arguments
output, exitCode, errMsg = ShellExec("git", "status --short")

' Start a background process
DIM pid AS INTEGER
pid, errMsg = ShellStart("python -m http.server 8080")
PRINT "Server started with PID: " & Str(pid)
```

### HTTP Requests

Built-in HTTP client functions:

```basic
DIM response AS STRING
DIM statusCode AS INTEGER
DIM errMsg AS STRING

' Simple GET request
response, statusCode, errMsg = CurlGet("https://api.example.com/data")

' POST request with body
response, statusCode, errMsg = CurlPost("https://api.example.com/submit", "{\"name\": \"test\"}")

' Custom HTTP method
response, statusCode, errMsg = CurlFetch("https://api.example.com/resource", "PUT", "{\"updated\": true}")

IF errMsg = "" AND statusCode = 200 THEN
    PRINT "Response: " & response
ENDIF
```

### Go Package Integration

```basic
IMPORT "fmt"
IMPORT "strings"

SUB Main()
    DIM text AS STRING = "hello world"
    PRINT strings.ToUpper(text)
    fmt.Printf("Value: %d\n", 42)
END SUB
```

### Built-in Functions

DBasic includes 60+ built-in functions:

| Category | Functions |
|----------|-----------|
| **String** | Len, Left, Right, Mid, Instr, UCase, LCase, Trim, Replace, Space, Chr, Asc |
| **Math** | Abs, Sqr, Sin, Cos, Tan, Log, Exp, Pow, Min, Max, Round, Floor, Ceil |
| **Conversion** | Str, Val, Int, Lng, Sng, Dbl, Bool |
| **Random** | Rnd, RndInt, RndRange, Randomize |
| **Date/Time** | Now, Date, Year, Month, Day, Hour, Minute, Second, Timer, Sleep |
| **File** | FileExists, ReadFile, WriteFile, AppendFile, DeleteFile, MkDir, RmDir |
| **JSON** | JSONParse, JSONStringify, JSONPretty, JSONGet, JSONSet, StructToJSON, JSONToStruct |
| **HTTP** | CurlGet, CurlPost, CurlFetch |
| **Shell** | Shell, ShellExec, ShellStart |
| **Bytes** | Encode, Decode, MakeBytes, LenBytes |
| **Error** | NewError, Errorf, WrapError |
| **Format** | Printf, Sprintf |

## Examples

The `examples/` directory contains sample programs:

### Basic Examples
- `hello.dbas` - Hello World with input
- `variables.dbas` - Variable types and operations
- `control_flow.dbas` - IF, FOR, WHILE, SELECT CASE
- `arrays.dbas` - Array operations
- `functions.dbas` - Functions and multiple returns
- `structs.dbas` - User-defined types
- `pointers.dbas` - Pointer operations
- `json.dbas` - JSON support
- `bytes.dbas` - Byte arrays and BSTRING
- `goroutines.dbas` - Concurrency with SPAWN and channels
- `new_features.dbas` - Slices, struct literals, APPEND

### Application Examples
- `edit/` - MS-DOS EDIT clone using Bubble Tea (TUI)
- `contacts/` - Win32 GUI contacts app using Walk
- `curl_example/` - HTTP client demo with CurlGet/CurlPost
- `vdbterm/` - Visual Basic-style IDE for terminal UI apps

### Games (`examples/games/`)
- `mines/` - Minesweeper, a faithful clone of the Windows classic (Ebitengine)
- `pacman/` - PacMan with audio, ghosts and 2-player modes (Ebitengine)
- `gorillas/` - The QBasic artillery game, in the terminal
- `tictactoe/` - Web server tic-tac-toe game with cookies

Run an example:

```bash
dbasic run examples/hello.dbas
```

## VS Code Extension

The `vscode-dbasic/` directory contains a VS Code extension that bundles:

- Syntax highlighting for `.dbas` / `.dbasic` files (TextMate grammar)
- Code snippets for common patterns (`sub`, `main`, `func`, `if`, `for`, `while`, etc.)
- Language configuration (comments, brackets, indent rules, code folding)
- LSP integration via `dbasic-lsp` (diagnostics, hover, completions, definition, references, rename, signature help, document outline)

To build and install:

```sh
cd vscode-dbasic
npm install
npx vsce package
code --install-extension dbasic-0.5.0.vsix
```

Make sure `dbasic-lsp` is on your PATH (`go install ./cmd/dbasic-lsp`), or set `dbasic.lspPath` in VS Code settings. Set `dbasic.lspEnabled` to `false` for grammar-only mode.

## Documentation

See [docs/LANGUAGE_REFERENCE.md](docs/LANGUAGE_REFERENCE.md) for a comprehensive language reference covering:

- All data types and operators
- Control flow statements
- Functions and subroutines
- User-defined types and methods
- Concurrency with goroutines and channels
- 60+ built-in functions
- Go package integration

## Project Structure

```
DBasic/
├── cmd/
│   ├── dbasic/         # Main compiler CLI
│   └── dbasic-lsp/     # Language server (stdio JSON-RPC)
├── pkg/
│   ├── lexer/          # Tokenizer
│   ├── parser/         # Parser and AST
│   ├── analyzer/       # Semantic analysis
│   ├── codegen/        # Go code generator
│   ├── formatter/      # Token-stream source formatter
│   ├── preprocessor/   # INCLUDE directive handling
│   ├── runtime/        # Runtime helper functions
│   └── errors/         # Error reporting with source locations
├── docs/               # Language reference and integration guides
├── examples/           # Example programs (and VDBTerm IDE)
├── vscode-dbasic/      # VS Code extension (syntax + snippets + LSP integration)
└── README.md
```

## Building from Source

```bash
# Run tests
go test ./...

# Build
go build -o dbasic ./cmd/dbasic

# Install
go install ./cmd/dbasic
```

## License

MIT License

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
