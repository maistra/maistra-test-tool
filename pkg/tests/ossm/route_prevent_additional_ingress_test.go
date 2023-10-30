package ossm

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestRoutePreventAdditionalIngress(t *testing.T) {
	NewTest(t).Id("T48").Groups(Full).MaxVersion(version.SMCP_2_5).Run(func(t TestHelper) {

		ER := "extra-routes"

		t.Cleanup(func() {
			oc.DeleteFromString(t, ER, smcp_additionalIngress)
			oc.DeleteNamespace(t, "extra-routes")
		})

		DeployControlPlane(t)

		t.LogStep("Create namespace")
		oc.CreateNamespace(t, ER)

		t.LogStep("Create SMCP on new Namespace")
		oc.ApplyString(t, ER, smcp_additionalIngress)

		t.LogStep("Verify the Ingress Route")
		shell.Execute(t,
			fmt.Sprintf("oc get routes test-ingress -n %s || true", ER),
			assert.OutputContains(
				"not found",
				"Ingress Route is not created",
				"Ingress Route is created"))

	})
}

var (
	smcp_additionalIngress = `
kind: ServiceMeshControlPlane
apiVersion: maistra.io/v2
metadata:
  name: basic
spec: 
  gateways: 
    additionalIngress: 
      test-ingress: 
        enabled: true
        service: 
          externalName: test-ingress
          metadata: 
            labels: 
              app: test
          ports: 
          - name: http
            port: 80
            protocol: TCP
            targetPort: 8080
          type: ClusterIP
        routeConfig: 
          enabled: false
    ingress: 
      routeConfig: 
        enabled: false
    openshiftRoute: 
      enabled: false 
  `
)
