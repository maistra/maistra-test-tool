# Maistra OpenShift Test Tool

[![](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat)](https://github.com/maistra/maistra-test-tool/blob/development/LICENSE)
![](https://img.shields.io/github/repo-size/maistra/maistra-test-tool.svg?style=flat)
[![](https://goreportcard.com/badge/github.com/maistra/maistra-test-tool)](https://goreportcard.com/report/github.com/maistra/maistra-test-tool)


A testing tool for running Maistra Service Mesh tasks on an OpenShift 4.x cluster.

## Introduction

This project aims to automate the running Maistra Service Mesh tasks on an OpenShift 4.x Cluster.

The testing follows [Istio Doc Tasks](https://istio.io/v1.6/docs/tasks/).

The test cases include several changes for an OpenShift environment. Currently, those changes will not work in origin Kubernetes environments.


## Versions

| Name      | Version       |
| --        | --            |
| OS        | Linux         |
| Golang    | 1.13+         |
| Python    | 3.7+          |

The host machine which triggers test scripts must have Golang and Python installed before running go tests.

## Testing Prerequisite

1. User can access a running OpenShift cluster from command line.
2. Service Mesh Control Plane (SMCP) has been installed on an OpenShift cluster. The SMCP is in namespace `istio-system` and the SMCP name is `basic`
3. An `oc` client has been installed. User has completed CLI login an OCP cluster as an admin user. Run `oc login -u [user] -p [token] --server=[OCP API server]`
4. Several test cases require nginx or mongoDB running on OCP4 and we need to configure additional scc permission for them after login as a cluster admin user.
   ```
   $ oc login -u kubeadmin -p [token] --server=[OCP API server]
   $ scripts/setup_ocp_scc_anyuid.sh
   ```

## Testing
- All test cases and testdata are in the `tests` directory.
- To run all the test cases: `go test -timeout 3h -v`. It is required to use the `-timeout` flag. Otherwise, the go test by default will fall into panic after 10 minutes.
- In order to save results in a XML report, we can run a go test command with "github.com/jstemmer/go-junit-report", e.g.
    ```
    $ cd tests
    $ export GODEBUG=x509ignoreCN=0
    $ go get -u github.com/jstemmer/go-junit-report
    $ go test -timeout 3h -v 2>&1 | tee >(${GOPATH}/bin/go-junit-report > results.xml) test.log
    ```
- All case numbers are mapped in the `test_cases.go` file. Users can run a single test with the `-run [case number]` flag, e.g. `go test -run 17 -timeout 1h -v`.
- The testdata `samples` and `samples_extend` are pulling from [Istio 1.6.5](https://github.com/istio/istio/releases/tag/1.6.5) and [Istio 1.6 Doc](https://archive.istio.io/v1.6/docs/tasks/).


## License

[Maistra OpenShift Test Tool](https://github.com/maistra/maistra-test-tool) is [Apache 2.0 licensed](https://github.com/maistra/maistra-test-tool/blob/development/LICENSE)
