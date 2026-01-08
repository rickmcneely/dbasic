# Contact Book - Win32 SQLite Example

A native Windows GUI contact manager demonstrating how to use Go packages (Walk, SQLite) from DBasic.

## Overview

This example shows how DBasic can integrate with Go's ecosystem by importing and using external packages. The contact manager uses:

- **github.com/lxn/walk** - Native Windows GUI toolkit (Win32 API wrapper)
- **modernc.org/sqlite** - Pure-Go SQLite database driver (no CGO required)

## Features

- Native Win32 GUI with modern visual styles
- SQLite database for persistent storage
- Contact list with sortable columns (click headers to sort)
- Add, edit, and delete contacts
- Search contacts by any field
- Pre-populated with 25 fictional TV character contacts

## Key DBasic Features Demonstrated

### Importing Go Packages

```basic
IMPORT "github.com/lxn/walk"
IMPORT "github.com/lxn/walk/declarative"
IMPORT "github.com/mattn/go-sqlite3"
```

### Using Win32 GUI Components

```basic
' TableView for listing contacts
DIM tableView AS walk.TableView
DIM model AS ContactModel

' Dialog for editing
DIM dlg AS walk.Dialog
DIM firstNameEdit AS walk.LineEdit
```

### Database Operations

```basic
' Open SQLite database
DIM db AS sql.DB
db = InitDatabase("contacts.db")

' Query contacts
DIM contacts AS []Contact
contacts = GetAllContacts(db, "last_name", TRUE)
```

## Building

### Prerequisites

- Windows OS (Walk uses Win32 API)
- Go 1.21 or later
- Optional: rsrc tool for embedding manifest (for modern visual styles)

### Build Steps

```bash
cd examples/contacts

# Download dependencies
go mod tidy

# Build the executable
go build -ldflags="-H windowsgui" -o contacts.exe

# Run
./contacts.exe
```

### With Manifest (for modern visual styles)

```bash
# Install rsrc tool
go install github.com/akavel/rsrc@latest

# Generate resource file from manifest
rsrc -manifest contacts.manifest -o rsrc.syso

# Build with embedded manifest
go build -ldflags="-H windowsgui" -o contacts.exe
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Ctrl+N | New contact |
| Enter | Edit selected contact |
| Delete | Delete contact (with confirmation) |
| F5 | Refresh list |
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
| `main.go` | Main application, Walk GUI, TableModel implementation |
| `database.go` | SQLite operations (CRUD functions) |
| `data.go` | TV character seed data (25 contacts) |
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

## Screenshot

```
+----------------------------------------------------------+
| Contact Book                                    [-][o][x] |
+----------------------------------------------------------+
| File   Edit   View   Help                                |
+----------------------------------------------------------+
| [New] [Edit] [Delete] [Refresh]        Search: [_______] |
+----------------------------------------------------------+
| First Name | Last Name  | City           | State | Phone |
|------------|------------|----------------|-------|-------|
| Al         | Bundy      | Chicago        | IL    | 555-  |
| Archie     | Bunker     | Queens         | NY    | 555-  |
| Cliff      | Huxtable   | Brooklyn       | NY    | 555-  |
| Don        | Draper     | New York       | NY    | 555-  |
| ...        | ...        | ...            | ...   | ...   |
+----------------------------------------------------------+
| 25 contacts                                              |
+----------------------------------------------------------+
```

## License

Part of the DBasic project - BASIC to Go transpiler.
