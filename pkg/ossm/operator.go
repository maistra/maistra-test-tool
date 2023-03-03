// Copyright Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ossm

import (
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupOperatorTest() {
	util.Log.Info("Cleanup ...")
	//delete the infra label from the worker node
	util.Log.Info("Delete Nodeselector and Tolerations from the operator")
	_, err := util.Shell(`oc patch subscription servicemeshoperator -n openshift-operators --type json -p='[{"op": "remove", "path": "/spec/config/nodeSelector"}]'`)
	if err != nil {
		util.Log.Error("Failed to remove the nodeSelector from the operator")
	}
	_, err = util.Shell(`oc patch subscription servicemeshoperator -n openshift-operators --type json -p='[{"op": "remove", "path": "/spec/config/tolerations"}]'`)
	if err != nil {
		util.Log.Error("Failed to remove the tolerations from the operator")
	}
	util.Log.Info("Operator patching completed")
	response, err := util.Shell(`oc adm taint nodes -l node-role.kubernetes.io/infra node-role.kubernetes.io/infra=reserved:NoSchedule- node-role.kubernetes.io/infra=reserved:NoExecute-`)
	if err != nil {
		util.Log.Error("Failed to taint the infra node")
	}
	util.Log.Info(response)
	if strings.Contains(response, "untainted") {
		util.Log.Info("Infra node is not tainted anymore")
	} else {
		util.Log.Error("Fail to remove the taint from the infra node")
	}

}

// TestOperator tests scenario to cover all the test cases related to the OSSM operators
func TestOperator(t *testing.T) {
	defer cleanupOperatorTest()
	defer util.RecoverPanic(t)

	util.Log.Info("Test cases related to OSSM Operators")

	//test case regarding the https://issues.redhat.com/browse/OSSM-2342 issue to run OSSM operator on infra nodes
	t.Run("test_ossm_operator_deploy_infra_nodes", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Testing: Run OSSM Operator on infra nodes")
		//Get and pick one worker node that does not have already installed the istio operator
		workername := pickWorkerNode(t)
		//Label the worker node as infra
		_, err := util.Shell(`oc label node %s node-role.kubernetes.io/infra=`, workername)
		if err != nil {
			t.Fatalf("Failed to label the worker node as infra")
		}
		_, err = util.Shell(`oc label node %s node-role.kubernetes.io=infra`, workername)
		if err != nil {
			t.Fatalf("Failed to label the worker node as infra")
		}
		//verify that the worker node is labeled as infra
		name, err := util.Shell(`oc get nodes -l node-role.kubernetes.io/infra= -o jsonpath='{.items[0].metadata.name}'`)
		if err != nil {
			t.Fatalf("Failed to get the infra node name")
		}
		if name != workername {
			t.Fatalf("Failed to label the worker node as infra")
		}
		//Taint the node. The only validation to check this is the output message
		response, err := util.Shell(`oc adm taint nodes -l node-role.kubernetes.io/infra node-role.kubernetes.io/infra=reserved:NoSchedule node-role.kubernetes.io/infra=reserved:NoExecute`)
		if err != nil {
			t.Fatalf("Failed to taint the infra node")
		}
		if !strings.Contains(response, "tainted") {
			t.Fatalf("Failed to taint the infra node")
		}
		//Edit the subscription to add the infra node to the node selector
		_, err = util.Shell(`oc patch subscription servicemeshoperator -n openshift-operators --type merge -p '{"spec":{"config":{"nodeSelector":{"node-role.kubernetes.io/infra":""},"tolerations":[{"effect":"NoSchedule","key":"node-role.kubernetes.io/infra","value":"reserved"},{"effect":"NoExecute","key":"node-role.kubernetes.io/infra","value":"reserved"}]}}}'`)

		if err != nil {
			t.Fatalf("Failed to patch the subscription")
		}
		//Verify that the operator pod is running on the infra node
		podname, err := util.Shell(`oc get pods -n openshift-operators -l name=istio-operator --field-selector spec.nodeName=%s -o jsonpath='{.items[0].metadata.name}'`, workername)
		if err != nil {
			time.Sleep(time.Duration(20) * time.Second)
			podname, err = util.Shell(`oc get pods -n openshift-operators -l name=istio-operator --field-selector spec.nodeName=%s -o jsonpath='{.items[0].metadata.name}'`, workername)
			if err != nil {
				t.Fatalf("Failed to get the operator pod name")
			}
		}
		node, err := util.Shell(`oc get pods -n openshift-operators %s -o jsonpath='{.spec.nodeName}'`, podname)
		if err != nil {
			t.Fatalf("Failed to get the node name")
		}
		if node != workername {
			t.Fatalf("Failed to run the operator on the infra node")
		}
		_, err = util.Shell(`oc label node %s node-role.kubernetes.io/infra-`, workername)
		if err != nil {
			t.Fatalf("Failed to unlabel the worker node as infra")
		}
		_, err = util.Shell(`oc label node %s node-role.kubernetes.io-`, workername)
		if err != nil {
			t.Fatalf("Failed to unlabel the worker node as infra")
		}
	})

}
func pickWorkerNode(t *testing.T) string {
	workername, err := util.Shell(`oc get nodes -l node-role.kubernetes.io/worker= -o jsonpath='{.items[0].metadata.name}'`)
	if err != nil {
		t.Fatalf("Failed to get the worker node name")
	}
	actualNode, err := util.Shell(`oc get pods -n openshift-operators -l name=istio-operator -o jsonpath='{.items[0].spec.nodeName}'`)
	if err != nil {
		t.Fatalf("Failed to get the actual node name")
	}
	if workername == actualNode {
		workername, err = util.Shell(`oc get nodes -l node-role.kubernetes.io/worker= -o jsonpath='{.items[1].metadata.name}'`)
		if err != nil {
			t.Fatalf("Failed to get the worker node name")
		}
	}

	return workername
}
