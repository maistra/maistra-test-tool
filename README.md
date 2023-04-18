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

1. An `oc` client can be downloaded from [mirror openshift-v4 clients](https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/). Extract the `openshift-client-...tar.gz` file and move both `oc` and `kubectl` binaries into a local PATH directory.

2. Access an OpenShift cluster from command line before running tests. Run a login command. `oc login -u [user] -p [token] --server=[OCP API server]`

3. RedHat Service Mesh Operator has been installed on the OpenShift cluster.

## Testing
- In order to save results in an XML report, we can run a go test command with "github.com/jstemmer/go-junit-report".
    ```
    $ go install github.com/jstemmer/go-junit-report/v2@latest
    ```

- Optionally to run all the test cases customizing the SMCP namespace and the SMCP name: A user can update the expected values in the `tests/test.env`.

- By default, the `tests/test.env` file uses `export SAMPLEARCH=x86`
    - For Power environment testing, a user can update the `tests/test.env` file `export SAMPLEARCH=p`
    - For Z environment testing, a user can update the `tests/test.env` file `export SAMPLEARCH=z`

- To run all the test cases, run the following command:
  ```
  make test
  ```

- To run a single test case, run `make test <name>`. For example:
  ```
  make test TestFaultInjection
  ```

## License

[Maistra OpenShift Test Tool](https://github.com/maistra/maistra-test-tool) is [Apache 2.0 licensed](https://github.com/maistra/maistra-test-tool/blob/development/LICENSE)
