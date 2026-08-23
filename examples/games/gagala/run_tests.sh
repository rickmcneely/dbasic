#!/bin/bash
# Run the Gagala tests.
#
# gagala.dbas is a single self-contained program, so to test its insides we
# take a copy with its Main chopped off, glue gagala_tests.dbas on the end
# (that file supplies its own Main), and build and run that. Nothing is
# written into the repository and no window is opened.
set -e
cd "$(dirname "$0")"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

awk '/^. ======================== 13\. MAIN/{exit} {print}' gagala.dbas > "$work/gagala_tests.dbas"
cat gagala_tests.dbas                                                  >> "$work/gagala_tests.dbas"

dbasic build "$work/gagala_tests.dbas" -o "$work/gagala_tests"
"$work/gagala_tests"
