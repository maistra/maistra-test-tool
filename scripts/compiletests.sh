#!/bin/bash

# Find all directories that contain test files
directories=$(find . -name "*_test.go" | xargs -n1 dirname | sort -u)

# Compile test binaries for each directory
for dir in $directories
do
  echo "Compiling tests in $dir ..."
  go test -c $dir
done
