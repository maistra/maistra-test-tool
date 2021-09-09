# Maistra OpenShift Test Tool

![](https://img.shields.io/github/repo-size/maistra/maistra-test-tool.svg?style=flat)
[![](https://goreportcard.com/badge/github.com/maistra/maistra-test-tool)](https://goreportcard.com/report/github.com/maistra/maistra-test-tool)


A testing tool for running Maistra Service Mesh tasks on an OpenShift 4.x cluster.

## Introduction

This project aims to automate the running Maistra Service Mesh tasks on an OpenShift 4.x Cluster.

The testing follows [Istio Doc Tasks](https://istio.io/v1.9/docs/tasks/)

The test cases include several changes for an OpenShift environment. Currently, those changes will not work in origin Kubernetes environments.

## Versions

| Name      | Version       |
| --        | --            |
| OS        | Linux         |
| Golang    | 1.13+         |

## Testing Prerequisite

1. User can access a running OpenShift cluster from command line.
2. Service Mesh Control Plane (SMCP) has been installed on an OpenShift cluster. The SMCP is in namespace `istio-system` and the SMCP name is `basic`
3. An `oc` client has been installed. User has completed CLI login an OCP cluster as an admin user. Run `oc login -u [user] -p [token] --server=[OCP API server]`

## Testing
- A main test is in the `tests` directory. All test cases are in the `test_cases.go` and are mapped to the implementations in the `pkg` directory.

- In order to save results in a XML report, we can run a go test command with "github.com/jstemmer/go-junit-report".
    ```
    $ go get -u github.com/jstemmer/go-junit-report
    ```

- By default, there is an environment variable `SAMPLEARCH=x86`
    - For Power environment testing, users can export an environment variable `SAMPLEARCH`
        ```
        $ export SAMPLEARCH=p
        ```
    - For Z environment testing, users can export an environment variable `SAMPLEARCH`
        ```
        $ export SAMPLEARCH=z
        ```

- To run all the test cases: `cd tests; go test -timeout 3h -v`. It is required to use the `-timeout` flag. Otherwise, the go test will fall into panic after 10 minutes.
    ```
    $ cd tests
    $ go test -timeout 3h -v 2>&1 | tee >(${GOPATH}/bin/go-junit-report > results.xml) test.log
    ```

## License

[Maistra OpenShift Test Tool](https://github.com/maistra/maistra-test-tool) is [Apache 2.0 licensed](https://github.com/maistra/maistra-test-tool/blob/development/LICENSE)
