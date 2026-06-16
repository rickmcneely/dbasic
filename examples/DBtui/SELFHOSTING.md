# DBtui self-hosting — roadmap & resume point

Goal: DBtui (the visual TUI IDE, written in DBasic) can eventually recreate
itself. The blockers were five capability gaps between "form-based app
generator" and "arbitrary TUI app." This doc is the live plan; pick up here.

## Status

**Foundation done — #1 (custom-draw surface) + #3 (raw input).** Commit
`1e38611`. These were the real framework blockers; #2/#4/#5 are now buildable
*on* this foundation rather than being separate frameworks.

**#5 done — Editor on a Canvas.** `examples/editor_demo/` is a complete
multi-line text editor built on ONE Canvas + `Form_Key`/`cv_Click`: a
line-number gutter, reverse-video cursor, insert/move (arrows/home/end/pgup/
pgdn), Enter-split / Backspace-join / Delete-pull-up, vertical + horizontal
scroll that keeps the cursor in view, click-to-place, and a Ctrl+S status
bar. In both regression suites (golden #9; smoke: 5 editor cases). This proves
the IDE's own editor (`internal/pad.dbas`) is reproducible in generated code.
Harness fix: `test/vtdrive.py` gained home/end/delete/pgup/pgdown +
Ctrl+<letter> keys and disables pty XON/XOFF so Ctrl+S reaches the program.

**#2(a) done — split-pane layout on a Canvas.** `examples/splitpane_demo/` is
an IDE shell (sidebar | editor / output) drawn on ONE full-form Canvas, with a
vertical + a horizontal splitter dragged through the raw `Form_Mouse` hook.
Because the hooks deliver click + motion but no release, dragging uses a
grab/move/drop model: click a bar to grab, slide with motion (or click the new
spot) to move, click to drop. This is exactly how the IDE's own splitters
work. Verified: both axes drag + clamp on a real terminal (tmux). NOTE: pyte
(the smoke harness's VT emulator) cannot emulate bubbletea's region-shrink
repaint — it leaves stale cells a real terminal clears — so the smoke suite
asserts the vbar drag (which grows/clips rather than blanks) via both click and
motion; the hbar shares that code path and is tmux-verified.

- **Canvas widget** (`internal/widgets/canvas.dbas` + wiring in `internal/form.dbas`):
  a `W×H` rectangle the program draws via `FUNCTION <Name>_Render(w, h) AS STRING`
  (every frame, ANSI passed through, size-pinned + stretch-aware). Stateless.
  Raw left-clicks → `SUB <Name>_Click(localX, localY)` (canvas-local cells).
- **Raw hooks** (emitted only when a form has a Canvas → existing goldens
  byte-identical): `SUB Form_Key(key)` (every key, before built-ins; ctrl+c/esc
  still quit, tab still cycles — no consume yet) and
  `SUB Form_Mouse(kind, button, x, y)` (kind = `click`|`motion`|`wheel`).
- **Compiler fix** (`pkg/codegen/codegen.go`): package-level `MAP` globals now
  auto-init (were nil → panicked on first write).
- **Reference demo**: `examples/canvas_demo/` (hand-drawn paint grid; arrows move
  a cursor, space/click toggle dots; resizes via stretch anchoring). In the
  golden suite (8 projects).

## Remaining: #2, #4, #5 — concrete plans

### #5 — Editor on a Canvas  (RECOMMENDED NEXT — highest leverage)
Build a reusable multi-line text editor entirely on Canvas + Form_Key. Proves
the IDE's own editor (`internal/pad.dbas`) is reproducible in generated code.
- New `examples/editor_demo/`: one Canvas + globals `gLines AS []STRING`,
  `gRow`, `gCol`, `gTop` (scroll).
- `cv_Render(w,h)`: draw visible `gLines[gTop..]` with a line-number gutter;
  reverse-video the cell at (gRow,gCol); clip to w×h.
- `Form_Key(key)`: printable single chars insert at cursor; `left/right/up/down`
  move; `enter` splits the line; `backspace` joins/deletes; `home/end`.
  (Single printable chars arrive as the key string, e.g. `"a"`.)
- `cv_Click(x,y)`: place the cursor (account for the gutter width).
- Stretch the Canvas (Anchors `T,B,L,R`) so it fills the form.
- Success = a usable editor; then compare structure to `pad.dbas`.

### #2 — Split-pane layout
Two routes; do (a) first to prove it, (b) only if a palette widget is wanted.
- **(a) app-level, on Canvas** — `examples/splitpane_demo/`: one full-form
  Canvas that draws N panes + splitter bars; `Form_Mouse("click"/"motion")`
  starts/continues a splitter drag (track a `gDragSplit` index + update a
  `gSplitX/gSplitH`). This is exactly how the IDE's own splitters work.
- **(b) first-class widget** — a `SplitPane`/`Dock` container kind (like
  `Panel`) hosting child widgets in resizable regions. Bigger: needs runtime
  child re-layout. Follow the widget-kind checklist (≈10 touch points in
  `form.dbas`; see the Canvas commit / the `dbtui_canvas_selfhost_foundation`
  memory) + child-layout codegen in the View.

### #4 — Dynamic UI
- **Dynamic *rendering* is already unblocked** by Canvas (render anything at
  runtime). For self-hosting this is likely enough — the IDE draws its panels
  via custom render anyway.
- **Runtime *widget* creation** (instantiate a real ListBox/Button after start)
  is the deeper, separate change: the generated `View` is static (widgets fixed
  at design time). Approach if needed: a runtime widget registry + a generic
  "render a widget spec" path. Defer unless a concrete need appears; prefer
  Canvas-based dynamic UI first.

## Build / test / drive (commands)

```bash
# Rebuild the DBasic compiler (system go is 1.22; repo needs 1.25 at /usr/local/go)
cd ~/DBasic && GOWORK=off /usr/local/go/bin/go build -o /tmp/dbasic_new ./cmd/dbasic \
  && cp /tmp/dbasic_new ~/.local/bin/dbasic
GOWORK=off /usr/local/go/bin/go test ./pkg/...          # compiler tests

# Rebuild DBtui + regen a project + run a built program
cd ~/DBasic/examples/DBtui
dbasic build dbtui.dbas -o /tmp/dbtui_gen
/tmp/dbtui_gen gen examples/<proj>/<proj>.dbproj         # writes <proj>.dbas (preserves user section)
dbasic build examples/<proj>/<proj>.dbas -o /tmp/<proj>

# Regression suites
bash test/codegen_golden.sh      # .dbproj -> .dbas round-trip, byte-identical (8 projects)
bash test/smoke_widgets.sh       # pty+pyte runtime asserts

# Drive a TUI headlessly at a chosen size
cp test/vtdrive.py /tmp/v.py; sed -i 's/^COLS, ROWS = .*/COLS, ROWS = 80, 24/' /tmp/v.py
timeout 12 python3 /tmp/v.py /tmp/<proj> "hold:2,right,space,hold:1"   # tokens: keys, lclick:C.R, hold:N
```

## Adding a new widget kind — the ~10 touch points (all in `internal/form.dbas` unless noted)
`formKindByIdx` + `formPaletteKinds` (alphabetical) · `defaultWidget` ·
the `hasX` scan loop · `widgetVisibleText` + `widgetVisibleLines` (design
preview) · `widgetEmitLayer` dispatch · the per-kind emit module under
`internal/widgets/` · `INCLUDE` it in `dbtui.dbas` · `generateUserStubs`
(stubs; `declDefined` matches FUNCTION stubs) · Update hit-test loop (mouse) /
key dispatch. Stateless kinds (like Canvas) skip the Model-field loop.

## Recommended order
1. **#5 editor_demo** ✅ (proves the IDE's hardest surface is reproducible).
2. **#2(a) splitpane_demo** ✅ (proves the IDE shell).
3. Decide on #2(b) first-class SplitPane and #4 runtime widgets based on what
   self-hosting the property panel / dialogs actually needs.

## Decision on #2(b) + #4 — DEFERRED (Canvas covers it)
With #5 and #2(a) shipped, the two surfaces that were the actual blockers — a
multi-line editor and a resizable multi-pane shell — are both reproducible in
generated DBasic *on the Canvas foundation alone*, with no new framework. That
covers what self-hosting needs: the IDE already draws its panels, editor, and
splitters via custom render, and a Canvas + `Form_Key`/`Form_Mouse` reproduces
exactly that pattern.

So **#2(b)** (a first-class `SplitPane`/`Dock` widget kind) and **#4**
(runtime widget creation — instantiating real widgets after start) stay
**deferred**. They are conveniences, not blockers: #2(b) only matters if we
want splitters to host *design-time-placed child widgets* (vs. custom-drawn
content), and #4 only matters for UIs whose widget set isn't known until
runtime. Neither is required to self-host. Revisit only when a concrete app
surfaces a need that Canvas genuinely can't express — the `StarWord` app
(WordStar 7.0 clone, `examples/StarWord/`) is a good forcing function: build it
on Canvas first and let any real gap, if one appears, justify #2(b)/#4.
