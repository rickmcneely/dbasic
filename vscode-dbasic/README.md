# DBasic Language Support for Visual Studio Code

This extension provides syntax highlighting, code snippets, and language configuration for DBasic - a modern BASIC-to-Go transpiler.

## Features

- **Syntax Highlighting** - Full syntax highlighting for DBasic source files (`.dbas`, `.dbasic`)
- **Code Snippets** - Quick snippets for common patterns:
  - `sub`, `main`, `func` - Function definitions
  - `dim`, `let`, `const` - Variable declarations
  - `if`, `ifelse`, `for`, `while` - Control flow
  - `select` - Select Case statement
  - `chan`, `spawn`, `send`, `receive` - Concurrency
  - `json`, `ptr`, `import` - Advanced features
- **Bracket Matching** - Auto-closing and matching for brackets and quotes
- **Code Folding** - Fold SUB/FUNCTION/IF/FOR/WHILE/SELECT blocks
- **Auto-Indentation** - Smart indentation for code blocks

## Installation

### From Source (Development)

1. Copy the `vscode-dbasic` folder to your VS Code extensions directory:
   - **Windows**: `%USERPROFILE%\.vscode\extensions\`
   - **macOS**: `~/.vscode/extensions/`
   - **Linux**: `~/.vscode/extensions/`

2. Restart VS Code

### From VSIX Package

```bash
cd vscode-dbasic
npm install -g vsce
vsce package
code --install-extension dbasic-0.1.0.vsix
```

## Usage

1. Create a file with `.dbas` or `.dbasic` extension
2. Start typing DBasic code - syntax highlighting is automatic
3. Use snippets by typing the prefix and pressing Tab:
   - Type `main` + Tab for a Main subroutine template
   - Type `for` + Tab for a FOR loop template
   - Type `if` + Tab for an IF statement template

## Snippet Prefixes

| Prefix | Description |
|--------|-------------|
| `sub` | Subroutine definition |
| `main` | Main subroutine |
| `func` | Function with return type |
| `funcmulti` | Function with multiple returns |
| `dim` | Variable declaration |
| `dims` | String variable |
| `dima` | Array declaration |
| `let` | Variable with type inference |
| `const` | Constant declaration |
| `if` | If statement |
| `ifelse` | If-Else statement |
| `ifelseif` | If-ElseIf-Else statement |
| `for` | For loop |
| `forstep` | For loop with step |
| `while` | While loop |
| `dowhile` | Do-While loop |
| `dountil` | Do-Until loop |
| `select` | Select Case |
| `print` | Print statement |
| `printv` | Print with variable |
| `input` | Input statement |
| `json` | JSON object |
| `chan` | Channel declaration |
| `spawn` | Spawn goroutine |
| `send` | Send to channel |
| `receive` | Receive from channel |
| `ptr` | Pointer declaration |
| `import` | Import Go package |
| `importas` | Import with alias |

## Language Features

DBasic supports:
- Strong typing with INTEGER, LONG, SINGLE, DOUBLE, STRING, BOOLEAN, JSON
- SUB and FUNCTION with multiple return values
- Pointers with `@` (address-of) and `^` (dereference)
- Goroutines via `SPAWN` and channels
- Native JSON type with dot notation
- Go package integration via `IMPORT`

## Requirements

- Visual Studio Code 1.75.0 or higher

## Release Notes

### 0.1.0

- Initial release
- Syntax highlighting for DBasic
- Code snippets
- Language configuration
