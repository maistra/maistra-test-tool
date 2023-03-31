#!/bin/bash

runAllTests() {
    echo "Executing all tests:"
    echo

    go test ./pkg/tests/... -failfast -timeout 2h -v -count 1 -p 1 2>&1 | tee >(${GOPATH}/bin/go-junit-report > tests/results.xml) tests/test.log
}

runTest() {
    local testName="$1"
    local dir=""
    for file in $(find . -type f -name "*_test.go"); do
        if grep -q "func $testName(" "$file"; then
            if [ -z "$dir" ]; then
                dir=$(dirname "$file")
            else
                echo "ERROR: Multiple tests with the given name were found. Please ensure test names are unique!"
                exit 1
            fi
        fi
    done

    if [ -z "$dir" ]; then
        echo "ERROR: Could not find test $testName"
        exit 1
    fi

    echo "Found $testName in file $file."
    echo "Executing test:"
    echo

    go test -timeout 30m -v -count 1 -run "$testName" "$dir/" 2>&1 | tee >(${GOPATH}/bin/go-junit-report > tests/results.xml) tests/test.log
}

testName="$1"
if [ -z "$testName" ]; then
    runAllTests
else
    runTest $testName
fi

