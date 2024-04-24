package ossm

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestRoutePreventAdditionalIngress(t *testing.T) {
	NewTest(t).Id("T48").Groups(Full, ARM).MaxVersion(version.SMCP_2_5).Run(func(t TestHelper) {

		ER := "extra-routes"

		t.Cleanup(func() {
			oc.DeleteNamespace(t, ER)
		})

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

		t.NewSubTest("ingress gateway route creation").Run(func(t TestHelper) {

			ER2 := "extra-routes22"

			t.Cleanup(func() {
				oc.DeleteNamespace(t, ER2)
			})

			t.LogStep("Create namespace")
			oc.CreateNamespace(t, ER2)

			t.LogStep("Create IGW on new namespace")
			oc.ApplyString(t, ER2, smcp_igw)

			t.LogStep("Verify the IGW Route")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Get(t,
					ER2,
					"routes", "",
					assert.OutputContains("igw",
						"Route igw is created",
						"Route igw is not created, need to check the additional ingress gateway"))
			})
		})
	})
}

var (
	smcp_igw = `
kind: ServiceMeshControlPlane
apiVersion: maistra.io/v2
metadata:
  name: basic
spec: 
  version: v2.5
  tracing:
    type: Jaeger
    sampling: 10000
  policy:
    type: Istiod
  telemetry:
    type: Istiod
  gateways:
    additionalIngress:
      igw:
        enabled: true
        namespace: extra-routes22
    ingress:
      enabled: false 
  `
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
