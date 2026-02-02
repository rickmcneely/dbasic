# VDBTerm - Visual DBasic Terminal IDE

## Project Goals

<!-- TODO: Edit this section -->

---

## Overview

VDBTerm is a Visual Basic DOS Pro clone for terminal environments. It's an IDE that allows creating TUI (Terminal User Interface) programs written in DBasic.

## Compilation Chain

```
Go → DBasic Compiler → vdbterm → .vdbp Projects
```

1. **DBasic Compiler** - A BASIC-to-Go transpiler written in Go (`dbasic` command)
2. **vdbterm** - The IDE itself, written in DBasic and compiled to a Go binary
3. **.vdbp Projects** - Visual projects created in vdbterm, which generate DBasic code

## Project Structure

```
examples/vdbterm/
├── vdbterm.dbas          # Main entry point, Bubble Tea model
├── vdbterm_const.dbas    # Constants (control types, dialog modes, etc.)
├── vdbterm_types.dbas    # Type definitions (VDBProject, VDBForm, VDBControl)
├── vdbterm_codegen.dbas  # Code generation - transforms .vdbp to .dbas
├── vdbterm_project.dbas  # Project loading/saving, GenerateCode function
├── vdbterm_input.dbas    # Keyboard and mouse input handling
├── vdbterm_render.dbas   # UI rendering
├── vdbterm_helpers.dbas  # Utility functions (string manipulation, etc.)
├── vdbterm_plugins.dbas  # Plugin system for custom widgets
├── examples/             # Example vdbterm projects
│   ├── file_dialog.vdbp  # File dialog example project
│   └── frmFileDialog.frm # Form definition for file dialog
└── CLAUDE.md             # This file
```

## Key Files

### vdbterm_codegen.dbas
The code generator that transforms vdbterm projects into compilable DBasic code. Contains:
- `GenerateCode()` - Main entry point for code generation
- `TransformEventCode()` - Transforms user event handler code, adding `m.` prefixes to control references
- Widget-specific transformations for each control type (TEXTBOX, LISTBOX, FILES, etc.)

### vdbterm_types.dbas
Defines the data structures:
- `VDBProject` - Project metadata, forms list, theme
- `VDBForm` - Form with controls and event code
- `VDBControl` - Individual UI controls (buttons, textboxes, etc.)

### vdbterm_const.dbas
Control type constants:
- `CTRL_BUTTON = 1`, `CTRL_LABEL = 2`, `CTRL_TEXTBOX = 3`, etc.
- `CTRL_PATH = 20`, `CTRL_DIRECTORIES = 21`, `CTRL_FILES = 22`

## Debugging Workflow

### Testing Code Generation

1. Create a test script to generate code for a project:
```basic
' test_project.dbas
IMPORT "fmt" AS fmt
INCLUDE "vdbterm_const.dbas"
INCLUDE "vdbterm_types.dbas"
INCLUDE "vdbterm_helpers.dbas"
INCLUDE "vdbterm_plugins.dbas"
INCLUDE "vdbterm_project.dbas"
INCLUDE "vdbterm_codegen.dbas"

SUB Main()
    DIM registry AS PluginRegistry
    DIM proj AS VDBProject
    DIM errMsg AS STRING
    proj, errMsg = LoadProject("examples/file_dialog.vdbp")
    IF errMsg <> "" THEN
        fmt.Println("Error:", errMsg)
        RETURN
    ENDIF
    DIM code AS STRING = GenerateCode(proj, registry)
    fmt.Println(code)
END SUB
```

2. Build and run:
```bash
dbasic build test_project.dbas -o test_project
./test_project > /tmp/generated.dbas
```

3. Check for errors:
```bash
dbasic check /tmp/generated.dbas
dbasic build /tmp/generated.dbas -o /tmp/test_app
```

### Common Issues

**Control property transformations**: User code like `xFiles.Text` must be transformed to `m.xFiles.SelectedFile()`. Each control type has specific property mappings in `TransformEventCode()`.

**Order of transformations matters**: Some transformations can corrupt others (e.g., `.Selected` matching within `.SelectedFile`). Guard conditions may be needed.

## Build Commands

```bash
# Build vdbterm
dbasic build vdbterm.dbas -o vdbterm

# Run vdbterm with a project
./vdbterm examples/file_dialog.vdbp

# Check syntax only
dbasic check vdbterm.dbas

# View generated Go code
dbasic emit vdbterm.dbas
```

## Dependencies

- github.com/charmbracelet/bubbletea - TUI framework
- github.com/charmbracelet/lipgloss - Styling
- github.com/charmbracelet/bubbles - UI components (textarea, textinput, list)
- github.com/atotto/clipboard - Clipboard support
