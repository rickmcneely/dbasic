# DBasic Language Reference

This document provides a complete reference for the DBasic programming language.

## Table of Contents

1. [Lexical Elements](#lexical-elements)
2. [Data Types](#data-types)
3. [Variables](#variables)
4. [Operators](#operators)
5. [Control Flow](#control-flow)
6. [Subroutines and Functions](#subroutines-and-functions)
7. [Arrays and Slices](#arrays-and-slices)
8. [Structs and Struct Literals](#structs-and-struct-literals)
9. [Pointers](#pointers)
10. [Channels and Concurrency](#channels-and-concurrency)
11. [JSON](#json)
12. [File Inclusion](#file-inclusion)
13. [Go Package Integration](#go-package-integration)
14. [Built-in Functions](#built-in-functions)
15. [Keywords](#keywords)

---

## Lexical Elements

### Comments

Single-line comments start with an apostrophe (`'`):

```basic
' This is a comment
DIM x AS INTEGER  ' This is also a comment
```

### Identifiers

Identifiers are names for variables, functions, and labels. They:
- Must start with a letter or underscore
- Can contain letters, digits, and underscores
- Are case-insensitive (DIM, dim, Dim are equivalent)

```basic
myVariable
_private
counter123
```

### Labels

Labels are identifiers followed by a colon:

```basic
start:
    PRINT "Loop"
    GOTO start
```

### Literals

**Integer literals:**
```basic
42
-17
0
```

**Float literals:**
```basic
3.14
0.5
.25
2.5e-3
```

**String literals:**
```basic
"Hello, World!"
"Line 1\nLine 2"
"Tab\there"
```

Escape sequences:
- `\n` - newline
- `\t` - tab
- `\r` - carriage return
- `\"` - double quote
- `\\` - backslash

**Boolean literals:**
```basic
TRUE
FALSE
```

**Nil literal:**
```basic
NIL
```

---

## Data Types

### Numeric Types

| Type | Description | Go Equivalent | Range |
|------|-------------|---------------|-------|
| INTEGER | Platform integer | int | Platform dependent (typically 64-bit) |
| LONG | 64-bit integer | int64 | -9,223,372,036,854,775,808 to 9,223,372,036,854,775,807 |
| SINGLE | 32-bit float | float32 | ±1.18e-38 to ±3.4e38 |
| DOUBLE | 64-bit float | float64 | ±2.23e-308 to ±1.80e308 |

### Other Types

| Type | Description | Go Equivalent |
|------|-------------|---------------|
| STRING | Text string | string |
| BOOLEAN | True or false | bool |
| JSON | JSON object | map[string]interface{} |
| POINTER TO X | Pointer to type X | *X |
| CHAN OF X | Channel of type X | chan X |
| []X | Slice of type X | []X |

### User-Defined Types (Structs)

```basic
TYPE Person
    DIM Name AS STRING
    DIM Age AS INTEGER
END TYPE

TYPE Rectangle
    DIM Width AS DOUBLE
    DIM Height AS DOUBLE
END TYPE
```

---

## Variables

### Declaration with DIM

```basic
' Simple declaration
DIM name AS STRING

' Declaration with initialization
DIM count AS INTEGER = 0
DIM pi AS DOUBLE = 3.14159

' Array declaration
DIM numbers(10) AS INTEGER
```

### Declaration with LET (Type Inference)

```basic
LET x = 42           ' Inferred as INTEGER
LET name = "John"    ' Inferred as STRING
LET active = TRUE    ' Inferred as BOOLEAN
LET ratio = 3.14     ' Inferred as DOUBLE
```

### Constants

```basic
CONST MAX_SIZE AS INTEGER = 100
CONST PI AS DOUBLE = 3.14159
```

### Assignment

```basic
x = 10
name = "Alice"
numbers[0] = 42
config.port = 8080
```

### Multiple Assignment

```basic
DIM result AS INTEGER
DIM ok AS BOOLEAN
result, ok = Divide(10, 2)
```

---

## Operators

### Arithmetic Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition | `5 + 3` → 8 |
| `-` | Subtraction | `5 - 3` → 2 |
| `*` | Multiplication | `5 * 3` → 15 |
| `/` | Division | `5 / 2` → 2.5 |
| `\` | Integer division | `5 \ 2` → 2 |
| `MOD` | Modulo | `5 MOD 3` → 2 |
| `^` | Exponentiation | `2 ^ 3` → 8 |
| `-` | Negation (unary) | `-5` |

### Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equal | `x = 5` |
| `<>` | Not equal | `x <> 5` |
| `<` | Less than | `x < 5` |
| `>` | Greater than | `x > 5` |
| `<=` | Less than or equal | `x <= 5` |
| `>=` | Greater than or equal | `x >= 5` |

### Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `AND` | Logical AND | `a AND b` |
| `OR` | Logical OR | `a OR b` |
| `NOT` | Logical NOT | `NOT a` |
| `XOR` | Logical XOR | `a XOR b` |

### String Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `&` | Concatenation | `"Hello" & " " & "World"` |
| `+` | Concatenation (alternate) | `"Hello" + "World"` |

### Pointer Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `@` | Address-of | `@variable` |
| `^` | Dereference | `^pointer` |

---

## Control Flow

### IF Statement

```basic
IF condition THEN
    ' statements
ENDIF

IF condition THEN
    ' statements
ELSE
    ' statements
ENDIF

IF condition1 THEN
    ' statements
ELSEIF condition2 THEN
    ' statements
ELSE
    ' statements
ENDIF
```

### FOR Loop

```basic
' Basic FOR loop
FOR i = 1 TO 10
    PRINT i
NEXT

' FOR loop with STEP
FOR i = 10 TO 0 STEP -1
    PRINT i
NEXT

' Nested loops
FOR i = 1 TO 3
    FOR j = 1 TO 3
        PRINT i; ","; j
    NEXT
NEXT
```

### WHILE Loop

```basic
WHILE condition
    ' statements
WEND
```

### DO-LOOP

```basic
' Post-condition WHILE
DO
    ' statements
LOOP WHILE condition

' Post-condition UNTIL
DO
    ' statements
LOOP UNTIL condition

' Pre-condition WHILE
DO WHILE condition
    ' statements
LOOP

' Pre-condition UNTIL
DO UNTIL condition
    ' statements
LOOP
```

### SELECT CASE

```basic
SELECT CASE expression
CASE value1
    ' statements
CASE value2, value3
    ' statements (multiple values)
CASE ELSE
    ' default statements
END SELECT
```

### EXIT Statement

```basic
FOR i = 1 TO 100
    IF i = 50 THEN
        EXIT FOR
    ENDIF
NEXT

WHILE TRUE
    IF done THEN
        EXIT WHILE
    ENDIF
WEND

DO
    IF finished THEN
        EXIT DO
    ENDIF
LOOP
```

### GOTO Statement

```basic
start:
    PRINT "Hello"
    GOTO start
```

---

## Subroutines and Functions

### Subroutines (SUB)

Subroutines do not return values:

```basic
SUB PrintGreeting()
    PRINT "Hello, World!"
END SUB

SUB Greet(name AS STRING)
    PRINT "Hello, "; name; "!"
END SUB

SUB SwapValues(BYREF a AS INTEGER, BYREF b AS INTEGER)
    DIM temp AS INTEGER = a
    a = b
    b = temp
END SUB
```

### Functions (FUNCTION)

Functions return values:

```basic
FUNCTION Add(a AS INTEGER, b AS INTEGER) AS INTEGER
    RETURN a + b
END FUNCTION

FUNCTION IsEven(n AS INTEGER) AS BOOLEAN
    RETURN n MOD 2 = 0
END FUNCTION
```

### Multiple Return Values

```basic
FUNCTION Divide(a AS INTEGER, b AS INTEGER) AS (INTEGER, BOOLEAN)
    IF b = 0 THEN
        RETURN 0, FALSE
    ENDIF
    RETURN a / b, TRUE
END FUNCTION

' Usage
DIM result AS INTEGER
DIM ok AS BOOLEAN
result, ok = Divide(10, 2)
```

### Parameter Passing

By default, parameters are passed by value. Use `BYREF` for pass-by-reference:

```basic
SUB Increment(BYREF value AS INTEGER)
    value = value + 1
END SUB
```

---

## Arrays and Slices

### Slice Type Declaration

Slices are dynamic arrays that can grow:

```basic
DIM names AS []STRING
DIM numbers AS []INTEGER
DIM people AS []Person
```

### Array Literals

```basic
LET numbers = [1, 2, 3, 4, 5]
LET names = ["Alice", "Bob", "Charlie"]

' Assign to typed slice
DIM scores AS []INTEGER
scores = [100, 95, 87, 92]
```

### Array/Slice Access

```basic
PRINT numbers[0]    ' First element
numbers[1] = 100    ' Modify element
```

### Slice Operations

DBasic supports Go-style slice operations:

```basic
DIM data AS []INTEGER
data = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

' Get first 3 elements
DIM first3 AS []INTEGER
first3 = data[0:3]      ' [1, 2, 3]

' Get from index 5 to end
DIM rest AS []INTEGER
rest = data[5:]         ' [6, 7, 8, 9, 10]

' Get from start to index 3
DIM start AS []INTEGER
start = data[:3]        ' [1, 2, 3]

' Copy entire slice
DIM copy AS []INTEGER
copy = data[:]
```

### APPEND Function

Add elements to a slice:

```basic
DIM names AS []STRING
names = APPEND(names, "Alice")
names = APPEND(names, "Bob")
names = APPEND(names, "Charlie")

PRINT Len(names)  ' Prints 3
```

### Fixed-Size Array Declaration

```basic
DIM arr(10) AS INTEGER
```

---

## Structs and Struct Literals

### Defining Types

```basic
TYPE Person
    DIM Name AS STRING
    DIM Age AS INTEGER
END TYPE

TYPE MenuItem
    DIM Label AS STRING
    DIM Shortcut AS STRING
END TYPE
```

### Struct Literals

Create struct instances with field initialization:

```basic
DIM p AS Person
p = Person{Name: "John", Age: 30}

DIM item AS MenuItem
item = MenuItem{Label: "New", Shortcut: "Ctrl+N"}
```

### Accessing Fields

```basic
PRINT p.Name      ' "John"
PRINT p.Age       ' 30

p.Age = 31        ' Modify field
```

### Slices of Structs

```basic
DIM people AS []Person
people = APPEND(people, Person{Name: "Alice", Age: 25})
people = APPEND(people, Person{Name: "Bob", Age: 35})

PRINT people[0].Name  ' "Alice"
PRINT people[1].Age   ' 35
```

---

## Pointers

### Declaration

```basic
DIM ptr AS POINTER TO INTEGER
```

### Address-Of Operator (@)

```basic
DIM x AS INTEGER = 42
DIM ptr AS POINTER TO INTEGER = @x
```

### Dereference Operator (^)

```basic
PRINT ^ptr       ' Read through pointer
^ptr = 100       ' Write through pointer
```

### Pointer Parameters

```basic
SUB Increment(ptr AS POINTER TO INTEGER)
    ^ptr = ^ptr + 1
END SUB

DIM value AS INTEGER = 10
Increment(@value)
PRINT value  ' Prints 11
```

---

## Channels and Concurrency

### Channel Declaration

```basic
DIM ch AS CHAN OF INTEGER
DIM ch AS CHAN OF INTEGER = MAKE_CHAN(INTEGER, 10)  ' Buffered
```

### Send and Receive

```basic
' Send value to channel
SEND value TO channel

' Receive value from channel
DIM received AS INTEGER
RECEIVE received FROM channel
```

### Goroutines (SPAWN)

```basic
SUB Worker(id AS INTEGER)
    PRINT "Worker "; id; " started"
END SUB

' Start goroutine
SPAWN Worker(1)
```

### Example: Worker Pool

```basic
SUB Worker(id AS INTEGER, jobs AS CHAN OF INTEGER)
    DIM job AS INTEGER
    WHILE TRUE
        RECEIVE job FROM jobs
        IF job < 0 THEN
            EXIT WHILE
        ENDIF
        PRINT "Worker "; id; " processing "; job
    WEND
END SUB

SUB Main()
    DIM jobs AS CHAN OF INTEGER = MAKE_CHAN(INTEGER, 10)

    ' Start workers
    SPAWN Worker(1, jobs)
    SPAWN Worker(2, jobs)

    ' Send jobs
    FOR i = 1 TO 5
        SEND i TO jobs
    NEXT

    ' Signal termination
    SEND -1 TO jobs
    SEND -1 TO jobs
END SUB
```

---

## JSON

### JSON Literals

```basic
DIM config AS JSON = {name: "app", version: 1, enabled: TRUE}
```

### Dot Notation Access

```basic
PRINT config.name
PRINT config.version

config.enabled = FALSE
```

### Nested JSON

```basic
DIM data AS JSON = {user: {name: "John", age: 30}}
PRINT data.user.name
```

---

## File Inclusion

The `INCLUDE` directive allows you to include other DBasic source files, enabling code reuse and modular organization.

### Basic Usage

```basic
INCLUDE "mathutils.dbas"
INCLUDE "stringutils.dbas"

SUB Main()
    ' Use functions from included files
    PRINT Factorial(5)
    PRINT ReverseStr("Hello")
END SUB
```

### Path Resolution

- Paths are relative to the including file's directory
- Absolute paths are also supported
- File extension should be `.dbas`

```basic
' Relative to current file
INCLUDE "utils/helpers.dbas"

' Parent directory
INCLUDE "../common/shared.dbas"
```

### Circular Include Prevention

The preprocessor automatically detects and prevents circular includes:

```basic
' File: a.dbas
INCLUDE "b.dbas"  ' OK

' File: b.dbas
INCLUDE "a.dbas"  ' Error: circular include detected
```

### Best Practices

1. **Library files**: Create reusable utility functions in separate files
2. **No Main() in libraries**: Include files typically contain only functions and types, not the Main() subroutine
3. **One responsibility**: Each include file should focus on related functionality

### Example: Modular Project Structure

```
project/
├── main.dbas           ' Main program with SUB Main()
├── mathutils.dbas      ' Math utility functions
├── stringutils.dbas    ' String utility functions
└── types.dbas          ' Shared type definitions
```

**main.dbas:**
```basic
INCLUDE "types.dbas"
INCLUDE "mathutils.dbas"
INCLUDE "stringutils.dbas"

SUB Main()
    DIM result AS INTEGER = Factorial(5)
    PRINT result
END SUB
```

---

## Go Package Integration

### Importing Packages

```basic
IMPORT "fmt"
IMPORT "strings"
IMPORT "net/http" AS http
```

### Using Package Functions

```basic
IMPORT "strings"

SUB Main()
    DIM text AS STRING = "hello world"
    PRINT strings.ToUpper(text)
END SUB
```

**Note:** Go package function calls are generated correctly but are not type-checked by the DBasic analyzer.

---

## Built-in Functions

The runtime library provides these built-in functions:

### Slice/Collection Functions

| Function | Description |
|----------|-------------|
| `APPEND(slice, elem)` | Append element to slice, returns new slice |
| `Len(s)` | Length of string, slice, array, or map |
| `CAP(slice)` | Capacity of slice |
| `COPY(dst, src)` | Copy slice elements, returns count copied |
| `MAKE(type, len, cap)` | Create slice/map/channel |
| `DELETE(map, key)` | Delete key from map |
| `CLOSE(channel)` | Close a channel |

### String Functions

| Function | Description |
|----------|-------------|
| `Len(s)` | Length of string |
| `Left(s, n)` | Left n characters |
| `Right(s, n)` | Right n characters |
| `Mid(s, start, length)` | Substring |
| `Instr(s, substr)` | Find substring position |
| `UCase(s)` | Convert to uppercase |
| `LCase(s)` | Convert to lowercase |
| `Trim(s)` | Remove leading/trailing spaces |
| `LTrim(s)` | Remove leading spaces |
| `RTrim(s)` | Remove trailing spaces |
| `Str(n)` | Convert number to string |
| `Val(s)` | Convert string to number |
| `Chr(n)` | Character from ASCII code |
| `Asc(s)` | ASCII code from character |

### Math Functions

| Function | Description |
|----------|-------------|
| `Abs(n)` | Absolute value |
| `Sqr(n)` | Square root |
| `Sin(n)` | Sine |
| `Cos(n)` | Cosine |
| `Tan(n)` | Tangent |
| `Atn(n)` | Arctangent |
| `Log(n)` | Natural logarithm |
| `Exp(n)` | Exponential |
| `Int(n)` | Integer part |
| `Fix(n)` | Truncate toward zero |
| `Sgn(n)` | Sign (-1, 0, 1) |
| `Pow(x, y)` | Power |
| `Min(a, b)` | Minimum |
| `Max(a, b)` | Maximum |

### Random Functions

| Function | Description |
|----------|-------------|
| `Rnd()` | Random float 0-1 |
| `RndInt(max)` | Random integer 0 to max-1 |
| `RndRange(min, max)` | Random integer in range |
| `Randomize()` | Seed random generator |

### Date/Time Functions

| Function | Description |
|----------|-------------|
| `Timer()` | Seconds since midnight |
| `Now()` | Current timestamp |
| `Date()` | Current date string |
| `Year(t)` | Year from timestamp |
| `Month(t)` | Month from timestamp |
| `Day(t)` | Day from timestamp |
| `Hour(t)` | Hour from timestamp |
| `Minute(t)` | Minute from timestamp |
| `Second(t)` | Second from timestamp |
| `Sleep(ms)` | Pause for milliseconds |

### File I/O Functions

| Function | Description |
|----------|-------------|
| `FileExists(path)` | Check if file exists |
| `ReadFile(path)` | Read file contents |
| `WriteFile(path, content)` | Write to file |
| `AppendFile(path, content)` | Append to file |
| `DeleteFile(path)` | Delete file |
| `MkDir(path)` | Create directory |
| `RmDir(path)` | Remove directory |
| `ListDir(path)` | List directory contents |

### Formatted I/O Functions

| Function | Description |
|----------|-------------|
| `Printf(format, args...)` | Print formatted output (like C printf) |
| `Sprintf(format, args...)` | Return formatted string |

Uses Go's fmt.Printf/Sprintf format specifiers:
- `%s` - string
- `%d` - integer
- `%f` - float (use `%.2f` for 2 decimal places)
- `%v` - default format for any type
- `%t` - boolean
- `\n` - newline

```basic
Printf("Hello, %s! You are %d years old.\n", name, age)
DIM msg AS STRING = Sprintf("Score: %d / %d", correct, total)
```

### Error Handling Functions

| Function | Description |
|----------|-------------|
| `NewError(message)` | Create a new error with source location |
| `Errorf(format, args...)` | Create a formatted error with source location |
| `WrapError(err, message)` | Wrap an existing error with context and location |

DBasic errors automatically include source location (file:line and function name):

```basic
' Create a simple error - includes file:line (function)
err = NewError("something went wrong")
' Output: errors.dbas:5 (Main): something went wrong

' Create a formatted error
err = Errorf("failed to read %s at line %d", filename, lineNum)
' Output: errors.dbas:8 (Main): failed to read config.txt at line 42

' Check for errors (NIL means no error)
IF err <> NIL THEN
    Printf("Error: %v\n", err)
ENDIF
```

**Error Wrapping (Call Chains):**

Use `WrapError` to add context when propagating errors up the call stack:

```basic
FUNCTION LoadConfig(path AS STRING) AS ERROR
    DIM err AS ERROR
    err = ReadConfigFile(path)
    IF err <> NIL THEN
        RETURN WrapError(err, "failed to load config")
    ENDIF
    RETURN NIL
END FUNCTION

' When printed, shows the full error chain:
' errors.dbas:12 (LoadConfig): failed to load config
'   caused by: errors.dbas:20 (ReadConfigFile): file not found: config.txt
```

**Function Returning Errors (Go idiom):**

```basic
FUNCTION SafeDivide(a AS INTEGER, b AS INTEGER) AS (INTEGER, ERROR)
    IF b = 0 THEN
        RETURN 0, NewError("division by zero")
    ENDIF
    RETURN a / b, NIL
END FUNCTION

SUB Main()
    DIM result AS INTEGER
    DIM err AS ERROR

    result, err = SafeDivide(10, 0)
    IF err <> NIL THEN
        Printf("Error: %v\n", err)
        ' Output: errors.dbas:4 (SafeDivide): division by zero
    ELSE
        Printf("Result: %d\n", result)
    ENDIF
END SUB
```

---

## Keywords

Reserved keywords in DBasic:

```
AND       APPEND    AS        BOOLEAN   BSTRING   BYREF
BYVAL     BYTES     CAP       CASE      CHAN      CHANNEL
CLOSE     CONST     COPY      DELETE    DIM       DO
DOUBLE    ELSE      ELSEIF    END       ENDIF     EXIT
FALSE     FOR       FROM      FUNCTION  GOSUB     GOTO
IF        IMPORT    INCLUDE   INPUT     INTEGER   JSON
LEN       LET       LONG      LOOP      MAKE      MAKE_CHAN
MOD       NEW       NEXT      NIL       NOT       OF
OR        POINTER   PRINT     RECEIVE   RETURN    SELECT
SEND      SINGLE    SPAWN     STEP      STRING    SUB
THEN      TO        TRUE      TYPE      UNTIL     WEND
WHILE     XOR
```

---

## Error Messages

DBasic provides detailed error messages with source context:

```
parse error at line 5, column 8: expected THEN, got NEWLINE instead
  5 | IF x > 5
             ^
  hint: IF statements require THEN after the condition
```

Error messages include:
- Error phase (parse, semantic)
- Line and column numbers
- The source line
- A caret pointing to the error location
- Helpful hints for common mistakes
