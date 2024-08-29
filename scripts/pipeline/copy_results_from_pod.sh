#!/bin/bash
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
#
# Copyright 2024 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

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

