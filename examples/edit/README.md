# EDIT.COM Clone

A TUI text editor inspired by MS-DOS EDIT.COM, built with Go and Bubble Tea.

## Files

- `main.go` - Full-featured Go TUI editor using Bubble Tea
- `edit.dbas` - Simple DBasic wrapper
- `edit_enhanced.dbas` - Enhanced DBasic editor implementation

## Features

- Full-screen text editing with menu bar
- File operations: New, Open, Save, Save As
- Edit operations: Cut, Copy, Paste, Select All
- Search: Find, Find Next, Replace
- Keyboard shortcuts (Ctrl+S, Ctrl+O, Ctrl+Q, etc.)
- Classic blue theme reminiscent of MS-DOS EDIT

## Building

```bash
# Build the Go version directly
cd examples/edit
go build -o edit main.go

# Or build the DBasic version
cd ../..
./dbasic build examples/edit/edit_enhanced.dbas -o edit_dbas
```

## Running

```bash
./edit [filename]
```

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
