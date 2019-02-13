# maistra-ocp-istio-test-tool
A Testing Tool For Running Istio Doc Tasks on OpenShift

Introduction
---------------------

This project aims to follow [Istio Doc Tasks](https://preliminary.istio.io/docs/tasks/) structure and organize upstream [Istio release-1.1 tests directory](https://github.com/istio/istio/tree/release-1.1/tests). All test cases can be run against an Istio system running on an OpenShift 3.11 or 4.0 cluster.

Most of the utilities are imported from upstream "istio.io/istio/tests/util". For more utilities information, user can check upstream [Istio release-1.1 util directory](https://github.com/istio/istio/tree/release-1.1/tests/util).


Go Version
-----------------

go1.10.7 or above

Prerequisite
---------------------

* `oc` client tool need to be installed and command `oc login [host] [--token=<hidden>]` need to be executed before running tests

* Istio system has been installed on an OpenShift 3.11 or 4.0 cluster

* A test namespace/project `bookinfo` need to be created and OCP cluster priviledge has been granted to the `bookinfo` namespace/project. There is No requirement to deploy the sample application `bookinfo` before running tests. Our test cases cover all of the sample applications deployment and cleanup.
  (Priviledge permission is a temporary requirement for any OCP namespace/project to work with sidecar deployments)

* If there is only `oc` client installed and no `kubectl` installed,  need to have a soft link. `sudo ln -s oc /usr/bin/kubectl`

* Two utility packages from Istio upstream are needed before running tests. `go get "istio.io/istio/tests/util"`;  `go get "istio.io/istio/pkg/log"`


How to run each test case
-------------------------

User can go to directory `tests/maistra` 
- To run all the test cases (End-to-End run): `go test -timeout 2h -v`
- To run a specific test case: `go test -run [test case number] -v`