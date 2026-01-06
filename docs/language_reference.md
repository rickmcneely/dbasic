# DBasic Language Reference

This document provides a complete reference for the DBasic programming language.

## Table of Contents

1. [Lexical Elements](#lexical-elements)
2. [Data Types](#data-types)
3. [Variables](#variables)
4. [Operators](#operators)
5. [Control Flow](#control-flow)
6. [Subroutines and Functions](#subroutines-and-functions)
7. [Arrays](#arrays)
8. [Pointers](#pointers)
9. [Channels and Concurrency](#channels-and-concurrency)
10. [JSON](#json)
11. [Go Package Integration](#go-package-integration)
12. [Built-in Functions](#built-in-functions)
13. [Keywords](#keywords)

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
| INTEGER | 32-bit integer | int32 | -2,147,483,648 to 2,147,483,647 |
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

## Arrays

### Array Literals

```basic
LET numbers = [1, 2, 3, 4, 5]
LET names = ["Alice", "Bob", "Charlie"]
```

### Array Access

```basic
PRINT numbers[0]    ' First element
numbers[1] = 100    ' Modify element
```

### Array Declaration (Fixed Size)

```basic
DIM arr(10) AS INTEGER
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

---

## Keywords

Reserved keywords in DBasic:

```
AND       AS        BOOLEAN   BYREF     BYVAL     CASE
CHAN      CHANNEL   CONST     DIM       DO        DOUBLE
ELSE      ELSEIF    END       ENDIF     EXIT      FALSE
FOR       FROM      FUNCTION  GOSUB     GOTO      IF
IMPORT    INPUT     INTEGER   JSON      LET       LONG
LOOP      MAKE_CHAN MOD       NEXT      NIL       NOT
OF        OR        POINTER   PRINT     RECEIVE   RETURN
SELECT    SEND      SINGLE    SPAWN     STEP      STRING
SUB       THEN      TO        TRUE      UNTIL     WEND
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
