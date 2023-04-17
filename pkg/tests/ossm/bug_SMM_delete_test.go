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

// referance - https://issues.redhat.com/browse/OSSM-2374

// testing Jira - https://issues.redhat.com/browse/OSSM-3450

func TestSMMDelete(t *testing.T) {
	NewTest(t).Id("T39").Groups(Full).Run(func(t TestHelper) {
		t.Log("Verify the SMMR will be creates Automatically with SMM and SMMR is not deleted when SMM is deleted")
		t.Cleanup(func() {
			oc.DeleteNamespace(t, "my-awesome-project")
			oc.ApplyString(t, meshNamespace, smmr) // If we apply the default SMMR over the current will be updated
		})

		t.LogStep("Delete default SMMR in mesh namespace and create new ServiceMeshMember in namespace: bookinfo and my-awesome-project. Expected: SMMR will be created automatically")
		shell.Execute(t, fmt.Sprintf(`oc delete smmr -n %s default`, meshNamespace))
		oc.CreateNamespace(t, "bookinfo")
		oc.CreateNamespace(t, "my-awesome-project")
		oc.ApplyString(t, "bookinfo", ServiceMeshMember)
		oc.ApplyString(t, "my-awesome-project", ServiceMeshMember)
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.WaitSMMRReady(t, meshNamespace)
		})

		t.LogStep("Delete SMM in namespace: my-awesome-project. Expected: SMMR is not deleted")
		oc.DeleteFromString(t, "my-awesome-project", ServiceMeshMember)
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.WaitSMMRReady(t, meshNamespace)
		})

		t.LogStep("Delete SMM in namespace: bookinfo. Expected: SMMR is deleted")
		oc.DeleteFromString(t, "bookinfo", ServiceMeshMember)
		retry.UntilSuccess(t, func(t test.TestHelper) {
			shell.Execute(t,
				fmt.Sprintf(`oc get smmr -n %s default || true`, meshNamespace),
				assert.OutputContains(
					"not found",
					"Sucess: SMMR is deleted",
					"Error: SMMR is not deleted"))
		})
	})
}

var (
	ServiceMeshMember = `
apiVersion: "maistra.io/v1"
kind: "ServiceMeshMember"
metadata:
  name: default
spec:
  controlPlaneRef:
    name: basic
    namespace: istio-system
  `
)
