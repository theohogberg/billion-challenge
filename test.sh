#!/bin/bash
# run against the reference samples from the 1brc repo
SAMPLES=1brc/src/test/resources/samples
BIN=build/brc
fail=0

go build -o $BIN . || exit 1

for f in $SAMPLES/*.txt testdata/*.txt; do
  expected="${f%.txt}.out"
  if diff -q <($BIN "$f" 2>/dev/null) "$expected" > /dev/null; then
    echo "ok   $(basename $f)"
  else
    echo "FAIL $(basename $f)"
    fail=1
  fi
done
exit $fail
