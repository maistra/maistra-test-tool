// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ossm

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestSMMRAutoCreationAndDeletion(t *testing.T) {
	NewTest(t).Id("T39").Groups(Full, Disconnected, ARM, Persistent).Run(func(t TestHelper) {
		t.Log("This test verifies what happens to the SMMR when SMM is created and deleted")

		t.Cleanup(func() {
			oc.ApplyString(t, meshNamespace, smmr) // revert SMMR to original state
		})

		DeployControlPlane(t)

		t.LogStep("Delete SMMR")
		oc.DeleteResource(t, meshNamespace, "smmr", "default")

		t.LogStep("Create two namespaces")
		oc.CreateNamespace(t, ns.Foo, ns.Bar)

		t.NewSubTest("create first SMM").Run(func(t TestHelper) {
			t.Log("This test checks if the SMMR is created when you create a ServiceMeshMember")

			t.LogStep("Create ServiceMeshMembers in namespaces foo and bar")
			oc.ApplyString(t, ns.Foo, smm)
			oc.ApplyString(t, ns.Bar, smm)

			t.LogStep("Wait for SMMR to be ready")
			oc.WaitSMMRReady(t, meshNamespace)

			t.LogStep("Check both namespaces are shown as members in SMMR")
			retry.UntilSuccess(t, func(t TestHelper) {
				shell.Execute(t,
					fmt.Sprintf(`oc get smmr default -n %s -o=jsonpath='{.status.members[*]}{"\n"}'`, meshNamespace),
					assert.OutputContains(ns.Foo, "SMMR has the member foo", "SMMR does not have the namespaces foo and bar"),
					assert.OutputContains(ns.Bar, "SMMR has the member bar", "SMMR does not have the namespaces foo and bar"))
			})
		})

		t.NewSubTest("delete non-terminal SMM").Run(func(t TestHelper) {
			t.Log("This test verifies that the SMMR isn't deleted when one SMM is deleted, but other SMMs still exist")
			t.Log("See https://issues.redhat.com/browse/OSSM-2374 (implementation)")
			t.Log("See https://issues.redhat.com/browse/OSSM-3450 (test)")

			t.LogStep("Delete one SMM, but keep the other")
			oc.DeleteFromString(t, ns.Bar, smm)

			t.LogStep("Check if SMMR becomes ready (it won't be if it gets deleted)")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.WaitSMMRReady(t, meshNamespace)
			})
		})

		t.NewSubTest("delete terminal SMM").Run(func(t TestHelper) {
			t.Log("This test verifies tht the SMMR is deleted when the last SMM is deleted")
			t.Log("See https://issues.redhat.com/browse/OSSM-2374 (implementation)")
			t.Log("See https://issues.redhat.com/browse/OSSM-3450 (test)")

			t.LogStep("Delete last SMM")
			oc.DeleteFromString(t, ns.Foo, smm)

			t.LogStep("Check that SMMR is deleted")
			retry.UntilSuccess(t, func(t TestHelper) {
				shell.Execute(t,
					fmt.Sprintf("oc get smmr -n %s default || true", meshNamespace),
					assert.OutputContains("not found",
						"SMMR has been deleted",
						"SMMR hasn't been deleted"))
			})
		})

	})
}

func TestSMMReconciliation(t *testing.T) {
	NewTest(t).Groups(Full, Disconnected, ARM).Run(func(t TestHelper) {
		t.Log("This test verifies whether the member-of label is added back to the namespace")
		t.Log("See https://issues.redhat.com/browse/OSSM-1397")

		if !(env.GetSMCPVersion().Equals(env.GetOperatorVersion())) {
			t.Skipf("Skipped because This test case is only needed to be tested when the SMCP version is the latest version available in the Operator. Operator version: %s SMCP version: %s", env.GetOperatorVersion(), env.GetSMCPVersion())
		}

		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
		})

		DeployControlPlane(t)

		t.Log("Remove maistra.io/member-of label from bookinfo namespace")
		oc.RemoveLabel(t, "", "Namespace", ns.Bookinfo, "maistra.io/member-of")

		t.LogStep("Check if label was added back by the operator")
		retry.UntilSuccess(t, func(t TestHelper) {
			oc.GetYaml(t,
				"", "namespace", ns.Bookinfo,
				assert.OutputContains(
					"maistra.io/member-of",
					"The maistra.io/member-of label was added back",
					"The maistra.io/member-of label was not added back"))
		})
	})
}

var (
	smm = `
apiVersion: maistra.io/v1
kind: ServiceMeshMember
metadata:
  name: default
spec:
  controlPlaneRef:
    name: basic
    namespace: ` + meshNamespace
)
