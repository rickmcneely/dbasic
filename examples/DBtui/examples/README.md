# DBtui worked examples

Sample `.dbproj` files that demonstrate the design-→-generate-→-edit loop.

## quit_demo

A quit-confirmation dialog modeled on DBtui's own quit modal: a message
Label and two Buttons (Discard / Cancel). The handler section below the
`DBTUI:USER` marker prints the user's choice to stderr and exits.

```bash
# Regenerate the partner .dbas without launching the TUI.
./dbtui gen examples/quit_demo.dbproj

# Build + run the demo program.
dbasic run examples/quit_demo.dbas
```

Open `examples/quit_demo.dbas` to see the split between the generated
prelude (above the marker) and your handlers (below). Editing the
handler bodies and re-running `dbtui gen` preserves your code; only the
prelude is rewritten.

## Round-trip from inside DBtui

You can also open `quit_demo.dbproj` from the project tree, rearrange
widgets on the canvas, and hit F5 — the partner `.dbas` regenerates
in-place and `dbasic run` takes the terminal. Your handler bodies
survive because they live below the `DBTUI:USER` sentinel that
`mergeFormDbas` scans for.
