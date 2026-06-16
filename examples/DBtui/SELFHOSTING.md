# DBtui self-hosting — roadmap & resume point

Goal: DBtui (the visual TUI IDE, written in DBasic) can eventually recreate
itself. The blockers were five capability gaps between "form-based app
generator" and "arbitrary TUI app." This doc is the live plan; pick up here.

## Status

**Foundation done — #1 (custom-draw surface) + #3 (raw input).** Commit
`1e38611`. These were the real framework blockers; #2/#4/#5 are now buildable
*on* this foundation rather than being separate frameworks.

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
1. **#5 editor_demo** (proves the IDE's hardest surface is reproducible).
2. **#2(a) splitpane_demo** (proves the IDE shell).
3. Decide on #2(b) first-class SplitPane and #4 runtime widgets based on what
   self-hosting the property panel / dialogs actually needs.
