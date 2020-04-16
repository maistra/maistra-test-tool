# Maistra OpenShift Istio Test Tool

[![](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat)](https://github.com/Maistra/istio-test-tool/blob/master/LICENSE)
![](https://img.shields.io/github/repo-size/Maistra/istio-test-tool.svg?style=flat)
[![](https://goreportcard.com/badge/github.com/Maistra/istio-test-tool)](https://goreportcard.com/report/github.com/Maistra/istio-test-tool)


A testing tool for running Istio Doc tasks on AWS OpenShift 4.x cluster. 

## Introduction

This project aims to automate the running Maistra Istio Doc tasks on an AWS OpenShift 4.x Cluster.

The testing follows [Istio Doc Tasks](https://istio.io/docs/tasks/) and [Maistra Doc](https://maistra-1-1.maistra.io/).


## Versions

| Name      | Version       |
| --        | --            |
| OS        | Fedora 28+    |
| Golang    | 1.13+         |


## Testing Prerequisite

* Maistra istio system has been installed on an OpenShift OCP4 cluster.
* Completed CLI login an OCP cluster. Run `oc login -u [user] -p [token] --server=[OCP API server]` login command in a shell.


## Testing
- The test cases include several changes for an OpenShift environment. Currently, those changes will not work in origin Kubernetes environments.
- To run all the test cases: `go test -timeout 3h -v`. It is required to use the `-timeout` flag. Otherwise, the go test by default will fall into panic after 10 minutes.
- In order to save results in a junit report, we can run a go test command with "github.com/jstemmer/go-junit-report", e.g.
    ```
    $ go get -u github.com/jstemmer/go-junit-report
    $ go test -timeout 3h -v 2>&1 | tee >(${GOPATH}/bin/go-junit-report > results.xml) test.log
    ```
- All case numbers are mapped in the `test_cases.go` file. Users can run a single test with the `-run [case number]` flag, e.g. `go -test -run 15 -timeout 1h -v`.
- The testdata `samples` and `samples_extend` are pulling from [Istio 1.4.6](https://github.com/istio/istio/releases/tag/1.4.6) and [Istio 1.4 Doc](https://archive.istio.io/v1.4/docs/tasks/).


## License

[Maistra OpenShift Istio Test Tool](https://github.com/Maistra/istio-test-tool) is [Apache 2.0 licensed](https://github.com/Maistra/istio-test-tool/blob/master/LICENSE)
