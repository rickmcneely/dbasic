# Contact Book - Win32 SQLite Example

A native Windows GUI contact manager written entirely in DBasic, demonstrating Go interface implementation with Walk and SQLite.

## Overview

This example shows how DBasic can implement Go interfaces and integrate with Go's ecosystem. The contact manager uses:

- **github.com/lxn/walk** - Native Windows GUI toolkit (Win32 API wrapper)
- **modernc.org/sqlite** - Pure-Go SQLite database driver (no CGO required)

## Features

- Native Win32 GUI with modern visual styles
- SQLite database for persistent storage
- Contact list with sortable columns (click headers to sort)
- Add, edit, and delete contacts
- Pre-populated with 25 fictional TV character contacts
- **Written entirely in DBasic** - no hand-written Go code required

## Key DBasic Features Demonstrated

### Go Interface Implementation

```basic
' Embed base types for interface implementation
TYPE ContactModel
    EMBED walk.TableModelBase
    EMBED walk.SorterBase
    DIM SortCol AS INTEGER
    DIM Items AS []POINTER TO Contact
END TYPE

' Implement TableModel interface methods
FUNCTION (m AS POINTER TO ContactModel) RowCount() AS INTEGER
    RETURN LEN((^m).Items)
END FUNCTION

FUNCTION (m AS POINTER TO ContactModel) Value(row AS INTEGER, col AS INTEGER) AS ANY
    ' Return interface{} for Walk's TableView
    RETURN (^item).FirstName
END FUNCTION

FUNCTION (m AS POINTER TO ContactModel) Sort(col AS INTEGER, order AS walk.SortOrder) AS ERROR
    ' Return error type
    RETURN NIL
END FUNCTION
```

### Declarative GUI with Struct Literals

```basic
err = declarative.MainWindow{
    AssignTo: @gMainWindow,
    Title: "Contact Book",
    MinSize: declarative.Size{Width: 600, Height: 400},
    Layout: declarative.VBox{},
    Children: []declarative.Widget{
        declarative.TableView{
            AssignTo: @tableView,
            Model: gModel,
            Columns: []declarative.TableViewColumn{
                declarative.TableViewColumn{Title: "First Name", Width: 100}
            }
        }
    }
}.Create()
```

### Line Continuation

```basic
DIM query AS STRING = "CREATE TABLE IF NOT EXISTS contacts (" + _
    "id INTEGER PRIMARY KEY AUTOINCREMENT, " + _
    "first_name TEXT NOT NULL)"
```

## Building

### From Linux (Cross-compile to Windows)

```bash
# Transpile DBasic to Go
cd ~/DBasic
./dbasic emit examples/contacts/contacts.dbas > examples/contacts/main.go

# Copy to Windows-accessible directory and build
cp examples/contacts/{main.go,go.mod,go.sum,contacts.manifest} /mnt/c/temp/contacts/

# Build on Windows (from WSL)
cd /mnt/c/temp/contacts
"/mnt/c/Program Files/Go/bin/go.exe" install github.com/akavel/rsrc@latest
/mnt/c/Users/YourUser/go/bin/rsrc.exe -manifest contacts.manifest -o rsrc.syso
"/mnt/c/Program Files/Go/bin/go.exe" build -ldflags="-H windowsgui" -o contacts.exe
```

### From Windows

```cmd
cd C:\path\to\contacts

REM Install rsrc for manifest embedding
go install github.com/akavel/rsrc@latest

REM Generate resource file
rsrc -manifest contacts.manifest -o rsrc.syso

REM Build
go build -ldflags="-H windowsgui" -o contacts.exe

REM Run
contacts.exe
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Ctrl+N | New contact |
| Ctrl+E | Edit selected contact |
| Delete | Delete contact (with confirmation) |
| Alt+F4 | Exit |

## Sample Data

The database is pre-populated with 25 fictional TV character contacts including:

| Character | Address | City | State |
|-----------|---------|------|-------|
| Herman Munster | 1313 Mockingbird Lane | Mockingbird Heights | CA |
| Gomez Addams | 0001 Cemetery Lane | Westfield | NJ |
| Fred Flintstone | 301 Cobblestone Way | Bedrock | |
| Homer Simpson | 742 Evergreen Terrace | Springfield | |
| Al Bundy | 9764 Jeopardy Lane | Chicago | IL |
| Walter White | 308 Negra Arroyo Lane | Albuquerque | NM |
| ... and 19 more | | | |

## Files

| File | Description |
|------|-------------|
| `contacts.dbas` | DBasic source (complete application) |
| `main.go` | Generated Go code (transpiled from contacts.dbas) |
| `contacts.manifest` | Windows application manifest for visual styles |
| `go.mod` | Go module dependencies |

## Database Schema

```sql
CREATE TABLE contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    address TEXT,
    city TEXT,
    state TEXT,
    zip TEXT,
    phone TEXT,
    email TEXT
);
```

## License

Part of the DBasic project - BASIC to Go transpiler.
