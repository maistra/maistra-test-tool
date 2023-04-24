#!/bin/bash

set -e

SUPPORTED_VERSIONS=("v2.2" "v2.3" "v2.4")

log() {
    echo "$*" | tee -a "$LOG_FILE"
}

logHeader() {
    log
    log "====== $*"
    log
}

require_SMCP_VERSION() {
    if [ -z "$SMCP_VERSION" ]; then
        echo "ERROR: must specify version in SMCP_VERSION env var"
        exit 1
    fi
}

runAllTests() {
    require_SMCP_VERSION

    local dir="$1"

    echo > "$LOG_FILE"
    if [ -n "$TEST_GROUP" ]; then
        logHeader "Executing tests in group '$TEST_GROUP' against SMCP $SMCP_VERSION"
    else
        logHeader "Executing all tests against SMCP $SMCP_VERSION"
    fi

    gotestsum -f standard-verbose --packages "$dir/..." \
    --max-fails 10 \
    --rerun-fails=5 --rerun-fails-max-failures 10 --rerun-fails-run-root-test --rerun-fails-report "$RERUNS_FILE" \
    --junitfile "$REPORT_FILE" --junitfile-project-name "maistra-test-tool-$SMCP_VERSION" --junitfile-hide-empty-pkg \
    --junitfile-testsuite-name relative --junitfile-testcase-classname relative \
    -- -timeout 1h -count 1 -p 1 2>&1 \
    | tee -a "$LOG_FILE"
}

runTest() {
    require_SMCP_VERSION

    local dir="$1"
    local testName="$2"

    echo > "$LOG_FILE"
    logHeader "Executing $testName against SMCP $SMCP_VERSION"

    gotestsum -f standard-verbose --packages "$dir/" \
    --rerun-fails=5 --rerun-fails-run-root-test --rerun-fails-report "$RERUNS_FILE" \
    --junitfile "$REPORT_FILE" --junitfile-project-name "maistra-test-tool-$SMCP_VERSION" --junitfile-hide-empty-pkg \
    --junitfile-testsuite-name relative --junitfile-testcase-classname relative \
    -- -timeout 30m -count 1 -p 1 -run "^$testName$" 2>&1 \
    | tee -a "$LOG_FILE"
}

resetCluster() {
    echo
    echo "Resetting cluster by deleting namespaces used in the test suite"
    oc delete namespace istio-system bookinfo foo bar legacy mesh-external cert-manager --ignore-not-found
}

main() {
    if [ -z "$SMCP_VERSION" ]; then
        echo
        echo "Executing tests against all supported versions: ${SUPPORTED_VERSIONS[*]}"
        echo "    Note: To run tests against a specific version, set the SMCP_VERSION environment variable."
    else
        echo "Executing tests against SMCP version $SMCP_VERSION"
    fi
    export OUTPUT_DIR="$PWD/tests/result-$(date +%Y%m%d%H%M%S)"
    echo "Output dir: $OUTPUT_DIR"
    mkdir -p "$OUTPUT_DIR"

    if [ -n "${OCP_CRED_PSW}" ]; then
        oc login -u ${OCP_CRED_USR} -p ${OCP_CRED_PSW} --server=${OCP_API_URL} --insecure-skip-tls-verify=true
    elif [ -n "${OCP_TOKEN}" ]; then
        oc login --token=${OCP_TOKEN} --server=${OCP_API_URL} --insecure-skip-tls-verify=true
    fi

    testName="${TEST_CASE:-$1}"
    if [ -n "$testName" ]; then
        # find the directory containing the specified test
        dir=""
        file=""
        for f in $(find . -type f -name "*_test.go"); do
            if grep -q "func $testName(" "$f"; then
                if [ -z "$dir" ]; then
                    dir=$(dirname "$f")
                    file="$f"
                else
                    echo >&2 "ERROR: Multiple tests with the given name were found. Please ensure test names are unique!"
                    exit 1
                fi
            fi
        done

        if [ -z "$dir" ]; then
            echo >&2 "ERROR: Could not find test $testName"
            exit 1
        fi
        echo "Found $testName in file $file."
    fi

    resetCluster

    declare -a versions=()
    declare -A logFiles
    declare -A reportFiles

    if [ -z "$SMCP_VERSION" ]; then
        for ver in ${SUPPORTED_VERSIONS[@]}; do
            export SMCP_VERSION="$ver"
            export LOG_FILE="$OUTPUT_DIR/output_${SMCP_VERSION}.log"
            export REPORT_FILE="$OUTPUT_DIR/report_${SMCP_VERSION}.xml"
            export RERUNS_FILE="$OUTPUT_DIR/reruns_${SMCP_VERSION}.txt"

            versions+=("$SMCP_VERSION")
            logFiles["$SMCP_VERSION"]="$LOG_FILE"
            reportFiles["$SMCP_VERSION"]="$REPORT_FILE"

            if [ -z "$testName" ]; then
                runAllTests "$PWD/pkg/tests"
            else
                runTest "$dir" "$testName"
            fi
            resetCluster
        done
    else
        SMCP_VERSION="v${SMCP_VERSION#v}" # prepend "v" if necessary
        export LOG_FILE="$OUTPUT_DIR/output_${SMCP_VERSION}.log"
        export REPORT_FILE="$OUTPUT_DIR/report_${SMCP_VERSION}.xml"
        export RERUNS_FILE="$OUTPUT_DIR/reruns_${SMCP_VERSION}.txt"

        versions+=("$SMCP_VERSION")
        logFiles["$SMCP_VERSION"]="$LOG_FILE"
        reportFiles["$SMCP_VERSION"]="$REPORT_FILE"

        if [ -z "$testName" ]; then
            runAllTests "$PWD/pkg/tests"
        else
            runTest "$dir" "$testName"
        fi
    fi

    echo
    echo "====== JUnit report file(s)"
    for ver in "${versions[@]}"; do
        echo "$ver: ${reportFiles[$ver]}"
    done

    echo
    echo "====== Test summary"
    for ver in "${versions[@]}"; do
        echo "${ver}: $(tail -1 ${logFiles[$ver]})"
    done
}

time main $@
