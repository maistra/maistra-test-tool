# moitt

[![](https://img.shields.io/github/watchers/yxun/moitt.svg?style=flat)](https://github.com/yxun/moitt/watchers)
[![](https://img.shields.io/github/stars/yxun/moitt.svg?style=flat)](https://github.com/yxun/moitt/stargazers)
[![](https://img.shields.io/github/forks/yxun/moitt.svg?style=flat)](https://github.com/yxun/moitt/network/members)
[![](https://img.shields.io/github/issues-pr-closed-raw/yxun/moitt.svg?style=flat)](https://github.com/yxun/moitt/issues)
[![Go Report Card](https://goreportcard.com/badge/github.com/yxun/moitt/test)](https://goreportcard.com/report/github.com/yxun/moitt)
[![](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat)](https://github.com/yxun/moitt/blob/master/LICENSE)
![](https://img.shields.io/github/repo-size/yxun/moitt.svg?style=flat)


moitt stands for Maistra OpenShift Istio Test Tool

A Testing Tool For Running Istio Doc Tasks on OpenShift

## Introduction

This project aims to automate installation and testing of  Maistra Istio system on an AWS OpenShift Cluster.

The installation follows [OpenShift Installer](https://github.com/openshift/installer) and [Maistra istio-operator](https://github.com/Maistra/istio-operator). 

The testing follows [Istio Doc Tasks](https://istio.io/docs/tasks/).


## Versions

OS Version: Fedora 28 or above

Go Version: go 1.11 or above

**Note**: If you clone this project inside of the `$GOPATH/src` tree, invoke the `go` command with `GO111MODULE=on` environment variable set.  https://github.com/golang/go/wiki/Modules#how-to-use-modules


Python Version: Python 3.7 or above


## Installation

### 1. Prepare 

* Prepare aws configuration files or configure them from `awscli`
* Install language runtime and tools. Run `scripts/setup_install.sh`
* Save OpenShift Pull Secret content and we need to paste all the content in one line later
* Download your Istio private registry pull secret and create a file called "`secret.yaml`"
* Confirm a shell has been started by pipenv. Otherwise, go to "`install`" directory and run "`pipenv install; pipenv shell`"


### 2. Environment Variables

| Name        | Description |
| ----------- | ----------- |
| AWS_PROFILE | AWS profile name |
| PULL_SEC    | Istio private registry pull secret.yaml file path |
| OC_VERSION | Maistra origin istiooc_linux version (e.g. 0.10.0) |
| OPERATOR_FILE | Maistra Istio operator.yaml file path |
| CR_FILE     | Istio ControlPlane CR file path  |
| JAEGER_OPERATOR_VERSION | Jaeger Operator version (e.g. v1.12.1) |
| KIALI_OPERATOR_VERSION | Kiali Operator version (e.g. v1.0.0) |

* Export the environment variables (See the table above) with their values


### 3. OCP/AWS
* Go to directory "`install`"
* Run "`python main.py -h`" and follow arguments help message. e.g. "`python main.py -i -c ocp`" will install an OCP cluster on AWS 
* After `Deploying the cluster...` starts, follow the prompts
  * Select a SSH public key
  * Select Platform > aws
  * Select a Region
  * Select a Base Domain
  * Create a Cluster Name
  * Paste the Pull Secret content ( This Pull Secret content is different from the environment variable `PULL_SEC` )
* Waiting for the cluster creation completes. It usually takes 40 - 50 minutes.
* After the cluster creation, this script automatically downloads a Maistra origin oc client and moves it to `/usr/bin/`. This script also automatically creates a kubectl soft link using `sudo ln -s oc /usr/bin/kubectl`

    When OCP installation compeleted, you should see INFO message "Install complete!" and following messages which includes export KUBECONFIG value and oc login credential.

### 4. Login the OCP cluster
* After OCP cluster deployment, follow the INFO message and execute the following two commands manually:
  * Run `export KUBECONFIG=[kubeconfig file]`
  * Run `oc login -u [login user] -p [token]`


### 5. (Optional) [registry-puller](https://github.com/knrc/registry-puller)
* If you need to pull images from a private registry, install this registry-puller tool on an OCP cluster first. 
* Go to directory "`install`"
* Run "`python main.py -h`" and follow arguments help message. e.g. "`python main.py -i -c registry-puller`" will deploy the registry-puller pod in registry-puller namespace on OCP


### 6. Maistra/Istio
* Go to directory "`install`"
* Run "`python main -h`" and follow arguments help message. e.g. "`python main.py -i -c istio`" will follow [Maistra istio-operator](https://github.com/Maistra/istio-operator) and install the Jaeger Operator, Kiali Operator, Istio Operator and Istio system on OCP
* Waiting for the Istio system installation completes. It usually takes 10 - 15 minutes

    When Istio system installation completed, you should see message "Installed=True, reason=InstallSuccessful".


## Testing Prerequisite

* Istio system has been installed on an OpenShift cluster

* Login the OCP cluster 


## Testing

Go to directory "`test/maistra`" 
- To run all the test cases (End-to-End run): `go test -timeout 2h -v`
- To run a specific test case: `go test -run [test case number, e.g. 03] -v`
    
    Note: tc_14_authentication_test execution time is more than 10 minutes. If you only want to run tc_14, use -timeout 20m: `go test -run 14 -timeout 20m -v` 



## License

moitt is [Apache 2.0 licensed](https://github.com/yxun/moitt/blob/master/LICENSE)
