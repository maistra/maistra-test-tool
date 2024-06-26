package ossm

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestRoutePreventAdditionalIngress(t *testing.T) {
	NewTest(t).Id("T48").Groups(Full, ARM).MaxVersion(version.SMCP_2_5).Run(func(t TestHelper) {

		meshValues := map[string]interface{}{
			"Version": env.GetSMCPVersion().String(),
			"Rosa":    env.IsRosa(),
		}

		ER := "extra-routes"

		t.Cleanup(func() {
			oc.DeleteNamespace(t, ER)
		})

		t.LogStep("Create namespace")
		oc.CreateNamespace(t, ER)

		t.LogStep("Create SMCP on new Namespace")
		oc.ApplyTemplate(t, ER, smcp_additionalIngress, meshValues)
		oc.WaitSMCPReady(t, ER, "basic")

		t.LogStep("Verify that Route for additional ingress was not created")
		shell.Execute(t,
			fmt.Sprintf("oc get routes test-ingress -n %s || true", ER),
			assert.OutputContains(
				"not found",
				"Ingress Route is not created",
				"Ingress Route is created"))

		t.NewSubTest("additional ingress gateway route creation").Run(func(t TestHelper) {
			t.Log("Verify that route for additional ingress was created")
			t.Log("Reference: https://issues.redhat.com/browse/OSSM-3909")

			ER2 := "extra-routes22"

			t.Cleanup(func() {
				oc.DeleteNamespace(t, ER2)
			})

			t.LogStep("Create namespace")
			oc.CreateNamespace(t, ER2)

			t.LogStep("Create IGW on new namespace")
			oc.ApplyTemplate(t, ER2, smcp_igw, meshValues)
			oc.WaitSMCPReady(t, ER2, "basic")

			t.LogStep("Verify that Route for additional ingress was created")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Get(t,
					ER2,
					"route", "igw",
					assert.OutputContains("igw-"+ER2,
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
  version: {{ .Version }}
  tracing:
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
  {{ if .Rosa }} 
  security:
    identity:
      type: ThirdParty
  {{ end }}
  `
	smcp_additionalIngress = `
kind: ServiceMeshControlPlane
apiVersion: maistra.io/v2
metadata:
  name: basic
spec:
  version: {{ .Version }}
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
  {{ if .Rosa }} 
  security:
    identity:
      type: ThirdParty
  {{ end }}
  `
)
