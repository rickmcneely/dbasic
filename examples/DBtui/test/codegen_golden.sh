#!/usr/bin/env bash
# =========================================================================
# codegen_golden.sh — regression test for DBtui's .dbproj -> .dbas codegen.
#
# WHY: the form-designer codegen is the core deliverable (a project must
# generate a compilable, correct bubbletea program). It has no other guard,
# and a broken commit once slipped through because an ad-hoc check diffed a
# STALE binary after a failed build. This script is that guard, made real:
#
#   1. BUILD GATE — rebuild the IDE from source. If it doesn't compile, the
#      test fails immediately (a stale binary can never mask a break again).
#   2. GOLDEN DIFF — for each example project, regenerate its .dbas via the
#      freshly-built `dbtui gen` and assert it is byte-identical to the
#      committed .dbas. Regen happens in a temp dir, so the committed
#      goldens are never touched.
#
# The committed <stem>.dbas files ARE the golden snapshots. The check is a
# ROUND-TRIP: the sandbox is seeded with BOTH the .dbproj and the committed
# .dbas, then `gen` runs. mergeFormDbas preserves the user section (the code
# below the DBTUI:USER marker) and regenerates the prelude, so idempotent
# codegen reproduces the committed file byte-for-byte. This mirrors the real
# IDE F5 path (the .dbas already exists) and means example projects may carry
# real user-handler code without the test wiping it to TODO stubs.
#
# If you intentionally change codegen, regenerate the goldens, eyeball the
# diff, and commit them — that updates the expected output.
#
# USAGE:   bash test/codegen_golden.sh
# EXIT:    0 = all pass; non-zero = build broke or some project drifted.
# =========================================================================
set -u

# Resolve the DBtui root (this script lives in <root>/test/).
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$TEST_DIR/.." && pwd)"
cd "$ROOT" || { echo "FATAL: cannot cd to DBtui root"; exit 2; }

DBASIC="${DBASIC:-dbasic}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
BIN="$WORK/dbtui_test"

# Projects to check: each is a path (relative to ROOT) to a .dbproj whose
# committed <stem>.dbas is the golden. Add new example projects here.
PROJECTS=(
  "examples/Widgets/Widgets.dbproj"
  "examples/dialog_demo.dbproj"
  "examples/quit_demo.dbproj"
)

fail=0

echo "== [1/2] build gate: $DBASIC build dbtui.dbas =="
if ! "$DBASIC" build dbtui.dbas -o "$BIN" > "$WORK/build.log" 2>&1; then
    echo "FAIL: IDE did not build"
    tail -20 "$WORK/build.log" | sed 's/^/    /'
    exit 1
fi
echo "  ok"

echo "== [2/2] golden codegen diff =="
for proj in "${PROJECTS[@]}"; do
    stem="$(basename "$proj" .dbproj)"
    dir="$(dirname "$proj")"
    golden="$ROOT/$dir/$stem.dbas"
    if [ ! -f "$golden" ]; then
        echo "  FAIL $proj — no committed golden $stem.dbas"
        fail=1
        continue
    fi
    # Regenerate in an isolated copy so the committed files are untouched.
    # Seed BOTH the .dbproj and the committed .dbas so `gen` preserves the
    # user section (round-trip), exactly as the IDE does on F5.
    sandbox="$WORK/$stem"
    mkdir -p "$sandbox"
    cp "$ROOT/$proj" "$sandbox/"
    cp "$golden" "$sandbox/"
    ( cd "$sandbox" && "$BIN" gen "$stem.dbproj" >/dev/null 2>&1 )
    if [ ! -f "$sandbox/$stem.dbas" ]; then
        echo "  FAIL $proj — gen produced no output"
        fail=1
        continue
    fi
    if diff -u "$golden" "$sandbox/$stem.dbas" > "$WORK/$stem.diff" 2>&1; then
        echo "  ok   $proj"
    else
        echo "  FAIL $proj — generated .dbas drifted from committed golden:"
        head -40 "$WORK/$stem.diff" | sed 's/^/      /'
        fail=1
    fi
done

if [ "$fail" -ne 0 ]; then
    echo "RESULT: FAIL"
    exit 1
fi
echo "RESULT: PASS (${#PROJECTS[@]} projects)"
