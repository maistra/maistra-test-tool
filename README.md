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

## Prerequisites

Before you run the tests in this repository, you must ensure that the following prerequisites are met:

1. The OpenShift CLI tool `oc` is installed. You can download it from [mirror openshift-v4 clients](https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/). Extract the `openshift-client-...tar.gz` file and ensure that the parent directory of `oc` and `kubectl` is part of your `PATH` environment variable.

2. You must be logged into the cluster. You can log in using the command `oc login -u [user] -p [token] --server=[OCP API server]`

3. RedHat Service Mesh Operator is installed in the cluster.

4. The `gotestsum` package is installed on the machine used to run the test suite:
    ```console
    go install gotest.tools/gotestsum@latest
    ```
5. The [Helm](https://helm.sh/docs/intro/install/) package manager is installed on the machine used to run the test suite:


## How to run tests

To run the tests, you can either use `make test` or run the `./scripts/runtests.sh` script directly. 
Both approaches are equivalent, since `make test` simply runs the `runtests.sh` script. 
Both support running a single test, a group of tests, or the entire suite. 
For information about the test results, refer to section [Test Results](#Test results).

### Running all tests

To run all the test cases against all the `ServiceMeshControlPlane` versions supported by the current version of the OSSM operator, run the following command:
```console
make test
```

This command runs the entire test suite against the minimum supported `ServiceMeshControlPlane` version, then for the next, and so on.

#### Running against a specific Operator version

By default, maistra-test-tool assumes that the OSSM Operator version is `2.5.0` and runs tests against the `v2.3`, `v2.4`, and `v2.5` version of the ServiceMeshControlPlane.
To run against the '2.4.x' version of the Operator, run the tests with the `OPERATOR_VERSION` environment variable set to `2.4.x`. For example, for Operator version `2.4.2`, run the tests as follows:

```console
OPERATOR_VERSION=2.4.2 make test
```

### Running a group of tests

To run all the test cases in a specific test group against all the supported `ServiceMeshControlPlane` versions, specify the test group name in the `TEST_GROUP` environment variable.

For example, to run the tests in the `smoke` group, run the following command:
```console
TEST_GROUP=smoke make test
```

Take in count that when you set `disconnected` as `TEST_GROUP`, the test will need to pass also the bastion host using this variable: `BASTION_HOST`, if you don't set it, the test will fail because the repleace of the image will fail.

See [pkg/util/test/test.go](pkg/util/test/test.go#L13-L18) for a list of available test groups.

### Running a single test case

To run a single test case against all the supported `ServiceMeshControlPlane` versions, specify the name of the test function after `make test <name>`.

For example, to run the `TestFaultInjection` test case, run the following command:
```console
make test TestFaultInjection
```

Alternatively, you can run a specific test case by specifying the name in the `TEST_CASE` environment.

For example, to run the `TestFaultInjection` test case, run the following command:
```console
TEST_CASE=TestFaultInjection make test
```


See the `*_test.go` source files to find the test case names.


### Running against a single ServiceMeshControlPlane version

By default, `make test` runs test cases against all supported versions of `ServiceMeshControlPlane`. When you want to run against a single `ServiceMeshControlPlane` version, specify the version in the `SMCP_VERSION` environment variable.

For example, to run the tests against version `v2.4`, run them as follows:
```console
SMCP_VERSION=2.4 make test
```

NOTE: you may include or omit the `v` prefix in the version number. 


### Running on architectures other than x86

By default, the tests assume that the cluster nodes use the x86 architecture. If your cluster uses a different architecture, set the `OCP_ARCH` environment variable before running the tests.

For IBM Power Systems, run tests with: 
```
OCP_ARCH=p make test
```

For IBM zSystems, run tests with:
```
OCP_ARCH=z make test
```

For ARM-based clusters, run tests with: 
```
OCP_ARCH=arm make test
```


### Running on Red Hat Openshift Service on AWS (ROSA)

To run tests on Red Hat Openshift Service on AWS (ROSA), set the `ROSA` environment variable to `true`:

```console
ROSA=true make test
```

### Disable must-gather for failed tests cases

To disable must-gather run after each test case failure in the test run, set the `MUST_GATHER` environment variable to `false`. Take into count that if the variable does not exist by default it is set to `true`:

```console
MUST_GATHER=false make test
```

### Running multi-cluster test cases

The test suite contains both single- and multi-cluster test cases. 
By default, only single-cluster test cases are run. 
To run multi-cluster tests, ensure that the `KUBECONFIG2` environment variable points to the `kubeconfig` file for the second cluster. 


### Running tests in VSCode, GoLand, or another IDE

The tests in maistra-test-tool are standard go tests and can be run in an IDE using standard methods. No special setup is required.

### Reducing log output

Due to the eventually-consistent nature of OpenShift clusters, each test performs a series of retry attempts of each action it performs. 
By default, each attempt is logged, which typically results in a series of transient failures being shown in the log. 
You can prevent maistra-test-tool from logging each attempt by setting the `LOG_FAILED_RETRY_ATTEMPTS` environment variable to `false`.
The verbose output is mostly useful while writing or debugging tests. In CI systems, using the less-verbose form might be preffered.

To disable verbose logging, run the tests as follows:
```console
LOG_FAILED_RETRY_ATTEMPTS=false make test
```

### Running tests in a container

You can also run the test suite in a container, using the image `quay.io/maistra/maistra-test-tool:latest`. 
In addition to the environment variables explained above, you must also set the following environment variables:
- `OCP_API_URL` - The URL of the OpenShift API server.
- `OCP_TOKEN` - The authentication token for OpenShift. Alternatively, you can set the username and password via the `OCP_CRED_USR` and `OCP_CRED_PSW` environment variables.
- `OCP_CRED_USR` - The username to use when logging into OpenShift.
- `OCP_CRED_PSW` - The login password.


For example, to run all the tests against your local CodeReady Containers cluster, run the following command:
```console
podman run -it \
  --add-host api.crc.testing:$(crc ip) \
  -e OCP_API_URL=https://api.crc.testing:6443 \
  -e OCP_TOKEN=<token> \ 
  quay.io/maistra/maistra-test-tool:latest
```

Use the `-e` option to set the environment variables that affect the execution of the test suite, as described in the previous sections. 

## Test Results

Each time you run `make test`, a new directory named `result-<timestamp>` containing the results of the test run is created under `tests/`. 
Additionally, a symbolic link named `result-latest` is updated to point to the latest result directory every time you run `make test`.

The result directory might look as the following example:

```console
tests/
└── result-20230203040506/          The root result directory for a particular test run.
    ├── v2.2/                       Contains the results against the v2.3 version of the ServiceMeshControlPlane.
    │   ├── failures-must-gather/   Contains must-gather snapshots of cluster resources for each failed test.
    │   │   └── 123-TestSomething   Must-gather captured when the TestSomething failed.
    │   ├── failed.log              Output of the failed test cases.
    │   ├── output.log              Output of all test cases executed for this ServiceMeshControlPlane version.
    │   ├── report.xml              JUnit XML report for this ServiceMeshControlPlane version.
    │   ├── reruns.txt              List of test cases that failed and were executed multiple times.
    │   └── skipped.log             List of skipped test cases and corresponding reasons.
    │   └── documentation.txt       List of all the test cases and subtest cases executed in the test run and his steps.
    ├── ...
    └── output.log                  The full test log across all SMCP versions.
```

## Help

You can run `make help` to get the available commands.

```console
$ make help
Usage: make <target>

Available targets:
  all               - build and run all tests
  build             - build the test binary
  check             - run all pre-commit checks
  lint              - run all linters
  lint-go           - run the Go linter
  test              - run all tests
  test-cleanup      - delete all test resources
  Test<test-name>   - run the specified test
  image             - build the container image
  push              - push the container image to the registry
  clean             - remove all generated files
  test-groups       - list all test groups
  test-groups-<group-name>
                    - list all tests in the specified group
  help              - print this help message
```

## Verify available testing groups

You can run `make test-groups` to get the available testing groups.

```console
$ make test-groups
Available test groups:

ARM
Full
Smoke
InterOp
Disconnected

Test group count: 5
To run all tests in a group, use 'TEST_GROUP=<group-name> make test'
```

## Verify test packages that are part of a testing group

You can run `make test-groups-<group-name>` to get the available test packages that are part of a testing group.

```console
$ make test-groups-Smoke
Available tests in group 'Smoke':

pkg/tests/tasks/traffic/request_routing_test.go
pkg/tests/ossm/smoke_test.go

Test package count in Test Group 'Smoke': 2
To run all tests in group 'Smoke', use 'TEST_GROUP='Smoke' make test'
```

## License

[Maistra OpenShift Test Tool](https://github.com/maistra/maistra-test-tool) is [Apache 2.0 licensed](https://github.com/maistra/maistra-test-tool/blob/development/LICENSE)
