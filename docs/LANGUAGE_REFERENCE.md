# DBasic Language Reference

**Version 0.2.0**

DBasic is a modern BASIC dialect that compiles to Go. It combines familiar BASIC syntax with Go's powerful type system, concurrency features, and package ecosystem.

---

## Table of Contents

1. [Program Structure](#program-structure)
2. [Comments](#comments)
3. [Data Types](#data-types)
4. [Variables and Constants](#variables-and-constants) — DIM, REDIM, STATIC, SHARED, OPTION, reserved-word relaxation
5. [Operators](#operators)
6. [Control Flow](#control-flow) — IF/SELECT CASE/FOR/WHILE/DO, EXIT/CONTINUE, single-line IF, WITH
7. [Functions and Subroutines](#functions-and-subroutines) — SUB, FUNCTION, DECLARE, CALL, lambdas
8. [User-Defined Types](#user-defined-types)
9. [Arrays and Slices](#arrays-and-slices)
10. [Maps](#maps)
11. [Deferred Calls](#deferred-calls)
12. [Pointers](#pointers)
13. [Error Handling](#error-handling) — explicit checks; `ONERR GOTO`/`GOFUNC` sugar (no try/catch)
14. [Concurrency](#concurrency)
15. [Go Package Integration](#go-package-integration) — IMPORT, generic instantiation
16. [Built-in Functions](#built-in-functions)
17. [HTTP Functions](#http-functions)
18. [Shell Functions](#shell-functions)
19. [Keywords Reference](#keywords-reference)
20. [Compiler Usage](#compiler-usage) — build, run, fmt, doc, test, cross-compile, LSP

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

DBasic accepts two comment styles, both terminated by end of line:

```basic
' This is an apostrophe comment
DIM x AS INTEGER  ' Inline comment

REM This is a REM comment (classic BASIC)
REM REM is recognized when followed by whitespace or end of line
```

`REM` only acts as a comment when it appears at statement-start position. Identifiers like `remainder` are unaffected.

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
| `RECEIVE CHAN OF T` | Receive-only channel | `<-chan T` |
| `SEND CHAN OF T` | Send-only channel | `chan<- T` |
| `[]T` | Slice/dynamic array | `[]T` |
| `MAP OF K TO V` | Lookup table from key K to value V | `map[K]V` |

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

### STATIC Variables

`STATIC` declares a variable inside a SUB or FUNCTION whose value persists across calls. The compiler hoists it to a uniquified package-level variable.

```basic
SUB Counter()
    STATIC count AS INTEGER = 0
    count = count + 1
    PRINT "called "; count; " time(s)"
END SUB
```

### SHARED Declarations

`SHARED name1, name2, ...` inside a sub declares that the listed module-level names are visible. DBasic already resolves identifiers up the scope chain, so `SHARED` is parsed and accepted without changing semantics — useful for QB-style code.

```basic
DIM gCounter AS INTEGER = 0

SUB Bump()
    SHARED gCounter AS INTEGER
    gCounter = gCounter + 1
END SUB
```

### Reserved Words as Identifiers

`STEP`, `BYTES`, `STRING`, `TYPE`, and `CONTINUE` may be used as variable, parameter, field, or sub/function names. They retain their keyword meaning in syntactic positions (e.g. `FOR i = 1 TO 10 STEP 2`, or `CONTINUE FOR` at the start of a statement).

```basic
DIM TYPE AS INTEGER = 99       ' a variable named TYPE
DIM BYTES AS INTEGER = 16
PRINT TYPE; BYTES
```

### REDIM

`REDIM x(N) AS T` resizes a slice. With `PRESERVE`, existing elements are copied into the new slice (truncated or zero-padded as needed).

```basic
DIM xs AS []INTEGER
REDIM xs(5) AS INTEGER         ' fresh slice of length 5
REDIM PRESERVE xs(8) AS INTEGER ' grow to 8, keep first 5 values
```

### Scope

- Variables declared at module level are global
- Variables declared in functions/subs are local
- FOR loop variables are scoped to the loop
- `STATIC` variables are local in name but persistent across calls

### OPTION Pragmas

| Pragma | Effect |
|--------|--------|
| `OPTION EXPLICIT` | Require DIM before use. DBasic always enforces this; the pragma is accepted as a no-op for QB compatibility. |
| `OPTION BASE 0` | Default; arrays/slices are 0-indexed. |
| `OPTION BASE 1` | Rewrites every `xs[i]` access to `xs[i-1]` so user code can be 1-indexed. Don't combine with map-style indexing under OPTION BASE 1. |

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

`AND`, `OR`, `XOR`, and `NOT` accept booleans (canonical) or numeric
operands. With numeric operands, `0` is false and any non-zero value is
true:

```basic
DIM x AS INTEGER = 5
DIM y AS INTEGER = 0
IF x AND y THEN ...     ' false (y is 0)
IF x OR y THEN ...      ' true  (x is non-zero)
IF NOT y THEN ...       ' true  (y is 0)
```

Function calls and struct field access are not auto-wrapped — Go's
strict bool semantics still apply there. Use a comparison
(`foo() = TRUE` or `foo() <> 0`) when you need to be explicit.

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
' Single line (no ELSE)
IF condition THEN statement

' Single line with ELSE
IF condition THEN x = 1 ELSE x = 2

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

`AND`, `OR`, `XOR`, and `NOT` accept boolean OR numeric operands; `0` is false, non-zero is true. Use a comparison (`x = TRUE`, `x <> 0`) for explicit checks against function-call results.

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

' Skip to the next iteration
FOR i = 1 TO 100
    IF i MOD 2 = 0 THEN
        CONTINUE FOR
    ENDIF
    PRINT i          ' odd numbers only
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

' Skip to the next iteration. Take care that whatever moves the loop
' along happens BEFORE the CONTINUE, or the condition never changes.
WHILE i < 10
    i = i + 1
    IF skip(i) THEN
        CONTINUE WHILE
    ENDIF
    PRINT i
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

### EXIT and CONTINUE

Both act on the **innermost** enclosing loop. Naming the loop kind
(`FOR` / `WHILE` / `DO`) is optional and purely documentation — `CONTINUE`
on its own means exactly the same as `CONTINUE FOR` inside a FOR loop.

| Statement | Effect |
|-----------|--------|
| `EXIT FOR` / `EXIT WHILE` / `EXIT DO` | Leave the loop altogether |
| `CONTINUE` / `CONTINUE FOR` / `CONTINUE WHILE` / `CONTINUE DO` | Abandon the rest of this pass and start the next one |
| `EXIT SUB` / `EXIT FUNCTION` | Return from the routine |

```basic
FOR i = 1 TO 100
    IF i MOD 3 = 0 THEN
        CONTINUE FOR       ' skip multiples of three
    ENDIF
    IF i > 50 THEN
        EXIT FOR           ' and stop altogether past fifty
    ENDIF
    PRINT i
NEXT
```

Both work inside `SELECT CASE`:

```basic
FOR i = 1 TO 10
    SELECT CASE grade(i)
        CASE 0
            CONTINUE FOR
        CASE 9
            EXIT FOR
        CASE ELSE
            PRINT i
    END SELECT
NEXT
```

`CONTINUE` is only legal inside a loop. Writing it anywhere else — including
in a lambda that happens to sit inside a loop, since the lambda is its own
routine — is reported as an error:

```
semantic error at line 12: CONTINUE is only allowed inside a FOR, WHILE or DO loop
```

**One thing to watch in a post-test loop.** `DO ... LOOP WHILE` normally lets
its condition refer to a variable declared inside the loop body. A body
containing `CONTINUE` is compiled differently so that the test still runs
after a `CONTINUE`, and in that form the condition is evaluated outside the
body. Declare the variable before the `DO` if you need both:

```basic
DIM line AS STRING            ' declared outside, so LOOP WHILE can see it
DO
    line = nextLine()
    IF line = "" THEN
        CONTINUE DO
    ENDIF
    process(line)
LOOP WHILE line <> "EOF"
```

`CONTINUE` is a **contextual** keyword: it only has this meaning at the start
of a statement, standing alone or followed by `FOR`, `WHILE` or `DO`. A
program that already uses `CONTINUE` as a variable or field name keeps
working.

### GOTO (Use Sparingly)

```basic
GOTO label

label:
    ' statements
```

### WITH

`WITH expr ... END WITH` introduces a block where `.field` is shorthand for `expr.field`. Useful for property-heavy types. The receiver expression is re-evaluated per `.field` access (so it should normally be a simple lvalue). Nested WITHs are supported.

```basic
DIM p AS Point
WITH p
    .X = 3.0
    .Y = 4.0
    .Label = "origin"
END WITH

' Nested
WITH config.window
    .Title = "Hello"
    WITH .Position
        .X = 100
        .Y = 50
    END WITH
END WITH
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

### Anonymous Functions (Lambdas)

A `FUNCTION(...) AS T ... END FUNCTION` or `SUB(...) ... END SUB`
expression evaluates to a function value. Use this when a Go API
expects a callback. The body can read variables from the enclosing
scope (closure capture).

```basic
IMPORT "sort" AS sort

SUB Main()
    DIM nums AS []INTEGER = []INTEGER{3, 1, 4, 1, 5, 9, 2, 6}

    ' Pass an inline comparator to sort.Slice. The lambda captures
    ' `nums` from the surrounding scope.
    sort.Slice(nums, FUNCTION(i AS INTEGER, j AS INTEGER) AS BOOLEAN
        RETURN nums[i] < nums[j]
    END FUNCTION)

    PRINT nums
END SUB
```

Use `SUB(...)` instead of `FUNCTION(...)` when the callback returns
nothing — for example, an `http.HandleFunc` handler:

```basic
IMPORT "net/http" AS http
IMPORT "fmt" AS fmt

SUB Main()
    http.HandleFunc("/hello",
        SUB(w AS http.ResponseWriter, r AS POINTER TO http.Request)
            fmt.Fprintln(w, "hello from a DBasic lambda")
        END SUB)
    http.ListenAndServe(":8080", NIL)
END SUB
```

Anonymous `FUNCTION` literals require an explicit return type after the
parameter list (same syntax as named functions). Anonymous `SUB`
literals omit the return-type clause.

### Variadic Spread (`xs...`)

Follow a slice argument with `...` to spread its elements into a variadic
parameter — the same as Go's `f(xs...)`. It may only appear on the final
argument of a call, and the spread value must be a slice whose element type
matches the variadic parameter.

```basic
IMPORT "fmt"

SUB Main()
    DIM parts AS []ANY
    parts = APPEND(parts, "score")
    parts = APPEND(parts, 42)

    fmt.Println(parts...)             ' -> fmt.Println(parts...)
    fmt.Println("result:", parts...)  ' prefix args are fine before the spread
END SUB
```

This is what lets you build an argument list at runtime (conditionally
`APPEND`-ing options, say) and then hand the whole slice to a variadic Go
function.

### DECLARE (Forward References)

`DECLARE SUB Name(params)` and `DECLARE FUNCTION Name(params) AS T` are accepted as no-ops. DBasic auto-resolves forward references, so the syntax is supported only for QB compatibility.

```basic
DECLARE SUB Hello(name AS STRING)
DECLARE FUNCTION Twice(x AS INTEGER) AS INTEGER

SUB Main()
    Hello("Rick")
    PRINT Twice(7)
END SUB
```

### CALL

`CALL` is an optional prefix on a sub call. The compiler strips it and parses the rest as an ordinary call expression.

```basic
CALL Hello("Rick")    ' equivalent to: Hello("Rick")
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

' Append (expression form: returns the new slice)
numbers = APPEND(numbers, 1)
numbers = APPEND(numbers, 2, 3, 4)

' Append (statement form: mutates in place)
APPEND(numbers, 5)
APPEND(numbers, 6, 7)

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

## Maps

A `MAP OF K TO V` is a lookup table from keys of type `K` to values of
type `V`. It corresponds directly to Go's `map[K]V`.

### Declaration and Use

```basic
' Declare an empty map (initialized automatically — no need for MAKE)
DIM phoneBook AS MAP OF STRING TO STRING

' Write
phoneBook["Alice"] = "555-0100"
phoneBook["Bob"]   = "555-0102"

' Read
PRINT phoneBook["Alice"]

' Number of entries
PRINT Len(phoneBook)
```

### Map of Structs

```basic
TYPE Contact
    DIM Email AS STRING
    DIM Phone AS STRING
END TYPE

DIM contacts AS MAP OF STRING TO Contact
contacts["Alice"] = Contact{Email: "alice@example.com", Phone: "555-0100"}
PRINT contacts["Alice"].Email
```

A map declared with `DIM m AS MAP OF K TO V` is initialized to an empty
(but writable) map automatically — you do not need a separate `MAKE`
call before assigning to it.

---

## Deferred Calls

`DEFER FuncName(args)` schedules a function call to run when the
enclosing `SUB` or `FUNCTION` returns — by any path: explicit `RETURN`,
falling off the end, or panic. Multiple deferred calls run in
last-in-first-out (LIFO) order.

This corresponds directly to Go's `defer` statement and is the
canonical way to clean up resources without forgetting cleanup at every
return path.

```basic
SUB ProcessFile(path AS STRING)
    DIM f AS POINTER TO os.File
    DIM err AS ERROR
    f, err = os.Open(path)
    IF err <> NIL THEN RETURN
    DEFER f.Close()                ' guaranteed to run on every return

    ' ... read from f ...
END SUB
```

`DEFER` requires a function call as its operand (not a statement).
Wrap inline cleanup logic in a small helper if needed:

```basic
SUB Cleanup(label AS STRING)
    PRINT "cleaning up:"; label
END SUB

SUB Demo()
    DEFER Cleanup("C")            ' runs THIRD
    DEFER Cleanup("B")            ' runs SECOND
    DEFER Cleanup("A")            ' runs FIRST (LIFO)
    PRINT "doing real work..."
END SUB
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

DBasic does **not** have `TRY`/`CATCH`/`FINALLY` and does not throw
exceptions. Errors are values of type `ERROR` (Go's `error` interface),
returned alongside other values from a function. The two acceptable
patterns are the canonical Go-style explicit check, and the `ONERR`
sugar that auto-emits the same boilerplate.

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

' Calling — canonical Go-idiomatic pattern.
DIM data AS STRING
DIM err AS ERROR
data, err = ReadData("config.txt")
IF err <> NIL THEN
    PRINT "Error: " & fmt.Sprintf("%v", err)
ENDIF
```

### ONERR — automatic err-checks

`ONERR` is a per-function compile-time directive that asks the compiler
to emit `if err != nil { ... }` after each multi-return assignment whose
final value is of type `ERROR`. There is no runtime mechanism — no
`defer`/`recover`, no exceptions, no hidden control flow. The generated
Go is exactly what a Go programmer would have written by hand.

Three forms:

| Form | Effect after each error-returning call |
|---|---|
| `ONERR GOTO Label`    | `if err != nil { goto Label }` |
| `ONERR GOFUNC Func`   | `if err != nil { Func(err); return }` |
| `ONERR GOTO 0`        | clear handler — back to manual checks |

`ONERR` is **lexical** and **per-function**. It does not cross
sub/function/method boundaries. Setting `ONERR` updates a per-function
slot in the codegen; `ONERR GOTO 0` clears it. Each new function starts
with no handler.

#### `ONERR GOTO`

```basic
IMPORT "os"

SUB Demo()
    ONERR GOTO Bad
    DIM data AS BYTES
    DIM err AS ERROR
    data, err = os.ReadFile("config.toml")     ' auto: if err != nil { goto Bad }

    DIM more AS STRING = "after the read"      ' DIMs after the protected
    DIM count AS INTEGER = 7                   ' call are fine — codegen
    PRINT more, count, " | bytes=", LEN(data)  ' hoists them to the top.
    RETURN
Bad:
    PRINT "read failed: ", err.Error()
END SUB
```

When a function contains `ONERR GOTO`, the compiler **automatically
hoists every non-`STATIC` `DIM`** in the function body to the function
top (with zero-value declarations) and rewrites the in-body `DIM`s as
plain assignments. This sidesteps Go's "goto cannot jump over a
declaration" rule transparently — you can write `DIM`s anywhere in the
body.

#### `ONERR GOFUNC`

```basic
SUB ReportErr(e AS ERROR)
    PRINT "[handler] failed: ", e.Error()
END SUB

SUB Demo()
    ONERR GOFUNC ReportErr
    DIM data AS BYTES
    DIM err AS ERROR
    data, err = os.ReadFile("nonexistent.txt") ' auto: if err != nil { ReportErr(err); return }
    PRINT "read ", LEN(data), " bytes"         ' only reached on success
END SUB
```

`ONERR GOFUNC` emits a bare `return` after the handler call — fine in
SUBs and in FUNCTIONs whose signature accepts a bare return. For
FUNCTIONs that must return concrete values, prefer `ONERR GOTO`.

#### `ONERR GOTO 0` — clear

```basic
SUB Demo()
    ONERR GOTO Bad
    DIM data AS BYTES
    DIM err AS ERROR
    data, err = os.ReadFile("a.txt")           ' guarded
    ONERR GOTO 0
    data, err = os.ReadFile("b.txt")           ' NOT auto-checked
    IF err <> NIL THEN
        PRINT "manual check caught: ", err.Error()
    END IF
    RETURN
Bad:
    PRINT "guarded read failed: ", err.Error()
END SUB
```

#### How ONERR detects "this is an error return"

The analyzer marks a multi-assignment as ONERR-eligible when its final
target resolves to type `ERROR`. For DBasic-defined functions this is
read directly from the function signature; for external Go calls
(`os.ReadFile`, etc.), the marker is set when the last target was
pre-declared with `DIM err AS ERROR` (or any `ERROR`-typed name). The
canonical pattern of pre-DIMing the err variable is therefore both
correct *and* what unlocks ONERR:

```basic
DIM data AS BYTES
DIM err AS ERROR
data, err = os.ReadFile("x")
```

A full demo lives at `examples/onerr.dbas`.

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

' Receive with a comma-ok flag: ok is FALSE once the channel is closed
' and drained. Use this to loop until a producer closes the channel.
DIM ok AS BOOLEAN
RECEIVE value, ok FROM ch

' The `<-ch` operator is the expression form of a receive. It works
' anywhere an expression is allowed, and supports the comma-ok pattern
' in a multiple assignment:
value = <-ch
value, ok = <-ch
DIM doubled AS INTEGER = (<-ch) * 2

' Close channel
Close(ch)
```

> Lexing note: `<-` is read as the receive operator whenever the `<` and `-`
> are adjacent (as in Go), so comparing against a negative literal needs a
> space: write `x < -5`, not `x<-5`.

#### Directional channels

`CHAN OF T` is bidirectional. Prefix it with `RECEIVE` or `SEND` to get a
receive-only (`<-chan T`) or send-only (`chan<- T`) channel. This matters for
interop with Go APIs that hand back a directional channel — a bidirectional
`CHAN OF T` value is *not* assignable to a `<-chan T` slot, so you must name the
direction to capture such a return.

```basic
' Consumer takes a receive-only channel; producer takes a send-only one.
SUB Consume(ch AS RECEIVE CHAN OF INTEGER)
    DIM v AS INTEGER
    DIM ok AS BOOLEAN = TRUE
    WHILE ok
        RECEIVE v, ok FROM ch
        IF ok THEN PRINT v
    WEND
END SUB

SUB Produce(ch AS SEND CHAN OF INTEGER)
    SEND 1 TO ch
    Close(ch)
END SUB
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
        RECEIVE value, ok FROM ch
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

### Interface-Typed Parameters

DBasic accepts Go interface types like `io.Reader` and `io.Writer`
directly as parameter and variable types. Any concrete value satisfying
the interface — `*strings.Reader`, `*os.File`, `*bytes.Buffer`, etc. —
can be passed in. The DBasic compiler does not type-check interface
satisfaction itself; that check happens at `go build` time, where any
mismatch is reported against the `.dbas` source line via `//line`
directives.

```basic
IMPORT "io" AS io
IMPORT "strings" AS strings
IMPORT "os" AS os

' Take any io.Writer / io.Reader, forward to io.Copy.
SUB CopyInterface(dst AS io.Writer, src AS io.Reader)
    io.Copy(dst, src)
END SUB

SUB Main()
    DIM r AS POINTER TO strings.Reader = strings.NewReader("hello via io.Reader")
    CopyInterface(os.Stdout, r)
END SUB
```

### Calling Generic Go Functions

Go's generic functions (e.g. `slices.Sort`, `slices.Contains`) work
through type inference. DBasic does not require explicit type-parameter
syntax — Go infers the type from the argument:

```basic
IMPORT "slices" AS slices

SUB Main()
    DIM nums AS []INTEGER = []INTEGER{3, 1, 4, 1, 5}
    slices.Sort(nums)            ' [int] inferred from nums
    PRINT nums
END SUB
```

Explicit type-parameter syntax (`Sort[int]`) is rarely needed in
practice and is not currently part of the DBasic surface.

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
| `INPUT var` | Read whitespace-delimited token from stdin |
| `LINE INPUT [prompt$,] var$` | Read a whole line including embedded spaces |
| `Printf(fmt, args...)` | Formatted print |
| `Sprintf(fmt, args...)` | Formatted string |

`LINE INPUT` accepts an optional prompt followed by `,` or `;` and a target variable. With no prompt, it just reads:

```basic
DIM name AS STRING
LINE INPUT "What is your name? ", name
PRINT "Hello, "; name

DIM line AS STRING
LINE INPUT line
```

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

## Shell Functions

DBasic provides built-in functions for executing external programs and shell commands, similar to Go's `os/exec` package.

### Shell

Executes a command via the system shell (`/bin/sh -c`) and returns the output, exit code, and any error message.

```basic
FUNCTION Shell(command AS STRING) AS (STRING, INTEGER, STRING)
```

**Parameters:**
- `command` - The shell command to execute

**Returns:**
- Standard output as STRING
- Exit code as INTEGER (0 for success, -1 for execution failure)
- Error/stderr output as STRING (empty if successful)

**Example:**
```basic
DIM output AS STRING
DIM exitCode AS INTEGER
DIM errMsg AS STRING

' Run a simple command
output, exitCode, errMsg = Shell("echo Hello World")
PRINT output  ' Output: Hello World

' Run multiple commands
output, exitCode, errMsg = Shell("cd /tmp && ls -la")

' Check for errors
IF exitCode <> 0 THEN
    PRINT "Command failed with exit code: " & Str(exitCode)
    PRINT "Error: " & errMsg
ENDIF
```

### ShellExec

Executes a specific program with arguments (without using a shell).

```basic
FUNCTION ShellExec(program AS STRING, args AS STRING) AS (STRING, INTEGER, STRING)
```

**Parameters:**
- `program` - The program/executable to run
- `args` - Space-separated arguments

**Returns:**
- Standard output as STRING
- Exit code as INTEGER
- Error/stderr output as STRING

**Example:**
```basic
DIM output AS STRING
DIM code AS INTEGER
DIM err AS STRING

' Run ls with arguments
output, code, err = ShellExec("ls", "-la /home")

' Run git command
output, code, err = ShellExec("git", "status --short")

' Compile a program
output, code, err = ShellExec("go", "build -o myapp main.go")
IF code = 0 THEN
    PRINT "Build successful!"
ELSE
    PRINT "Build failed: " & err
ENDIF
```

### ShellStart

Starts a command in the background and returns immediately with the process ID.

```basic
FUNCTION ShellStart(command AS STRING) AS (INTEGER, STRING)
```

**Parameters:**
- `command` - The shell command to run in background

**Returns:**
- Process ID (PID) as INTEGER
- Error message as STRING (empty if successful)

**Example:**
```basic
DIM pid AS INTEGER
DIM err AS STRING

' Start a long-running process
pid, err = ShellStart("sleep 60")
IF err = "" THEN
    PRINT "Started background process with PID: " & Str(pid)
ENDIF

' Start a server in background
pid, err = ShellStart("python -m http.server 8080")
```

---

## Keywords Reference

### Declaration Keywords
`DIM`, `LET`, `CONST`, `TYPE`, `AS`, `EMBED`, `IMPLEMENTS`, `REDIM`, `PRESERVE`, `STATIC`, `SHARED`, `OPTION`, `EXPLICIT`, `BASE`

### Data Type Keywords
`INTEGER`, `LONG`, `SINGLE`, `DOUBLE`, `STRING`, `BOOLEAN`, `BYTES`, `BSTRING`, `JSON`, `POINTER`, `CHAN`, `MAP`, `ANY`, `ERROR`

### Control Flow Keywords
`IF`, `THEN`, `ELSE`, `ELSEIF`, `ENDIF`, `FOR`, `TO`, `STEP`, `NEXT`, `WHILE`, `WEND`, `DO`, `LOOP`, `UNTIL`, `EXIT`, `CONTINUE`, `SELECT`, `CASE`, `END`, `GOTO`, `RETURN`, `DEFER`, `WITH`

### Function Keywords
`SUB`, `FUNCTION`, `BYREF`, `BYVAL`, `DECLARE`, `CALL`

### Error-Handling Keywords
`ONERR`, `GOFUNC` — see [Error Handling → ONERR](#onerr--automatic-err-checks). DBasic has no `TRY`/`CATCH`/`FINALLY`.

### Logical Keywords
`AND`, `OR`, `NOT`, `XOR`, `MOD`

### Value Keywords
`TRUE`, `FALSE`, `NIL`

### I/O Keywords
`PRINT`, `INPUT`, `LINE INPUT`, `REM`

### Go Integration Keywords
`IMPORT`, `SPAWN`, `SEND`, `RECEIVE`, `FROM`, `MAKE_CHAN`, `OF`, `INCLUDE`

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

# Reformat source (token-stream, comment-preserving, idempotent)
dbasic fmt program.dbas              # print to stdout
dbasic fmt -w program.dbas           # write back in place
dbasic fmt -l examples/*.dbas        # list files needing formatting

# Generate Markdown API docs from a .dbas file (top-level decls + leading ' comments)
dbasic doc program.dbas              # print Markdown to stdout
dbasic doc -o api.md program.dbas    # write to api.md

# Run *_test.dbas files; subs named Test* are invoked in a recover() wrapper
dbasic test                          # current dir, recursive
dbasic test ./tests
dbasic test mypkg_test.dbas          # single file

# Cross-compile (auto-disables CGO; .exe added for windows targets)
dbasic build program.dbas --target windows/amd64
dbasic build program.dbas --target linux/arm64
dbasic build program.dbas --target darwin/arm64

# Build using only cached Go modules (when proxy.golang.org is unreachable)
dbasic build program.dbas --offline

# Show version
dbasic version

# Show help
dbasic help
```

### Explicit Generic Instantiation

When Go's type inference can't pin the type parameters, use the `(OF T1, T2, ...)` form to specify them explicitly:

```basic
IMPORT "slices"

SUB Main()
    DIM nums AS []INTEGER = [3, 1, 4, 1, 5, 9, 2, 6]
    slices.Sort(OF []INTEGER)(nums)   ' emits: slices.Sort[[]int](nums)
END SUB
```

`go build` errors and runtime panics are reported against your `.dbas`
source file and line number, not the temporary `main.go` — DBasic emits
`//line` directives during transpilation that the Go compiler honors
natively.

### Debugging

Because of those `//line` directives, the standard Go debugger
[Delve](https://github.com/go-delve/delve) sees DBasic source files
directly. Build with debug symbols (the default), then run dlv
against the resulting binary and set breakpoints in `.dbas` files:

```bash
dbasic build hello.dbas -o hello
dlv exec ./hello
(dlv) break hello.dbas:23
(dlv) continue
(dlv) print myVar
```

Delve resolves `.dbas:line` directly to the matching machine
instruction. Run `dlv` from the directory containing the `.dbas`
source so it can find the file by its `//line`-recorded relative
path.

### Editor Support (LSP)

`dbasic-lsp` is a Language Server Protocol implementation for any LSP-aware editor (VS Code, Neovim, Helix, Emacs, JetBrains). It runs over stdio and provides:

| Capability | What it does |
|-----------|--------------|
| Diagnostics | Live parse + analyze errors as you type |
| Document symbols | File outline (TYPEs, SUBs, FUNCTIONs, fields) |
| Hover | Signature info for top-level identifiers |
| Go-to-definition | Jump to top-level decl |
| Find references | Locate every use, scope-aware (locals stay in their sub) |
| Rename | Rename a symbol with the same scope rules |
| Completions | Keywords + top-level idents; type-narrowed members after `.` |
| Signature help | Parameter info inside a `(...)` call |

Install with `go install ./cmd/dbasic-lsp`. The VS Code package at `tools/vscode-dbasic/` bundles it for VS Code; for Neovim, point your LSP client at the `dbasic-lsp` binary for `*.dbas` files.

### Go-Build Errors Are DBasic-Flavored

When the Go compiler reports an error against generated code, DBasic strips the `# dbasic_program` package header and rewrites common Go phrasings into BASIC-flavored ones. For example, `cannot use X (variable of type T1) as T2 value in argument to F` becomes `type mismatch: X is T1 but F expects T2`. The file:line in the error always points to your `.dbas` source via `//line`.

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
