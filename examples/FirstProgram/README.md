# FirstProgram

A tiny DBasic example that uses **one Go package** and **one DBasic include**.

## What does it do? (ELI5)

Imagine your computer is a friendly robot.

1. The robot says **"What is your name?"** and waits for you to type your name.
2. You type something like `alice` and press Enter.
3. The robot uses a **dictionary helper from Go** (called `strings`) to turn your name into **ALL CAPS**: `ALICE`.
4. Then the robot uses a **little helper file** (`greeter.dbas`) that knows how to say hello, and it greets you: **"Hello, ALICE! Welcome to DBasic."**

That's the whole program.

## Two ways DBasic borrows code

This program shows the two main ways DBasic pulls in code that lives somewhere else.

### 1. `IMPORT "strings"` — borrow a **Go** package

Go (the language DBasic compiles to) ships with hundreds of useful "packages." A package is just a folder full of helper functions. The `strings` package knows how to do things to text — chop it, glue it, search it, change its case.

When the program says

```basic
IMPORT "strings"
```

it tells DBasic: *"I want to use the `strings` package."* After that, you can call any function from it. We use `strings.ToUpper(...)`, which takes a piece of text and gives you back the same text in capital letters.

### 2. `INCLUDE "greeter.dbas"` — paste in another **DBasic** file

Sometimes you want to split your own code into more than one file. Maybe one file holds the main program, and another file holds little helper subroutines. DBasic's `INCLUDE` is like a magic copy-paste button:

```basic
INCLUDE "greeter.dbas"
```

When DBasic sees this, it pretends the entire contents of `greeter.dbas` were typed right there. So `SUB SayHello`, defined inside `greeter.dbas`, becomes available to call from `FirstProgram.dbas`.

## Files

| File | What it has |
|------|-------------|
| `FirstProgram.dbas` | The main program. Asks for a name, calls a Go package, calls our helper. |
| `greeter.dbas` | A helper file with one SUB: `SayHello(name)`. |
| `README.md` | This file. |

## How to compile and run it

You need DBasic installed (`go install ./cmd/dbasic` from the repo root, or download a binary).

### Option A: just run it (one command)

```sh
cd examples/FirstProgram
dbasic run FirstProgram.dbas
```

`dbasic run` compiles the program in a temp folder and runs it immediately.

### Option B: build an executable, then run

```sh
cd examples/FirstProgram
dbasic build FirstProgram.dbas -o firstprogram
./firstprogram
```

`dbasic build` produces a real native executable named `firstprogram` (or `firstprogram.exe` on Windows). You can run it without DBasic being installed afterward.

### Option C: build for a different OS/CPU

```sh
dbasic build FirstProgram.dbas --target windows/amd64 -o firstprogram.exe
dbasic build FirstProgram.dbas --target darwin/arm64  -o firstprogram-mac
dbasic build FirstProgram.dbas --target linux/arm64   -o firstprogram-pi
```

The `--target` flag tells DBasic to cross-compile for that platform.

## What you'll see

```
$ dbasic run FirstProgram.dbas
What is your name? alice
Hello, ALICE! Welcome to DBasic.
```

## What to try next

- Change `strings.ToUpper` to `strings.ToLower` and rebuild. Now your name comes back lower-cased.
- Add another SUB to `greeter.dbas` (say, `SayGoodbye(name)`) and call it after `SayHello`.
- Add `IMPORT "time"` and have the program print `time.Now().Format("2006-01-02")` at the end.

## Where to learn more

- [Language reference](../../docs/LANGUAGE_REFERENCE.md) — every keyword, every operator, every built-in.
- [Other examples](..) — slightly bigger programs that show off slices, structs, JSON, HTTP, channels, web servers, games.
