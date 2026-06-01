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
# SCOPE: this builds the COMMITTED examples/Widgets/Widgets.dbas (the shipped
# generated program) — it does NOT re-run codegen. The emitter->behaviour
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

echo "== build: showcase program =="
# The committed Widgets.dbas is the generated showcase (all 14 placeable
# kinds). Build it directly — that's what a user would run.
if ! "$DBASIC" build examples/Widgets/Widgets.dbas -o "$BIN" > "$WORK/build.log" 2>&1; then
    echo "FAIL: showcase did not build"
    tail -20 "$WORK/build.log" | sed 's/^/    /'
    exit 1
fi
echo "  ok"

fail=0
# case <label> <keys> <assert...>   — assert tokens pass straight to vtdrive
case_run() {
    local label="$1"; shift
    local keys="$1"; shift
    echo "== $label =="
    if python3 "$DRIVE" "$BIN" "$keys" "$@"; then :; else fail=1; fi
}

# T (tab) helper strings for reaching each focus index.
T3="tab,tab,tab"
T6="tab,tab,tab,tab,tab,tab"
T9="tab,tab,tab,tab,tab,tab,tab,tab,tab"
T10="tab,tab,tab,tab,tab,tab,tab,tab,tab,tab"

# Startup: the canvas renders with focus on the Button.
case_run "startup renders" "hold:0.3" \
    --want "[ Button ]" --want "[ ] option"

# Checkbox (idx 3): Space toggles [ ] -> [x].
case_run "checkbox toggles on Space" "$T3,space,hold:0.3" \
    --want "[x] option"

# ListBox (idx 6): Down moves the selection caret to item two.
case_run "listbox Down moves selection" "$T6,down,hold:0.3" \
    --want "▶ item two"

# TabStrip (idx 9): Right activates the next tab (bracketed) + content row.
case_run "tabstrip Right switches tab" "$T9,right,hold:0.3" \
    --want "[Advanced]" --want "Advanced tab"

# Tree (idx 10): Enter on the collapsed-able root hides its children.
case_run "tree Enter collapses root" "$T10,enter,hold:0.3" \
    --want "▸ Project" --not "main.dbas"

if [ "$fail" -ne 0 ]; then
    echo "RESULT: FAIL"
    exit 1
fi
echo "RESULT: PASS (5 cases)"
