# Go Package Integration - Analysis and Proposals

## Current State

DBasic currently supports importing Go packages with the `IMPORT` statement:

```basic
IMPORT "fmt"
IMPORT "strings"
IMPORT "net/http" AS http
```

Package functions can be called using dot notation:

```basic
fmt.Println("Hello")
strings.ToUpper(text)
```

## Current Limitations

### 1. No Type Checking for Go Calls

The DBasic analyzer cannot validate Go package function calls because it doesn't know:
- What functions/types the package exports
- Function parameter types and counts
- Return types

**Result**: Errors are only caught at Go compile time, giving poor error messages to DBasic users.

### 2. Type Mapping Uncertainty

Go types don't always map cleanly to DBasic types:
- `int` vs `int32` vs `int64`
- `[]byte` vs `string`
- `interface{}` and generics
- Structs and custom types

### 3. Error Handling Pattern

Go's idiomatic `(value, error)` return pattern isn't directly supported:

```go
// Go pattern
data, err := ioutil.ReadFile("test.txt")
if err != nil {
    // handle error
}
```

### 4. Struct Types

Can't easily work with Go struct types:

```go
// How to create/use http.Request in DBasic?
req := &http.Request{...}
```

---

## Proposed Solutions

### Option A: Declaration Files (Recommended)

Create `.dbdecl` (DBasic Declaration) files that define Go package interfaces:

```basic
' strings.dbdecl - Type definitions for Go strings package
DECLARE PACKAGE "strings"

DECLARE FUNCTION ToUpper(s AS STRING) AS STRING
DECLARE FUNCTION ToLower(s AS STRING) AS STRING
DECLARE FUNCTION Contains(s AS STRING, substr AS STRING) AS BOOLEAN
DECLARE FUNCTION Split(s AS STRING, sep AS STRING) AS STRING()
DECLARE FUNCTION Join(parts AS STRING(), sep AS STRING) AS STRING
DECLARE FUNCTION TrimSpace(s AS STRING) AS STRING
```

**Pros:**
- Explicit, user-controllable
- Works offline
- Can define subsets of packages
- Easy to understand

**Cons:**
- Manual effort to create declarations
- Can become out of sync with Go packages

### Option B: Go Introspection at Compile Time

Use `go doc -json` or reflection to automatically get package info:

```bash
go doc -json strings | dbasic-typegen > strings.dbdecl
```

**Pros:**
- Accurate, always up-to-date
- Automatic

**Cons:**
- Requires Go toolchain at compile time
- Complex implementation
- Slow for large packages

### Option C: Inline Type Annotations

Allow inline type hints for Go calls:

```basic
DIM result AS STRING = strings.ToUpper AS STRING(text AS STRING)
' Or
DIM result AS STRING = [STRING] strings.ToUpper(text)
```

**Pros:**
- No external files needed
- Explicit at point of use

**Cons:**
- Verbose
- Repetitive

### Option D: Trust-Based (Current)

Continue generating Go code without validation, trusting the Go compiler:

**Pros:**
- Simple
- Already working

**Cons:**
- Poor error messages
- No IDE support for Go packages

---

## Recommended Implementation Path

### Phase 1: Standard Library Declarations

Create built-in declaration files for commonly used Go packages:
- `fmt`
- `strings`
- `strconv`
- `time`
- `os`
- `io`
- `math`

### Phase 2: Declaration File Support

1. Add `DECLARE PACKAGE` and `DECLARE FUNCTION` syntax
2. Load `.dbdecl` files from a search path
3. Validate Go package calls against declarations

### Phase 3: Auto-Generation Tool

Create a tool to generate `.dbdecl` files from Go packages:

```bash
dbasic decl-gen net/http > net_http.dbdecl
```

---

## Syntax Proposals

### Package Declaration File (.dbdecl)

```basic
' Package declaration
DECLARE PACKAGE "fmt"

' Function with single return
DECLARE FUNCTION Sprintf(format AS STRING, args AS ANY...) AS STRING

' Function with error return (second return is error)
DECLARE FUNCTION Errorf(format AS STRING, args AS ANY...) AS ERROR

' Function with multiple returns
DECLARE FUNCTION Sscanf(str AS STRING, format AS STRING, args AS ANY...) AS (INTEGER, ERROR)

' Type declaration
DECLARE TYPE Stringer AS INTERFACE
    DECLARE FUNCTION String() AS STRING
END TYPE
```

### Using Declarations in DBasic

```basic
' Import pulls in declarations if available
IMPORT "fmt"
IMPORT "strings"

SUB Main()
    ' Now these calls are type-checked
    DIM upper AS STRING = strings.ToUpper("hello")
    fmt.Println(upper)

    ' Error handling with Go-style multiple returns
    DIM n AS INTEGER
    DIM err AS ERROR
    n, err = fmt.Sscanf("42", "%d")
    IF err <> NIL THEN
        PRINT "Error: "; err
    ENDIF
END SUB
```

---

## Implementation Notes

### Changes Required

1. **Lexer**: Add `DECLARE` keyword
2. **Parser**: Parse declaration syntax
3. **Analyzer**:
   - Load declaration files
   - Validate Go package calls against declarations
4. **Codegen**: No changes needed (already generates correct Go code)

### Declaration Search Path

Look for declarations in:
1. `./declarations/` (project local)
2. `~/.dbasic/declarations/` (user)
3. Built-in declarations (bundled with compiler)

---

## Future Considerations

### Struct Support

```basic
DECLARE TYPE http.Request AS STRUCT
    DECLARE FIELD Method AS STRING
    DECLARE FIELD URL AS POINTER TO url.URL
    DECLARE FIELD Header AS http.Header
END TYPE

' Usage
DIM req AS http.Request
req.Method = "GET"
```

### Interface Implementation

```basic
' Implement Go interface
TYPE MyWriter AS STRUCT
    DIM data AS STRING
END TYPE

' Implement io.Writer interface
FUNCTION (w AS POINTER TO MyWriter) Write(p AS BYTE()) AS (INTEGER, ERROR)
    w.data = w.data & STRING(p)
    RETURN Len(p), NIL
END FUNCTION
```
