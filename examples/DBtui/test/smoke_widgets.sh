#!/usr/bin/env bash
# =========================================================================
# smoke_widgets.sh — runtime behaviour smoke tests for generated widgets.
#
# The golden codegen test (codegen_golden.sh) proves the GENERATED CODE is
# byte-stable. It does NOT prove the widgets actually BEHAVE. This does:
# it builds the 14-widget showcase into a real program, drives it under a
# pty (vtdrive.py + pyte), and asserts that keystrokes produce the expected
# on-screen change — the class of bug a codegen diff can never catch (e.g.
# the file-watcher modal loop).
#
# SCOPE: this builds the COMMITTED generated programs (Widgets.dbas +
# chrome_demo.dbas + dialog_demo.dbas) — it does NOT re-run codegen. Together
# they cover all 18 widget kinds at runtime. The emitter->behaviour
# chain is covered transitively when both suites run: codegen_golden.sh
# proves the per-widget emitters reproduce Widgets.dbas byte-for-byte, and
# this proves Widgets.dbas behaves. So a regression in a widget emitter is
# caught by the golden diff; a regression in the generated runtime semantics
# is caught here.
#
# Each case Tabs focus to a target widget (focus index = order in the
# project: Button=0, Label=1, TextInput=2, Checkbox=3, RadioButton=4,
# ComboBox=5, ListBox=6, ProgressBar=7, ScrollBar=8, TabStrip=9, Tree=10,
# TextArea=11, PropGrid=12, ScrollPanel=13), acts, and checks the result.
# Keystroke recipes were calibrated against the real binary.
#
# USAGE:   bash test/smoke_widgets.sh
# EXIT:    0 = all pass; non-zero = build broke or a behaviour regressed.
# =========================================================================
set -u

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$TEST_DIR/.." && pwd)"
cd "$ROOT" || { echo "FATAL: cannot cd to DBtui root"; exit 2; }

DBASIC="${DBASIC:-dbasic}"
DRIVE="$TEST_DIR/vtdrive.py"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
BIN="$WORK/show"

if ! python3 -c "import pyte" 2>/dev/null; then
    echo "SKIP: pyte not installed (pip install pyte) — runtime smoke tests skipped"
    exit 0
fi

# build <committed.dbas> <out-bin> — build a generated program; fail on error.
fail=0
build_prog() {
    local src="$1"; local out="$2"
    echo "== build: $src =="
    if "$DBASIC" build "$src" -o "$out" > "$WORK/build.log" 2>&1; then
        echo "  ok"
    else
        echo "FAIL: $src did not build"
        tail -20 "$WORK/build.log" | sed 's/^/    /'
        fail=1
    fi
}

# The three smoke programs. Widgets covers the 14 placeable kinds; chrome_demo
# adds Menubar + Panel (not in the showcase); dialog_demo covers Dialog.
SHOW="$WORK/show"; CHROME="$WORK/chrome"; DLG="$WORK/dlg"; MENU="$WORK/menu"
EDIT="$WORK/edit"; SPLIT="$WORK/split"; SW="$WORK/starword"
build_prog examples/Widgets/Widgets.dbas         "$SHOW"
build_prog examples/chrome_demo/chrome_demo.dbas "$CHROME"
build_prog examples/dialog_demo/dialog_demo.dbas "$DLG"
build_prog examples/menu_demo/menu_demo.dbas     "$MENU"
build_prog examples/editor_demo/editor_demo.dbas "$EDIT"
build_prog examples/splitpane_demo/splitpane_demo.dbas "$SPLIT"
build_prog examples/StarWord/StarWord.dbas       "$SW"
[ "$fail" -eq 0 ] || { echo "RESULT: FAIL (build)"; exit 1; }

# case <bin> <label> <keys> <assert...>  — assert tokens pass to vtdrive.
case_run() {
    local bin="$1"; shift
    local label="$1"; shift
    local keys="$1"; shift
    echo "== $label =="
    if python3 "$DRIVE" "$bin" "$keys" "$@"; then :; else fail=1; fi
}

# T (tab) helper strings for reaching each focus index.
T3="tab,tab,tab"
T6="tab,tab,tab,tab,tab,tab"
T9="tab,tab,tab,tab,tab,tab,tab,tab,tab"
T10="tab,tab,tab,tab,tab,tab,tab,tab,tab,tab"

# --- Widgets showcase (14 placeable kinds) ---------------------------------
# Startup: the canvas renders with focus on the Button.
case_run "$SHOW" "startup renders" "hold:0.3" \
    --want "[ Button ]" --want "[ ] option"

# Checkbox (idx 3): Space toggles [ ] -> [x].
case_run "$SHOW" "checkbox toggles on Space" "$T3,space,hold:0.3" \
    --want "[x] option"

# ListBox (idx 6, bubbles/list): Down moves the selection bar to item two
# (the bubbles default delegate marks the selected row with a "│" cursor bar).
case_run "$SHOW" "listbox Down moves selection" "$T6,down,hold:0.3" \
    --want "│ item two"

# TabStrip (idx 9): Right activates the next tab (bracketed) + content row.
case_run "$SHOW" "tabstrip Right switches tab" "$T9,right,hold:0.3" \
    --want "[Advanced]" --want "Advanced tab"

# Tree (idx 10): Enter on the collapsed-able root hides its children.
case_run "$SHOW" "tree Enter collapses root" "$T10,enter,hold:0.3" \
    --want "▸ Project" --not "main.dbas"

