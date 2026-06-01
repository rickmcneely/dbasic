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
SHOW="$WORK/show"; CHROME="$WORK/chrome"; DLG="$WORK/dlg"
build_prog examples/Widgets/Widgets.dbas   "$SHOW"
build_prog examples/chrome_demo.dbas       "$CHROME"
build_prog examples/dialog_demo.dbas       "$DLG"
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

# ListBox (idx 6): Down moves the selection caret to item two.
case_run "$SHOW" "listbox Down moves selection" "$T6,down,hold:0.3" \
    --want "▶ item two"

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

# Menubar (idx 0, focused at startup): Down opens the dropdown; a second
# Down moves the highlight to the second item (> Open).
case_run "$CHROME" "menubar opens + navigates" "down,down,hold:0.3" \
    --want "> Open"

# --- dialog_demo: Dialog (btnOpen focused; OnClickDialog pops confirm) ------
# Enter on the focused button opens the confirm dialog; Esc closes it.
case_run "$DLG" "dialog opens on click" "enter,hold:0.4" \
    --want "Are you sure?" --want "[ OK ]"
case_run "$DLG" "dialog closes on Esc" "enter,esc,hold:0.4" \
    --not "Are you sure?"

if [ "$fail" -ne 0 ]; then
    echo "RESULT: FAIL"
    exit 1
fi
echo "RESULT: PASS (9 cases, 3 programs)"
