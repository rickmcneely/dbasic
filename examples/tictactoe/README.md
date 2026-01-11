# Tic-Tac-Toe Web Server - Go Package Integration Example

A web-based tic-tac-toe game demonstrating how to build HTTP servers with DBasic using Go's `net/http` package.

## Overview

This example shows how DBasic can create a complete web application by importing and using Go's standard library packages:

- **net/http** - HTTP server and request handling
- **encoding/json** - JSON marshaling/unmarshaling
- **path/filepath** - File path operations
- **math/rand** - Random number generation

## Files

```
tictactoe/
├── tictactoe.dbas     # DBasic web server source
├── tictactoe          # Compiled executable
└── root/
    ├── index.html     # Main web page
    ├── css/
    │   └── style.css  # Styling
    ├── js/
    │   └── game.js    # Client-side game logic
    └── static/
        └── favicon.svg # Favicon
```

## Key DBasic Features Demonstrated

### Importing Go Packages

```basic
IMPORT "net/http" AS http
IMPORT "encoding/json" AS jsonpkg
IMPORT "fmt" AS fmt
```

### Using Go Types

```basic
DIM w AS http.ResponseWriter
DIM r AS POINTER TO http.Request
DIM cookie AS http.Cookie
```

### HTTP Handler Functions

```basic
SUB HandleNewGame(w AS http.ResponseWriter, r AS POINTER TO http.Request)
    w.Header().Set("Content-Type", "application/json")
    ' ... handle request
END SUB

' Register handlers
http.HandleFunc("/api/newgame", HandleNewGame)
http.ListenAndServe(":8080", NIL)
```

### JSON Marshaling

```basic
TYPE GameResponse
    DIM Board AS []STRING
    DIM Winner AS STRING
    DIM Wins AS INTEGER
END TYPE

DIM jsonData AS BYTES
jsonData, err = jsonpkg.Marshal(response)
w.Write(jsonData)
```

### Cookie Management

```basic
DIM cookie AS http.Cookie
cookie.Name = "tictactoe_record"
cookie.Value = "5-3-2"
cookie.MaxAge = 60 * 60 * 24 * 365
http.SetCookie(w, @cookie)
```

## Building

```bash
# From DBasic root directory
./dbasic build examples/tictactoe/tictactoe.dbas

# Move executable to example directory
mv tictactoe examples/tictactoe/

# Run the server
cd examples/tictactoe
./tictactoe
```

Then open http://localhost:8080 in your browser.

## Features

- Play tic-tac-toe against an AI opponent
- AI uses strategic play (wins if possible, blocks opponent wins, prefers center/corners)
- Randomly decides who starts each game (player or AI)
- Cookie-based record tracking (wins/losses/draws persisted across sessions)
- Reset record button
- Modern dark theme UI
- Responsive design

## Game API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/newgame` | GET | Start a new game |
| `/api/move` | POST | Make a move |
| `/api/reset` | GET | Reset win/loss record |
| `/*` | GET | Serve static files |

## Note on JSON Field Names

DBasic generates Go structs without custom JSON tags, so JSON field names use PascalCase (e.g., `Board`, `PlayerSymbol`) rather than camelCase. Client-side JavaScript must use these PascalCase names when parsing responses.
