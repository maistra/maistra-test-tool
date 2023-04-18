package ossm

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestSMM(t *testing.T) {
	NewTest(t).Id("T39").Groups(Full).Run(func(t TestHelper) {
		t.Log("This Suite verifies SMM and SMMR behaviors")
		foo := "foo"
		bar := "bar"
		t.Cleanup(func() {
			oc.ApplyString(t, meshNamespace, smmr) // If we apply the default SMMR over the current will be updated
		})

		t.LogStep("Setup: Delete default SMMR in mesh namespace and create two namespaces: foo and bar")
		oc.DeleteResource(t, meshNamespace, "smmr", "default")
		oc.CreateNamespace(t, foo)
		oc.CreateNamespace(t, bar)

		t.NewSubTest("Create SMM creates SMMR").Run(func(t TestHelper) {
			t.LogStep("Create SMM in namespace:foo and bar; expected: SMMR is created")
			oc.ApplyString(t, foo, ServiceMeshMember)
			oc.ApplyString(t, bar, ServiceMeshMember)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.WaitSMMRReady(t, meshNamespace)
			})

			t.LogStep("Check SMMR has the members foo and bar")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf(`oc get smmr default -n %s -o=jsonpath='{.status.members[*]}{"\n"}'`, meshNamespace),
					assert.OutputContains(foo, "SMMR has the members foo", "SMMR does not have the namespaces foo and bar"),
					assert.OutputContains(bar, "SMMR has the members bar", "SMMR does not have the namespaces foo and bar"))
			})
		})

		t.NewSubTest("Delete SMM with a SMMR with more than one members").Run(func(t TestHelper) {
			t.Log("See https://issues.redhat.com/browse/OSSM-2374 as reference and https://issues.redhat.com/browse/OSSM-3450 for testing")
			t.Log("Delete SMM in bar namespace; expected: SMMR is not deleted because has more than one namespace in configuration")
			t.LogStep("Delete SMM in namespace: bar. Expected: SMMR is not deleted")
			oc.DeleteFromString(t, bar, ServiceMeshMember)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.WaitSMMRReady(t, meshNamespace)
			})
		})

		t.NewSubTest("Delete SMM with a SMMR with one members").Run(func(t TestHelper) {
			t.Log("See https://issues.redhat.com/browse/OSSM-2374 as reference and https://issues.redhat.com/browse/OSSM-3450 for testing")
			t.Log("Delete SMM in foo namespace; expected: SMMR is deleted because has only one namespace in configuration")
			t.LogStep("Delete SMM in namespace: foo. Expected: SMMR is deleted")
			oc.DeleteFromString(t, foo, ServiceMeshMember)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf(`oc get smmr -n %s default || true`, meshNamespace),
					assert.OutputContains(
						"not found",
						"SMMR is deleted",
						"SMMR is not deleted"))
			})
		})

	})
}

var (
	ServiceMeshMember = `
apiVersion: maistra.io/v1
kind: ServiceMeshMember
metadata:
  name: default
spec:
  controlPlaneRef:
    name: basic
    namespace: istio-system
  `
)
