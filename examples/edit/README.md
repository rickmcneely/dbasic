# EDIT.COM Clone - Go Package Integration Example

A TUI text editor demonstrating how to use Go packages (Bubble Tea, Lip Gloss) from DBasic.

## Overview

This example shows how DBasic can integrate with Go's ecosystem by importing and using external packages. The editor uses:

- **github.com/charmbracelet/bubbletea** - Terminal UI framework
- **github.com/charmbracelet/lipgloss** - Style definitions

## Files

- `edit.dbas` - DBasic source using Go package imports
- `main.go` - Reference Go implementation for comparison
- `go.mod` / `go.sum` - Go module dependencies

## Key DBasic Features Demonstrated

### Importing Go Packages

```basic
IMPORT "github.com/charmbracelet/bubbletea" AS tea
IMPORT "github.com/charmbracelet/lipgloss"
```

### Using Go Types

```basic
DIM model AS tea.Model
DIM style AS lipgloss.Style
```

### Calling Go Package Functions

```basic
tea.NewProgram(model)
lipgloss.NewStyle().Background(lipgloss.Color("12"))
```

## Building

```bash
# From DBasic root directory
./dbasic build examples/edit/edit.dbas -o edit

# Run the editor
./edit [filename]
```

## Features

- Full-screen text editing with menu bar
- File operations: New, Open, Save, Save As
- Edit operations: Cut, Copy, Paste, Select All
- Search: Find, Find Next, Replace
- Keyboard shortcuts (Ctrl+S, Ctrl+O, Ctrl+Q, etc.)
- Classic blue theme reminiscent of MS-DOS EDIT

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Ctrl+N | New file |
| Ctrl+O | Open file |
| Ctrl+S | Save |
| Ctrl+Q | Quit |
| Ctrl+F | Find |
| Ctrl+H | Replace |
| F3 | Find Next |
| F10 | Menu |
| Esc | Cancel/Close dialog |
