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
* Completed login the OCP cluster.


## Testing

- To run all the test cases (End-to-End run): `go test -timeout 3h -v`

## License

[Maistra OpenShift Istio Test Tool](https://github.com/Maistra/istio-test-tool) is [Apache 2.0 licensed](https://github.com/Maistra/istio-test-tool/blob/master/LICENSE)
