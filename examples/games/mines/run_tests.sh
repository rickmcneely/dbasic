#!/bin/bash
# Run the BombSquad tests.
#
# mines.dbas is a single self-contained program, so to test its insides we
# take a copy with its Main chopped off, glue mines_tests.dbas on the end
# (that file supplies its own Main), and build and run that. Nothing is
# written into the repository and no window is opened.
set -e
cd "$(dirname "$0")"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

awk '/^. ======================== 9\. MAIN/{exit} {print}' mines.dbas  > "$work/mines_tests.dbas"
cat mines_tests.dbas                                                  >> "$work/mines_tests.dbas"

dbasic build "$work/mines_tests.dbas" -o "$work/mines_tests"
"$work/mines_tests"
