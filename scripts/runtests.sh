#!/bin/bash

set -e

OPERATOR_VERSION=${OPERATOR_VERSION:-"2.6.0"}
OPERATOR_VERSION="${OPERATOR_VERSION#v}" # remove "v" prefix if necessary

echo "OSSM Operator version is $OPERATOR_VERSION"

case "$OPERATOR_VERSION" in
    2.3.*) SUPPORTED_VERSIONS=("v2.1" "v2.2" "v2.3") ;;
    2.4.*) SUPPORTED_VERSIONS=("v2.2" "v2.3" "v2.4") ;;
    2.5.*) SUPPORTED_VERSIONS=("v2.3" "v2.4" "v2.5") ;;
    2.6.*) SUPPORTED_VERSIONS=("v2.4" "v2.5" "v2.6") ;;
    *) echo "ERROR: unknown operator version: $OPERATOR_VERSION; expect either 2.3.x, 2.4.x, 2.5.x or 2.6.x"; exit 1 ;;
esac

log() {
    echo "$*" | tee -a "$LOG_FILE"
}

logHeader() {
    log
    log "====== $*"
    log
}

runTestsAgainstVersion() {
    if [ -z "$SMCP_VERSION" ]; then
        echo "ERROR: must specify version in SMCP_VERSION env var"
        exit 1
    fi
    
    unset EXPECTED_VERSION

    if [ -n "$EXPECTED_VERSIONS" ]; then 
        for version in "${EXPECTED_VERSIONS[@]}"; do
            if [[ "$version" == "$SMCP_VERSION"* ]]; then
                export EXPECTED_VERSION="$version"
                break
            fi
        done
    fi

    echo "Output dir: $OUTPUT_DIR"
    mkdir -p "$OUTPUT_DIR"

    echo > "$LOG_FILE"

    if [ -z "$TEST_CASE" ]; then
        if [ -n "$TEST_GROUP" ]; then
            logHeader "Executing tests in group '$TEST_GROUP' against SMCP $SMCP_VERSION"
            if [ "$TEST_GROUP" = "disconnected" ]; then
                if [ -z "$BASTION_HOST" ]; then
                    echo "BASTION_HOST=$BASTION_HOST"
                    echo "ERROR: must specify BASTION_HOST env var when running disconnected tests"
                    exit 1
                fi
                log "NOTE: The script will modify the host of the image to be deployed in the ../images.yaml file"
                log "      Please make sure the image is accessible from the disconnected environment doing the correct mirroring"
                log "      and the host is correct"
                sed -i "s|quay.io/maistra/examples|${BASTION_HOST}:55555/maistra/examples|g" images.yaml
                sed -i "s|quay.io/openshifttest/|${BASTION_HOST}:55555/openshifttest/|g" images.yaml
            fi
        else
            logHeader "Executing all tests against SMCP $SMCP_VERSION"
        fi

        gotestsum -f standard-verbose --packages "$TEST_DIR" \
        --max-fails 10 \
        --rerun-fails=2 --rerun-fails-max-failures 10 --rerun-fails-run-root-test --rerun-fails-report "$RERUNS_FILE" \
        --junitfile "$REPORT_FILE" --junitfile-project-name "maistra-test-tool-$SMCP_VERSION" --junitfile-hide-empty-pkg \
        --junitfile-testsuite-name relative --junitfile-testcase-classname relative \
        -- -timeout 1h -count 1 -p 1 2>&1 \
        | tee -a "$LOG_FILE"
    else
        logHeader "Executing $TEST_CASE against SMCP $SMCP_VERSION"

        gotestsum -f standard-verbose --packages "$TEST_DIR" \
        --rerun-fails=2 --rerun-fails-run-root-test --rerun-fails-report "$RERUNS_FILE" \
        --junitfile "$REPORT_FILE" --junitfile-project-name "maistra-test-tool-$SMCP_VERSION" --junitfile-hide-empty-pkg \
        --junitfile-testsuite-name relative --junitfile-testcase-classname relative \
        -- -timeout 30m -count 1 -p 1 -run "^$TEST_CASE$" 2>&1 \
        | tee -a "$LOG_FILE"
    fi

    # prepend SMCP version to testcase names in JUnit XML
    sed -i -E "s~<testcase .* name=\"~\0${SMCP_VERSION}/~g" "$REPORT_FILE"

    # extract skipped tests into skipped.log
    sed -En '/=== Skipped/,/=== Failed|DONE/ { /=== Failed|DONE/!p }' "$LOG_FILE" > "$OUTPUT_DIR/skipped.log"

    # extract failed tests into failed.log
    sed -En '/=== Failed/,/DONE/ { /DONE/!p }' "$LOG_FILE" > "$OUTPUT_DIR/failed.log"
}

resetCluster() {
    echo
    echo "Resetting cluster by deleting namespaces used in the test suite"
    oc delete namespace istio-system bookinfo foo bar legacy mesh-external cert-manager --ignore-not-found
    echo
}

