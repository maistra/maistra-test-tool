# Maistra OpenShift Test Tool

![](https://img.shields.io/github/repo-size/maistra/maistra-test-tool.svg?style=flat)
[![](https://goreportcard.com/badge/github.com/maistra/maistra-test-tool)](https://goreportcard.com/report/github.com/maistra/maistra-test-tool)


A testing tool for running Maistra Service Mesh tasks on an OpenShift 4.x cluster.

## Introduction

This project aims to automate the running Maistra Service Mesh tasks on an OpenShift 4.x Cluster.

The testing follows [Istio Doc Tasks](https://istio.io/v1.9/docs/tasks/)


## Versions

| Name      | Version       |
| --        | --            |
| OS        | Linux         |
| Golang    | 1.13+         |
| Python    | 3.7+          |
| python3-pip |             |


The host machine which triggers test scripts must have Golang and Python installed before running go tests.

python3-pip is required for testing Security_authentication_end-user_JWT

## Testing Prerequisite

1. Service Mesh Control Plane has been installed on an OpenShift OCP4 cluster.
2. An `oc` client has been installed.  Completed CLI login an OCP cluster as a regular user. Run `oc login -u [user] -p [token] --server=[OCP API server]`
3. For the test case using JWT token, the system needs python3 installed to be able to run the python script.

## Testing
- All test cases and testdata are in the `tests` directory.
- The test cases include several changes for an OpenShift environment. Currently, those changes will not work in origin Kubernetes environments.
- To run all the test cases: `go test -timeout 3h -v`. It is required to use the `-timeout` flag. Otherwise, the go test by default will fall into panic after 10 minutes.
- In order to save results in a junit report, we can run a go test command with "github.com/jstemmer/go-junit-report".
    ```
    $ go get -u github.com/jstemmer/go-junit-report
    ```
- By default, an environment variable `SAMPLEARCH=x86`

- For Power environment testing, user can export an environment variable `SAMPLEARCH`
    ```
    $ export SAMPLEARCH=p
    ```

- For Z environment testing, user can export an environment variable `SAMPLEARCH`
    ```
    $ export SAMPLEARCH=z
    ```

    ```
    $ cd tests
    $ go test -timeout 3h -v 2>&1 | tee >(${GOPATH}/bin/go-junit-report > results.xml) test.log
    ```

## License

[Maistra OpenShift Test Tool](https://github.com/maistra/maistra-test-tool) is [Apache 2.0 licensed](https://github.com/maistra/maistra-test-tool/blob/development/LICENSE)
