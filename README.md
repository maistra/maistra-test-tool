# Maistra OpenShift Test Tool

![](https://img.shields.io/github/repo-size/maistra/maistra-test-tool.svg?style=flat)
[![](https://goreportcard.com/badge/github.com/maistra/maistra-test-tool)](https://goreportcard.com/report/github.com/maistra/maistra-test-tool)


A testing tool for running Maistra Service Mesh tasks on an OpenShift 4.x cluster.

## Introduction

This project aims to automate Maistra Service Mesh tasks on an OpenShift 4.x Cluster.

The testing tasks are based on [istio.io Doc Tasks](https://istio.io/v1.9/docs/tasks/)

## Versions

| Name      | Version       |
| --        | --            |
| OS        | Linux         |
| Golang    | 1.13+         |
| OpenSSl   | 1.1.1+        |
| oc client | 4.x           |

## Testing Prerequisite

1. An OpenShift `oc` client has been installed. A user can access an OpenShift cluster from command line by running a login command. `oc login -u [user] -p [token] --server=[OCP API server]`
2. RedHat Service Mesh Operators and Control Plane (SMCP) have been installed on the OpenShift cluster.
3. OpenSSL tool has been installed from the client command line.

## Testing
- A main test is in the `tests` directory. All test cases are in the `test_cases.go` and are mapped to the implementations in the `pkg` directory.

- In order to save results in a XML report, we can run a go test command with "github.com/jstemmer/go-junit-report".
    ```
    $ go get -u github.com/jstemmer/go-junit-report
    ```

- Optionally to run all the test cases customizing the SMCP namespace and the SMCP name: A user can update the expected values in the `tests/test.env`.

- By default, the `tests/test.env` file uses `export SAMPLEARCH=x86`
    - For Power environment testing, a user can update the `tests/test.env` file `export SAMPLEARCH=p`
    - For Z environment testing, a user can update the `tests/test.env` file `export SAMPLEARCH=z`

- To run all the test cases: `cd tests; go test -timeout 3h -v`

    ```
    $ cd tests
    $ go test -timeout 3h -v 2>&1 | tee >(${GOPATH}/bin/go-junit-report > results.xml) test.log
    ```
- To run a single test case: e.g. `cd tests; go test -run S1 -timeout 3h -v`

    Test cases shortname and mapping are in the `tests/test_cases.go` file.
    `A...` are shortname of smoke tests.
    `T...` are shortname of tasks and functional test cases.


## License

[Maistra OpenShift Test Tool](https://github.com/maistra/maistra-test-tool) is [Apache 2.0 licensed](https://github.com/maistra/maistra-test-tool/blob/development/LICENSE)
