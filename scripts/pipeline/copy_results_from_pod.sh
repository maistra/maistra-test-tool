#!/bin/bash

# login
oc login -u ike -p $IKE_PWD --server=$OCP_SERVER --insecure-skip-tls-verify=true

# create pipeline
cd pipeline
oc apply -f openshift-pipeline-subscription.yaml
sleep 40
oc apply -f pipeline-cluster-role-binding.yaml

# start running all tests
oc apply -f pipeline-run-acc-tests.yaml
sleep 10

podName=$(oc get pods -n maistra-pipelines -l tekton.dev/task=run-all-acc-tests -o jsonpath="{.items[0].metadata.name}")

# check test completed
oc logs -n maistra-pipelines ${podName} -c step-run-all-test-cases | grep "#Acc Tests completed#"
while [ $? -ne 0 ]; do
    sleep 60;
    oc logs -n maistra-pipelines ${podName} -c step-run-all-test-cases | grep "#Acc Tests completed#"
done

# collect logs
oc cp maistra-pipelines/${podName}:test.log ${WORKSPACE}/tests/test.log -c step-run-all-test-cases
oc cp maistra-pipelines/${podName}:results.xml ${WORKSPACE}/tests/results.xml -c step-run-all-test-cases

