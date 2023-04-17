package ossm

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
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
		oc.WaitCondition(t, "bookinfo", "smm", "default", "Ready")
		oc.ApplyString(t, "my-awesome-project", ServiceMeshMember)
		oc.WaitCondition(t, "my-awesome-project", "smm", "default", "Ready")
		oc.WaitCondition(t, meshNamespace, "smmr", "default", "Ready")

		t.LogStep("Delete SMM in namespace: my-awesome-project. Expected: SMMR is not deleted")
		oc.DeleteFromString(t, "my-awesome-project", ServiceMeshMember)
		oc.WaitCondition(t, meshNamespace, "smmr", "default", "Ready")

		t.LogStep("Delete SMM in namespace: bookinfo. Expected: SMMR is not deleted")
		oc.DeleteFromString(t, "bookinfo", ServiceMeshMember)
		oc.WaitCondition(t, meshNamespace, "smmr", "default", "Ready")

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
