# maistra-ocp-istio-test-tool
A Testing Tool For Running Istio Doc Tasks on OpenShift

Introduction
---------------------

This project aims to follow [Istio Doc Tasks](https://istio.io/docs/tasks/) structure and organize upstream [Istio release-1.1 tests directory](https://github.com/istio/istio/tree/release-1.1/tests). All test cases can be run against an Istio system running on an OpenShift 3.11 or 4.0 cluster.

Most of the utilities are imported from upstream "istio.io/istio/tests/util". For more utilities information, user can check upstream [Istio release-1.1 util directory](https://github.com/istio/istio/tree/release-1.1/tests/util).

Prerequisite
---------------------

* `oc` client tool need to be installed and `oc login [host] [--token=<hidden>]` need to be executed before running tests

* `kubectl` client tool need to be installed before running tests. Because tests use the "istio.io/istio/tests/util" and `kubectl` is hard coded in that util package. ( An alternative way also works fine: Copy `oc` client tool and rename the copy as `kubectl` in `/usr/bin`)

