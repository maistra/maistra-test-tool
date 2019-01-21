# maistra-ocp-istio-test-tool
A Testing Tool For Running Istio Doc Tasks on OpenShift

Prerequisite
---------------------

* `oc` client tool need to be installed and `oc login [host] [--token=<hidden>]` need to be executed before running tests

* `kubectl` client tool need to be installed before running tests. Because tests use the "istio.io/istio/tests/util" and `kubectl` is hard coded in that util package. ( An alternative way also works fine: Copy `oc` client tool and rename the copy as `kubectl` in `/usr/bin`)