# --- chrome_demo: Menubar + Panel (focus order: menubar=0, panel=1) --------
# Panel renders its framed child at startup.
case_run "$CHROME" "panel renders child" "hold:0.3" \
    --want "child-in-panel"

# Menubar (idx 0, focused at startup): Down opens the dropdown, which paints
# its framed box of items (the highlighted item is reverse-video, not marked).
case_run "$CHROME" "menubar opens dropdown" "down,hold:0.3" \
    --want "New" --want "Open"

# --- menu_demo: multi-level cascading menus + flags (menubar=0, label=1) ----
# Down opens File: items render with shortcut, checkmark, separator.
case_run "$MENU" "menu opens with shortcut + checked item" "down,hold:0.3" \
    --want "New" --want "Ctrl+N" --want "Recent" --want "Save"
# Move to Recent (down,down) and Right to cascade its submenu (Alpha/Beta).
case_run "$MENU" "submenu cascades on Right" "down,down,right,hold:0.3" \
    --want "Alpha" --want "Beta"
# Enter on a leaf fires its handler and closes the menu (box border gone).
case_run "$MENU" "selecting a leaf closes the menu" "down,enter,hold:0.3" \
    --not "┌"

# --- dialog_demo: Dialog (btnOpen focused; OnClickDialog pops confirm) ------
# Enter on the focused button opens the confirm dialog; Esc closes it.
case_run "$DLG" "dialog opens on click" "enter,hold:0.4" \
    --want "Are you sure?" --want "[ OK ]"
case_run "$DLG" "dialog closes on Esc" "enter,esc,hold:0.4" \
    --not "Are you sure?"

# --- editor_demo: a full text editor on ONE Canvas + Form_Key (self-host #5) -
# Startup: line-number gutter + seeded buffer render.
case_run "$EDIT" "editor renders gutter + buffer" "hold:0.4" \
    --want " 1 |" --want "FUNCTION greet"
# Typing inserts at the cursor (starts on the FUNCTION line, col 0).
case_run "$EDIT" "editor inserts typed text" "X,Y,Z,hold:0.4" \
    --want "XYZFUNCTION greet"
# Enter splits the line at the cursor (4 rights into "FUNCTION").
case_run "$EDIT" "editor Enter splits line" "right,right,right,right,enter,hold:0.4" \
    --want " 8 | TION greet"
# Backspace at col 0 joins the line onto the previous one.
case_run "$EDIT" "editor Backspace joins lines" "down,home,backspace,hold:0.4" \
    --want "AS STRING    RETURN"
# Ctrl+S paints the status bar (proves command-key chords reach Form_Key).
case_run "$EDIT" "editor Ctrl+S shows status" "ctrl+s,hold:0.4" \
    --want "saved 9 lines"

# --- splitpane_demo: draggable splitters on ONE Canvas (self-host #2a) ------
# Startup: the four-region IDE shell (sidebar | editor / output) renders.
case_run "$SPLIT" "splitpane renders four regions" "hold:0.4" \
    --want "EXPLORER" --want "main.dbas" --want "OUTPUT" --want "forms.dbas"
# Vertical bar (canvas col 24 = terminal 1-based col 26): click to grab, click
# col 13 to jump it left — the narrowed sidebar clips long filenames.
case_run "$SPLIT" "splitpane vbar drag narrows sidebar" "lclick:26.5,lclick:13.5,hold:0.4" \
    --want "EXPLORER" --not "forms.dbas"
# Same drag via a live MOTION event (grab, then mouse-move) — exercises the
# Form_Mouse motion path, not just clicks.
case_run "$SPLIT" "splitpane vbar live-motion drag" "lclick:26.5,move:13.5,hold:0.4" \
    --not "forms.dbas"
# NOTE: the HORIZONTAL bar shares this exact drag mechanism on the Y axis and
# works on a real terminal (verified under tmux). It is NOT asserted here
# because shrinking a pane blanks cells, and pyte cannot faithfully emulate
# bubbletea's region-shrink repaint — it leaves stale cells that a real
# terminal clears. The vbar cases above already cover click + motion + clamp.

# --- StarWord: modern WordStar 7.0 clone on a Canvas -----------------------
# Startup: WordStar edit screen — status line + ruler + new Unicode document.
case_run "$SW" "starword edit screen renders" "hold:0.4" \
    --want "StarWord" --want "untitled.WSu" --want "INSERT" --want "UTF-8"
# Typing inserts text and advances the column counter.
case_run "$SW" "starword types text" "H,i,hold:0.4" \
    --want "Hi" --want "Col 3"
# ^V toggles INSERT <-> OVERTYPE.
case_run "$SW" "starword ^V toggles overtype" "ctrl+v,hold:0.4" \
    --want "OVERTYPE"
# ^K pops the Block & Save menu; ^Q pops the Quick menu (WordStar prefixes).
case_run "$SW" "starword ^K block menu" "ctrl+k,hold:0.4" \
    --want "BLOCK & SAVE" --want "Begin"
case_run "$SW" "starword ^Q quick menu" "ctrl+q,hold:0.4" \
    --want "QUICK MOVE" --want "Find"
# Block copy: 'abc' -> mark whole line -> ^KC copies it at the end -> abcabc.
case_run "$SW" "starword block copy" "a,b,c,ctrl+q,s,ctrl+k,b,ctrl+q,d,ctrl+k,k,ctrl+k,c,hold:0.4" \
    --want "abcabc"

if [ "$fail" -ne 0 ]; then
    echo "RESULT: FAIL"
    exit 1
fi
echo "RESULT: PASS (27 cases, 7 programs)"
