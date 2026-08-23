#!/bin/bash
# Run the BombSquad tests.
#
# bombsquad.dbas is a single self-contained program, so to test its insides
# we take a copy with its Main chopped off, glue bombsquad_tests.dbas on the
# end (that file supplies its own Main), and build and run that. Nothing is
# written into the repository and no window is opened.
set -e
cd "$(dirname "$0")"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

awk '/^. ======================== 9\. MAIN/{exit} {print}' bombsquad.dbas  > "$work/bombsquad_tests.dbas"
cat bombsquad_tests.dbas                                              >> "$work/bombsquad_tests.dbas"

dbasic build "$work/bombsquad_tests.dbas" -o "$work/bombsquad_tests"
"$work/bombsquad_tests"
