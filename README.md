# moitt

moitt stands for Maistra OpenShift Istio Test Tool

A Testing Tool For Running Istio Doc Tasks on OpenShift

Introduction
=========================

This project aims to follow [Istio Doc Tasks](https://istio.io/docs/tasks/) structure and organize upstream [Istio release-1.1 tests directory](https://github.com/istio/istio/tree/release-1.1/tests). All test cases can be run against an Istio system running on an OpenShift cluster on AWS.

Most of the utilities are imported from upstream "istio.io/istio/tests/util". For more utilities information, user can check upstream [Istio release-1.1 util directory](https://github.com/istio/istio/tree/release-1.1/tests/util).


Versions
-----------------

OS Version: Fedora 28 or above

Go Version: go 1.10.7 or above

Python Version: Python 3.7 or above


Installation
=====================

1. Prepare 

* Prepare aws configuration files or configure them from `awscli`
* Install language runtime and tools. Run `scripts/setup.sh`
* Save OpenShift Pull Secret content and we need to paste all the content in one line later
* Download your Istio private registry pull secret and create a file called "`secret.yaml`"
* Confirm a shell has been started by pipenv. Otherwise, go to "`install`" directory and run "`pipenv install; pipenv shell`"


2. Environment Variables
---------------------

| Name        | Description |
| ----------- | ----------- |
| AWS_PROFILE | AWS profile name |
| PULL_SEC    | Istio private registry pull secret.yaml file path |
| OPERATOR_FILE | Maistra Istio operator.yaml file path |
| CR_FILE     | Istio ControlPlane CR file path  |

* Export the environment variables (See the table above) with their values


3. OCP/AWS
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
* After the cluster creation, this script automatically downloads an oc client and moves it to `/usr/bin/`. This script also automatically creates a kubectl soft link using `sudo ln -s oc /usr/bin/kubectl`


4. Login the OCP cluster
* After OCP cluster deployment, follow the INFO message and execute the following two commands manually:
  * Run `export KUBECONFIG=[kubeconfig file]`
  * Run `oc login -u [login user] -p [token]`


5. [registry-puller](https://github.com/knrc/registry-puller)
* Go to directory "`install`"
* Run "`python main.py -h`" and follow arguments help message. e.g. "`python main.py -i -c registry-puller`" will deploy the registry-puller pod in registry-puller namespace on OCP


6. Maistra/Istio
* Go to directory "`install`"
* Run "`python main -h`" and follow arguments help message. e.g. "`python main.py -i -c istio`" will install [Maistra istio-operator](https://github.com/Maistra/istio-operator) and Istio system on OCP
* Waiting for the Istio system installation completes. It usually takes 10 - 15 minutes



Testing Prerequisite
=====================

* Step 2 from Installation above need to be executed before running tests

* Istio system has been installed on an OpenShift cluster

* A test namespace/project `bookinfo` need to be created and OCP cluster priviledge has been granted to the `bookinfo` namespace/project. 
  * `$ oc adm policy add-scc-to-user privileged -z default -n bookinfo`
  * `$ oc adm policy add-scc-to-user anyuid -z default -n bookinfo`
  * Priviledge permission is a temporary requirement for any OCP namespace/project to work with sidecar deployments. We don't need to deploy the sample application `bookinfo` before running tests


Testing
-------------------------

Go to directory "`test/maistra`" 
- To run all the test cases (End-to-End run): `go test -timeout 2h -v`
- To run a specific test case: `go test -run [test case number, e.g. 03] -v`
    
    Note: tc_14_authentication_test execution time is more than 10 minutes. If you only want to run tc_14, use -timeout 20m: `go test -run 14 -timeout 20m -v` 
