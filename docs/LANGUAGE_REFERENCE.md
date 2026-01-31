# DBasic Language Reference

**Version 0.2.0**

DBasic is a modern BASIC dialect that compiles to Go. It combines familiar BASIC syntax with Go's powerful type system, concurrency features, and package ecosystem.

---

## Table of Contents

1. [Program Structure](#program-structure)
2. [Comments](#comments)
3. [Data Types](#data-types)
4. [Variables and Constants](#variables-and-constants)
5. [Operators](#operators)
6. [Control Flow](#control-flow)
7. [Functions and Subroutines](#functions-and-subroutines)
8. [User-Defined Types](#user-defined-types)
9. [Arrays and Slices](#arrays-and-slices)
10. [Pointers](#pointers)
11. [Error Handling](#error-handling)
12. [Concurrency](#concurrency)
13. [Go Package Integration](#go-package-integration)
14. [Built-in Functions](#built-in-functions)
15. [HTTP Functions](#http-functions)
16. [Keywords Reference](#keywords-reference)

---

## Program Structure

A DBasic program consists of:

1. Import statements (must come first)
2. Type definitions
3. Global variable/constant declarations
4. Function and subroutine definitions
5. A `Main()` subroutine as the entry point

```basic
' Import statements
IMPORT "fmt" AS fmt
IMPORT "os" AS os

' Type definitions
TYPE Person
    DIM Name AS STRING
    DIM Age AS INTEGER
END TYPE

' Global constants
CONST MAX_SIZE AS INTEGER = 100

' Entry point
SUB Main()
    PRINT "Hello, World!"
END SUB
```

---

## Comments

DBasic uses single-quote for line comments:

```basic
' This is a comment
DIM x AS INTEGER  ' Inline comment
```

---

## Data Types

### Primitive Types

| Type | Description | Go Equivalent |
|------|-------------|---------------|
| `INTEGER` | 32-bit signed integer | `int` |
| `LONG` | 64-bit signed integer | `int64` |
| `SINGLE` | 32-bit floating point | `float32` |
| `DOUBLE` | 64-bit floating point | `float64` |
| `STRING` | Unicode text string | `string` |
| `BOOLEAN` | Boolean value | `bool` |

### Complex Types

| Type | Description | Go Equivalent |
|------|-------------|---------------|
| `BYTES` | Byte array | `[]byte` |
| `JSON` | JSON object | `map[string]interface{}` |
| `ANY` | Any type | `interface{}` |
| `ERROR` | Error type | `error` |
| `POINTER TO T` | Pointer to type T | `*T` |
| `CHAN OF T` | Channel of type T | `chan T` |
| `[]T` | Slice/dynamic array | `[]T` |

### Type Literals

```basic
' Integer
DIM i AS INTEGER = 42

' Float
DIM f AS DOUBLE = 3.14159

' String
DIM s AS STRING = "Hello"

' Byte string
DIM b AS BYTES = B"raw bytes"

' Boolean
DIM flag AS BOOLEAN = TRUE

' Nil/null
DIM ptr AS POINTER TO INTEGER = NIL
```

---

## Variables and Constants

### Variable Declaration

```basic
' Basic declaration
DIM name AS STRING

' Declaration with initialization
DIM count AS INTEGER = 0

' Type inference with LET
LET message = "Hello"  ' Inferred as STRING

' Multiple variables (separate statements)
DIM x AS INTEGER
DIM y AS INTEGER
DIM z AS INTEGER
```

### Constants

```basic
CONST PI AS DOUBLE = 3.14159
CONST MAX_USERS AS INTEGER = 100
CONST APP_NAME AS STRING = "MyApp"
```

### Scope

- Variables declared at module level are global
- Variables declared in functions/subs are local
- FOR loop variables are scoped to the loop

---

## Operators

### Arithmetic Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition | `a + b` |
| `-` | Subtraction | `a - b` |
| `*` | Multiplication | `a * b` |
| `/` | Division (float) | `a / b` |
| `\` | Integer division | `a \ b` |
| `^` | Exponentiation | `a ^ b` |
| `MOD` | Modulo | `a MOD b` |

### Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equal | `a = b` |
| `<>` | Not equal | `a <> b` |
| `<` | Less than | `a < b` |
| `>` | Greater than | `a > b` |
| `<=` | Less or equal | `a <= b` |
| `>=` | Greater or equal | `a >= b` |

### Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `AND` | Logical AND | `a AND b` |
| `OR` | Logical OR | `a OR b` |
| `XOR` | Logical XOR | `a XOR b` |
| `NOT` | Logical NOT | `NOT a` |

### String Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `&` | Concatenation | `"Hello" & " World"` |

### Pointer Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `@` | Address-of | `@variable` |
| `^` | Dereference | `^pointer` |

### Operator Precedence (Highest to Lowest)

1. Function calls, indexing `()`, `[]`
2. Member access `.`
3. Unary operators `-`, `NOT`, `@`, `^`
4. Exponentiation `^`
5. Multiplication `*`, `/`, `\`, `MOD`
6. Addition `+`, `-`, `&`
7. Comparison `<`, `>`, `<=`, `>=`
8. Equality `=`, `<>`
9. `AND`
10. `OR`, `XOR`

---

## Control Flow

### IF Statement

```basic
' Single line
IF condition THEN statement

' Block form
IF condition THEN
    ' statements
ENDIF

' With ELSE
IF condition THEN
    ' true branch
ELSE
    ' false branch
ENDIF

' With ELSEIF
IF condition1 THEN
    ' branch 1
ELSEIF condition2 THEN
    ' branch 2
ELSEIF condition3 THEN
    ' branch 3
ELSE
    ' default branch
ENDIF
```

### FOR Loop

```basic
' Basic FOR loop
FOR i = 1 TO 10
    PRINT i
NEXT

' With STEP
FOR i = 10 TO 0 STEP -1
    PRINT i
NEXT

' Iterating arrays
FOR i = 0 TO Len(array) - 1
    PRINT array[i]
NEXT

' Exit early
FOR i = 1 TO 100
    IF condition THEN
        EXIT FOR
    ENDIF
NEXT
```

### WHILE Loop

```basic
WHILE condition
    ' statements
WEND

' Exit early
WHILE TRUE
    IF done THEN
        EXIT WHILE
    ENDIF
WEND
```

### DO Loop

```basic
' Infinite loop
DO
    ' statements
    IF done THEN EXIT DO
LOOP

' Pre-test WHILE
DO WHILE condition
    ' statements
LOOP

' Pre-test UNTIL
DO UNTIL condition
    ' statements
LOOP

' Post-test WHILE
DO
    ' statements
LOOP WHILE condition

' Post-test UNTIL
DO
    ' statements
LOOP UNTIL condition
```

### SELECT CASE

```basic
SELECT CASE expression
    CASE value1
        ' statements
    CASE value2, value3
        ' statements for multiple values
    CASE ELSE
        ' default case
END SELECT
```

### GOTO (Use Sparingly)

```basic
GOTO label

label:
    ' statements
```

---

## Functions and Subroutines

### Subroutines (No Return Value)

```basic
SUB SayHello()
    PRINT "Hello!"
END SUB

SUB Greet(name AS STRING)
    PRINT "Hello, " & name & "!"
END SUB

' Calling
SayHello()
Greet("World")
```

### Functions (With Return Value)

```basic
FUNCTION Add(a AS INTEGER, b AS INTEGER) AS INTEGER
    RETURN a + b
END FUNCTION

FUNCTION GetMessage() AS STRING
    RETURN "Hello"
END FUNCTION

' Calling
DIM sum AS INTEGER = Add(5, 3)
DIM msg AS STRING = GetMessage()
```

### Multiple Return Values

```basic
FUNCTION Divide(a AS INTEGER, b AS INTEGER) AS (INTEGER, INTEGER)
    DIM quotient AS INTEGER = a \ b
    DIM remainder AS INTEGER = a MOD b
    RETURN quotient, remainder
END FUNCTION

' Calling
DIM q AS INTEGER
DIM r AS INTEGER
q, r = Divide(17, 5)
```

### Parameter Passing

```basic
' By value (default) - changes don't affect original
SUB ModifyValue(x AS INTEGER)
    x = x + 1  ' Only modifies local copy
END SUB

' By reference - changes affect original
SUB ModifyRef(x AS INTEGER BYREF)
    x = x + 1  ' Modifies original variable
END SUB

DIM num AS INTEGER = 10
ModifyValue(num)  ' num is still 10
ModifyRef(num)    ' num is now 11
```

### Early Exit

```basic
SUB Process()
    IF error THEN
        EXIT SUB
    ENDIF
    ' Continue processing
END SUB

FUNCTION Calculate() AS INTEGER
    IF invalid THEN
        EXIT FUNCTION  ' Returns zero value
    ENDIF
    RETURN result
END FUNCTION
```

---

## User-Defined Types

### Basic Type Definition

```basic
TYPE Person
    DIM Name AS STRING
    DIM Age AS INTEGER
    DIM Email AS STRING
END TYPE

' Creating instance
DIM p AS Person
p.Name = "John"
p.Age = 30

' Struct literal
DIM p2 AS Person = Person{Name: "Jane", Age: 25}
```

### Methods

```basic
TYPE Rectangle
    DIM Width AS DOUBLE
    DIM Height AS DOUBLE
END TYPE

' Method definition
FUNCTION (r AS Rectangle) Area() AS DOUBLE
    RETURN r.Width * r.Height
END FUNCTION

SUB (r AS Rectangle) Print()
    PRINT "Width: " & Str(r.Width) & ", Height: " & Str(r.Height)
END SUB

' Using methods
DIM rect AS Rectangle
rect.Width = 10
rect.Height = 5
PRINT rect.Area()  ' Outputs: 50
rect.Print()
```

### Implementing Go Interfaces

```basic
IMPORT "github.com/charmbracelet/bubbletea" AS tea

TYPE MyModel IMPLEMENTS tea.Model
    DIM Count AS INTEGER
END TYPE

FUNCTION (m AS MyModel) Init() AS tea.Cmd
    RETURN NIL
END FUNCTION

FUNCTION (m AS MyModel) Update(msg AS tea.Msg) AS (tea.Model, tea.Cmd)
    RETURN m, NIL
END FUNCTION

FUNCTION (m AS MyModel) View() AS STRING
    RETURN "Count: " & Str(m.Count)
END FUNCTION
```

### Embedding External Types

```basic
TYPE MyWidget
    EMBED externalpackage.Widget
    DIM CustomField AS STRING
END TYPE
```

---

## Arrays and Slices

### Dynamic Arrays (Slices)

```basic
' Declaration
DIM numbers AS []INTEGER

' Literal initialization
DIM primes AS []INTEGER = []INTEGER{2, 3, 5, 7, 11}

' Append elements
numbers = APPEND(numbers, 1)
numbers = APPEND(numbers, 2, 3, 4)

' Access elements (0-indexed)
DIM first AS INTEGER = numbers[0]
numbers[1] = 100

' Length
DIM length AS INTEGER = Len(numbers)

' Iteration
FOR i = 0 TO Len(numbers) - 1
    PRINT numbers[i]
NEXT
```

### Array Slicing

```basic
DIM arr AS []INTEGER = []INTEGER{1, 2, 3, 4, 5}

DIM sub1 AS []INTEGER = arr[1:3]   ' Elements 1, 2 (indices 1 to 2)
DIM sub2 AS []INTEGER = arr[:3]    ' Elements 0, 1, 2
DIM sub3 AS []INTEGER = arr[2:]    ' Elements from index 2 to end
DIM copy AS []INTEGER = arr[:]     ' Full copy
```

### Fixed-Size Arrays

```basic
' Legacy syntax
DIM fixedArray(10) AS INTEGER  ' 10-element array
```

---

## Pointers

### Creating Pointers

```basic
DIM x AS INTEGER = 42
DIM ptr AS POINTER TO INTEGER = @x  ' Get address

' Or use New() for heap allocation
DIM heapPtr AS POINTER TO INTEGER = New(INTEGER)
```

### Dereferencing

```basic
DIM x AS INTEGER = 42
DIM ptr AS POINTER TO INTEGER = @x

' Read value
DIM value AS INTEGER = ^ptr  ' value = 42

' Write value
^ptr = 100  ' x is now 100

' Or use parentheses for clarity
(^ptr) = 100
```

### Pointer to Struct

```basic
TYPE Data
    DIM Value AS INTEGER
END TYPE

DIM d AS Data
DIM ptr AS POINTER TO Data = @d

' Access fields through pointer
(^ptr).Value = 42
```

---

## Error Handling

### The Error Type

```basic
DIM err AS ERROR

' Check for error
IF err <> NIL THEN
    PRINT "Error occurred"
ENDIF
```

### Creating Errors

```basic
' Simple error
DIM err AS ERROR = NewError("Something went wrong")

' Formatted error
DIM err AS ERROR = Errorf("Failed to open file: %s", filename)
```

### Functions Returning Errors

```basic
FUNCTION ReadData(path AS STRING) AS (STRING, ERROR)
    IF NOT FileExists(path) THEN
        RETURN "", NewError("File not found")
    ENDIF
    DIM content AS STRING = ReadFile(path)
    RETURN content, NIL
END FUNCTION

' Calling
DIM data AS STRING
DIM err AS ERROR
data, err = ReadData("config.txt")
IF err <> NIL THEN
    PRINT "Error: " & fmt.Sprintf("%v", err)
ENDIF
```

---

## Concurrency

### Goroutines with SPAWN

```basic
SUB Worker(id AS INTEGER)
    PRINT "Worker " & Str(id) & " running"
END SUB

SUB Main()
    ' Launch goroutines
    SPAWN Worker(1)
    SPAWN Worker(2)
    SPAWN Worker(3)

    ' Wait for goroutines (simple delay)
    Sleep(1000)
END SUB
```

### Channels

```basic
' Create channel
DIM ch AS CHAN OF INTEGER = MAKE_CHAN(CHAN OF INTEGER, 10)

' Send to channel
SEND 42 TO ch

' Receive from channel
DIM value AS INTEGER
RECEIVE value FROM ch

' Close channel
Close(ch)
```

### Channel Example

```basic
IMPORT "fmt" AS fmt

SUB Producer(ch AS CHAN OF INTEGER)
    FOR i = 1 TO 5
        SEND i TO ch
    NEXT
    Close(ch)
END SUB

SUB Main()
    DIM ch AS CHAN OF INTEGER = MAKE_CHAN(CHAN OF INTEGER, 0)

    SPAWN Producer(ch)

    ' Receive until channel closed
    DIM value AS INTEGER
    DIM ok AS BOOLEAN = TRUE
    WHILE ok
        value, ok = <-ch
        IF ok THEN
            fmt.Printf("Received: %d\n", value)
        ENDIF
    WEND
END SUB
```

---

## Go Package Integration

### Importing Packages

```basic
' Standard library
IMPORT "fmt" AS fmt
IMPORT "os" AS os
IMPORT "strings" AS strings

' Third-party packages
IMPORT "github.com/charmbracelet/lipgloss" AS lipgloss

' Blank import (for side effects)
IMPORT _ "image/png"
```

### Using Go Types and Functions

```basic
IMPORT "net/http" AS http
IMPORT "io" AS io

' Using Go types
DIM client AS http.Client
DIM req AS POINTER TO http.Request
DIM resp AS POINTER TO http.Response

' Calling Go functions
req, _ = http.NewRequest("GET", "https://example.com", NIL)
resp, _ = client.Do(req)

' Type assertions
DIM keyMsg AS tea.KeyMsg
DIM ok AS BOOLEAN
keyMsg, ok = msg.(tea.KeyMsg)
IF ok THEN
    ' Handle key message
ENDIF
```

---

## Built-in Functions

### String Functions

| Function | Description |
|----------|-------------|
| `Len(s)` | Length of string |
| `Left(s, n)` | First n characters |
| `Right(s, n)` | Last n characters |
| `Mid(s, start, len)` | Substring (1-indexed) |
| `Instr(s, find)` | Find substring (1-indexed, 0 if not found) |
| `UCase(s)` | Convert to uppercase |
| `LCase(s)` | Convert to lowercase |
| `Trim(s)` | Remove leading/trailing whitespace |
| `LTrim(s)` | Remove leading whitespace |
| `RTrim(s)` | Remove trailing whitespace |
| `Replace(s, old, new)` | Replace all occurrences |
| `Space(n)` | Create string of n spaces |
| `Chr(code)` | Character from ASCII code |
| `Asc(s)` | ASCII code of first character |
| `Str(value)` | Convert to string |

### Math Functions

| Function | Description |
|----------|-------------|
| `Abs(x)` | Absolute value |
| `Sqr(x)` | Square root |
| `Sin(x)` | Sine (radians) |
| `Cos(x)` | Cosine (radians) |
| `Tan(x)` | Tangent (radians) |
| `Atn(x)` | Arctangent |
| `Atn2(y, x)` | Arctangent of y/x |
| `Log(x)` | Natural logarithm |
| `Log10(x)` | Base-10 logarithm |
| `Exp(x)` | e^x |
| `Pow(x, y)` | x^y |
| `Sgn(x)` | Sign (-1, 0, or 1) |
| `Fix(x)` | Truncate toward zero |
| `Floor(x)` | Floor (round down) |
| `Ceil(x)` | Ceiling (round up) |
| `Round(x)` | Round to nearest |
| `Min(a, b)` | Minimum |
| `Max(a, b)` | Maximum |
| `Clamp(x, min, max)` | Clamp to range |
| `PI()` | Value of pi |

### Random Functions

| Function | Description |
|----------|-------------|
| `Rnd()` | Random float 0.0 to 1.0 |
| `RndInt(max)` | Random integer 0 to max-1 |
| `RndRange(min, max)` | Random integer in range |
| `Randomize(seed)` | Seed generator |

### Type Conversion

| Function | Description |
|----------|-------------|
| `Int(x)` | Convert to INTEGER |
| `Lng(x)` | Convert to LONG |
| `Sng(x)` | Convert to SINGLE |
| `Dbl(x)` | Convert to DOUBLE |
| `Bool(x)` | Convert to BOOLEAN |
| `Val(s)` | Parse string to number |

### Byte Functions

| Function | Description |
|----------|-------------|
| `Encode(s)` | String to BYTES |
| `Decode(b)` | BYTES to string |
| `MakeBytes(n)` | Create n-byte array |
| `LenBytes(b)` | Length of BYTES |

### Date/Time Functions

| Function | Description |
|----------|-------------|
| `Timer()` | Seconds since midnight |
| `Now()` | Unix timestamp |
| `Date()` | Current date (YYYY-MM-DD) |
| `Year()` | Current year |
| `Month()` | Current month (1-12) |
| `Day()` | Current day of month |
| `Hour()` | Current hour (0-23) |
| `Minute()` | Current minute |
| `Second()` | Current second |
| `Sleep(ms)` | Pause execution |

### File Functions

| Function | Description |
|----------|-------------|
| `FileExists(path)` | Check if file exists |
| `ReadFile(path)` | Read entire file |
| `WriteFile(path, content)` | Write file |
| `AppendFile(path, content)` | Append to file |
| `DeleteFile(path)` | Delete file |
| `MkDir(path)` | Create directory |
| `RmDir(path)` | Remove directory |

### JSON Functions

| Function | Description |
|----------|-------------|
| `JSONParse(s)` | Parse JSON string |
| `JSONStringify(j)` | Convert to JSON string |
| `JSONPretty(j)` | Pretty-print JSON |
| `JSONGet(j, path)` | Get value by path |
| `JSONSet(j, path, val)` | Set value by path |
| `StructToJSON(v)` | Struct to JSON |
| `JSONToStruct(j, v)` | JSON to struct |

### Array/Slice Functions

| Function | Description |
|----------|-------------|
| `Len(arr)` | Length of array |
| `Cap(arr)` | Capacity of array |
| `APPEND(arr, items...)` | Append to slice |
| `Copy(dst, src)` | Copy between slices |
| `Make(type, len, cap)` | Create slice |

### I/O Functions

| Function | Description |
|----------|-------------|
| `PRINT expr, ...` | Print to console |
| `INPUT var` | Read line from console |
| `Printf(fmt, args...)` | Formatted print |
| `Sprintf(fmt, args...)` | Formatted string |

---

## HTTP Functions

DBasic provides built-in HTTP helper functions for making web requests. These are automatically generated when using the Curl widget in VDBTerm projects, or can be manually implemented.

### CurlFetch

Performs an HTTP request and returns the response body, status code, and any error.

```basic
FUNCTION CurlFetch(url AS STRING, method AS STRING, body AS STRING) AS (STRING, INTEGER, STRING)
```

**Parameters:**
- `url` - The URL to request
- `method` - HTTP method (GET, POST, PUT, DELETE, etc.)
- `body` - Request body (use "" for empty)

**Returns:**
- Response body as STRING
- HTTP status code as INTEGER
- Error message as STRING (empty if successful)

**Example:**
```basic
DIM response AS STRING
DIM statusCode AS INTEGER
DIM errStr AS STRING

response, statusCode, errStr = CurlFetch("https://api.example.com/data", "POST", "{\"key\":\"value\"}")

IF errStr <> "" THEN
    PRINT "Error: " & errStr
ELSEIF statusCode = 200 THEN
    PRINT "Success: " & response
ENDIF
```

### CurlGet

Convenience function for GET requests.

```basic
FUNCTION CurlGet(url AS STRING) AS (STRING, INTEGER, STRING)
```

**Example:**
```basic
DIM data AS STRING
DIM code AS INTEGER
DIM err AS STRING

data, code, err = CurlGet("https://httpbin.org/get")
IF err = "" THEN
    PRINT "Response: " & data
ENDIF
```

### CurlPost

Convenience function for POST requests.

```basic
FUNCTION CurlPost(url AS STRING, body AS STRING) AS (STRING, INTEGER, STRING)
```

**Example:**
```basic
DIM result AS STRING
DIM status AS INTEGER
DIM errMsg AS STRING

result, status, errMsg = CurlPost("https://httpbin.org/post", "{\"name\":\"test\"}")
```

### Required Imports for HTTP

When implementing HTTP functionality manually, include these imports:

```basic
IMPORT "net/http" AS http
IMPORT "io" AS io
IMPORT "bytes" AS bytebuf
IMPORT "fmt" AS fmt
```

**Note:** The `bytes` package must be aliased (e.g., `bytebuf`) since `bytes` is a reserved word in DBasic.

---

## Keywords Reference

### Declaration Keywords
`DIM`, `LET`, `CONST`, `TYPE`, `AS`, `EMBED`, `IMPLEMENTS`

### Data Type Keywords
`INTEGER`, `LONG`, `SINGLE`, `DOUBLE`, `STRING`, `BOOLEAN`, `BYTES`, `BSTRING`, `JSON`, `POINTER`, `CHAN`, `ANY`, `ERROR`

### Control Flow Keywords
`IF`, `THEN`, `ELSE`, `ELSEIF`, `ENDIF`, `FOR`, `TO`, `STEP`, `NEXT`, `WHILE`, `WEND`, `DO`, `LOOP`, `UNTIL`, `EXIT`, `SELECT`, `CASE`, `END`, `GOTO`, `RETURN`

### Function Keywords
`SUB`, `FUNCTION`, `BYREF`, `BYVAL`

### Logical Keywords
`AND`, `OR`, `NOT`, `XOR`, `MOD`

### Value Keywords
`TRUE`, `FALSE`, `NIL`

### I/O Keywords
`PRINT`, `INPUT`

### Go Integration Keywords
`IMPORT`, `SPAWN`, `SEND`, `RECEIVE`, `FROM`, `MAKE_CHAN`, `OF`

---

## Compiler Usage

```bash
# Compile to executable
dbasic build program.dbas -o program

# Compile and run
dbasic run program.dbas

# Output generated Go code
dbasic emit program.dbas

# Check for errors only
dbasic check program.dbas

# Show version
dbasic version

# Show help
dbasic help
```

---

## Example Program

```basic
' Complete example demonstrating DBasic features

IMPORT "fmt" AS fmt
IMPORT "strings" AS strings

' Type definition
TYPE Task
    DIM ID AS INTEGER
    DIM Title AS STRING
    DIM Done AS BOOLEAN
END TYPE

' Method on Task
FUNCTION (t AS Task) String() AS STRING
    DIM status AS STRING
    IF t.Done THEN
        status = "[X]"
    ELSE
        status = "[ ]"
    ENDIF
    RETURN fmt.Sprintf("%s %d: %s", status, t.ID, t.Title)
END FUNCTION

' Global task list
DIM tasks AS []Task

' Add a task
SUB AddTask(title AS STRING)
    DIM t AS Task
    t.ID = Len(tasks) + 1
    t.Title = title
    t.Done = FALSE
    tasks = APPEND(tasks, t)
END SUB

' Mark task as done
SUB CompleteTask(id AS INTEGER)
    FOR i = 0 TO Len(tasks) - 1
        IF tasks[i].ID = id THEN
            tasks[i].Done = TRUE
            RETURN
        ENDIF
    NEXT
END SUB

' List all tasks
SUB ListTasks()
    IF Len(tasks) = 0 THEN
        PRINT "No tasks."
        RETURN
    ENDIF

    PRINT "Tasks:"
    PRINT strings.Repeat("-", 40)
    FOR i = 0 TO Len(tasks) - 1
        PRINT tasks[i].String()
    NEXT
END SUB

' Entry point
SUB Main()
    PRINT "Task Manager Demo"
    PRINT ""

    AddTask("Learn DBasic")
    AddTask("Build an app")
    AddTask("Deploy to production")

    ListTasks()

    PRINT ""
    PRINT "Completing task 1..."
    CompleteTask(1)

    PRINT ""
    ListTasks()
END SUB
```

---

*DBasic - A Modern BASIC for the Go Era*
