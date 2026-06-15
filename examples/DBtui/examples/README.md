# DBtui worked examples

Sample `.dbproj` files that demonstrate the design-→-generate-→-edit loop.

## radio — internet radio player

A real-world demo: browse and play internet radio from the free
[radio-browser.info](https://www.radio-browser.info/) directory. It shows
off two things at once:

1. **Runtime-populated widgets.** The station ListBox is filled from live
   API data with the generated `lstStations_SetItems()` / `_SelectedIndex()`
   helpers — proof that a designed widget can be driven by handler code,
   not just static design-time items.
2. **An ffmpeg-backed audio pipeline:**
   `stream URL → ffmpeg (decode + resample) → github.com/ebitengine/oto/v3
   → speakers`. ffmpeg runs as a child process (`os/exec`, not Cgo, so the
   program still cross-compiles) and is asked for 44.1 kHz / stereo / s16le
   PCM on its stdout, which oto plays. On Linux/WSL oto talks to PulseAudio
   via the pure-Go `jfreymuth/pulse` client and falls back to ALSA.

Because ffmpeg does the demux/decode, **every codec works** — MP3, AAC and
AAC+/HE-AAC alike — so the handler code (below the `DBTUI:USER` marker in
`radio.dbas`) no longer filters the directory by codec. The only requirement
is that the `ffmpeg` binary is on the user's PATH; if it is missing, Play
shows a one-line hint instead of failing silently. Fixing ffmpeg's output to
a single sample rate also means oto's once-per-session device setup is always
correct — the old "different station, wrong sample rate" rough edge is gone.

```bash
# Regenerate the partner .dbas (preserves the handlers below the marker).
./DBtui gen examples/radio/radio.dbproj

# Build + run.
dbasic build examples/radio/radio.dbas -o examples/radio/radio && ./examples/radio/radio
```

Controls: type a name and Tab to **Search** (or **Top 100**), Tab into the
list and pick with the arrow keys, Tab to **Play** / **Stop**, Esc to quit.

## quit_demo

A quit-confirmation dialog modeled on DBtui's own quit modal: a message
Label and two Buttons (Discard / Cancel). The handler section below the
`DBTUI:USER` marker prints the user's choice to stderr and exits.

```bash
# Regenerate the partner .dbas without launching the TUI.
./dbtui gen examples/quit_demo/quit_demo.dbproj

# Build + run the demo program.
dbasic run examples/quit_demo/quit_demo.dbas
```

Open `examples/quit_demo/quit_demo.dbas` to see the split between the generated
prelude (above the marker) and your handlers (below). Editing the
handler bodies and re-running `dbtui gen` preserves your code; only the
prelude is rewritten.

## Round-trip from inside DBtui

You can also open `quit_demo.dbproj` from the project tree, rearrange
widgets on the canvas, and hit F5 — the partner `.dbas` regenerates
in-place and `dbasic run` takes the terminal. Your handler bodies
survive because they live below the `DBTUI:USER` sentinel that
`mergeFormDbas` scans for.