writeDocumentation() {
    echo "Note: This file contains all the test cases executed in this test Run" >> $DOCUMENTATION_FILE

    while read line; do
        if [[ $line == *"RUN"* ]]; then
            test_name=$(echo $line | awk '{print $3}')
            if [[ $test_name =~ "/" ]]; then
                echo " " >> $DOCUMENTATION_FILE
                echo "SUB TEST: $test_name" >> $DOCUMENTATION_FILE
            else
                echo " " >> $DOCUMENTATION_FILE
                echo "TEST CASE: $test_name" >> $DOCUMENTATION_FILE
            fi
        fi
        if [[ $line == *"STEP"* ]]; then
            step_desc=$(echo $line | awk '{for(i=3;i<=NF;++i) printf "%s ", $i; print ""}')
            echo "Step $step_desc" >> $DOCUMENTATION_FILE
        fi
        if [[ $line == *"Skipping test"* ]]; then
            echo "This Test cases is Skipped for this run" >> $DOCUMENTATION_FILE
        fi
    done < $LOG_FILE | awk '!seen[$0]++'
}

main() {
    if [ -n "$1" ]; then
        export TEST_CASE="$1"
    fi

    if [ -z "$SMCP_VERSION" ]; then
        echo
        echo "Executing tests against all supported versions: ${SUPPORTED_VERSIONS[*]}"
        echo "    Note: To run tests against a specific version, set the SMCP_VERSION environment variable."
    else
        echo "Executing tests against SMCP version $SMCP_VERSION"
    fi

    # make sure the token takes precedence over usr/pass
    if [ -n "${OCP_TOKEN}" ]; then
        oc login --token=${OCP_TOKEN} --server=${OCP_API_URL} --insecure-skip-tls-verify=true
    elif [ -n "${OCP_CRED_PSW}" ]; then
        oc login -u ${OCP_CRED_USR} -p ${OCP_CRED_PSW} --server=${OCP_API_URL} --insecure-skip-tls-verify=true
    fi

    export TEST_DIR=""
    if [ -n "$TEST_CASE" ]; then
        # find the directory containing the specified test
        TEST_DIR=""
        file=""
        for f in $(find . -type f -name "*_test.go"); do
            if grep -q "func $TEST_CASE(" "$f"; then
                if [ -z "$TEST_DIR" ]; then
                    TEST_DIR=$(dirname "$f")
                    file="$f"
                else
                    echo >&2 "ERROR: Multiple tests with the given name were found. Please ensure test names are unique!"
                    exit 1
                fi
            fi
        done

        if [ -z "$TEST_DIR" ]; then
            echo >&2 "ERROR: Could not find test $TEST_CASE"
            exit 1
        fi
        TEST_DIR="$TEST_DIR/"
        echo "Found $TEST_CASE in file $file"
    else
        TEST_DIR="$PWD/pkg/tests/..."
    fi

    resetCluster

    declare -a versions=()
    declare -a logFiles=()
    declare -a reportFiles=()

    if [ -z "$SMCP_VERSION" ]; then
        for ver in "${SUPPORTED_VERSIONS[@]}"; do
            export OPERATOR_VERSION="$OPERATOR_VERSION"
            export SMCP_VERSION="$ver"
            export OUTPUT_DIR="${OUTPUT_DIR_BASE}/${SMCP_VERSION}"  # also used in env.GetOutputDir(), so must be exported
            export LOG_FILE="$OUTPUT_DIR/output.log"
            export DOCUMENTATION_FILE="$OUTPUT_DIR/documentation.txt"
            export REPORT_FILE="$OUTPUT_DIR/report.xml"
            export RERUNS_FILE="$OUTPUT_DIR/reruns.txt"

            versions+=("$SMCP_VERSION")
            logFiles+=("$LOG_FILE")
            reportFiles+=("$REPORT_FILE")

            runTestsAgainstVersion
            resetCluster
            writeDocumentation
        done
    else
        SMCP_VERSION="v${SMCP_VERSION#v}" # prepend "v" if necessary
        export OUTPUT_DIR="${OUTPUT_DIR_BASE}/${SMCP_VERSION}"  # also used in env.GetOutputDir(), so must be exported
        export LOG_FILE="$OUTPUT_DIR/output.log"
        export DOCUMENTATION_FILE="$OUTPUT_DIR/documentation.txt"
        export REPORT_FILE="$OUTPUT_DIR/report.xml"
        export RERUNS_FILE="$OUTPUT_DIR/reruns.txt"

        versions+=("$SMCP_VERSION")
        logFiles+=("$LOG_FILE")
        reportFiles+=("$REPORT_FILE")

        runTestsAgainstVersion
        writeDocumentation
    fi

    echo
    echo "====== JUnit report file(s)"
    for (( i=0; i<${#versions[@]}; i++ )); do
        echo "${versions[$i]}: ${reportFiles[$i]}"
    done

    echo
    echo "====== Test summary"
    for (( i=0; i<${#versions[@]}; i++ )); do
        tail -10 ${logFiles[$i]} | tac | sed -n -e "0,/DONE/{s/^/${versions[$i]}: /p}" | tac
    done
}

timedMain() {
    time main $@
}


export OUTPUT_DIR_BASE="$PWD/tests/result-$(date +%Y%m%d%H%M%S)"
echo "Output dir: $OUTPUT_DIR_BASE"
mkdir -p "$OUTPUT_DIR_BASE"
ln -sfn "$OUTPUT_DIR_BASE" "$PWD/tests/result-latest"

timedMain $@ 2>&1 | tee "${OUTPUT_DIR_BASE}/output.log"
