package ingress

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestGatewayApiControllerMode(t *testing.T) {
	NewTest(t).Id("T41").Groups(Full, InterOp).Run(func(t TestHelper) {
		if env.GetSMCPVersion().LessThan(version.SMCP_2_3) {
			t.Skip("TestGatewayApiControllerMode was added in v2.3")
		}
		ns := "foo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		ossm.DeployControlPlane(t)

		smcpName := "basic"

		t.LogStep("Install Gateway API CRD's")
		shell.Executef(t, "kubectl get crd gateways.gateway.networking.k8s.io &> /dev/null ||   { kubectl kustomize github.com/kubernetes-sigs/gateway-api/config/crd/experimental?ref=v0.5.1 | kubectl apply -f -; }")

		oc.CreateNamespace(t, ns)

		t.LogStep("Install httpbin")
		app.InstallAndWaitReady(t, app.Httpbin(ns))
		t.Cleanup(func() {
			app.Uninstall(t, app.Httpbin(ns))
		})

		t.LogStep("Deploy the Gateway app")
		oc.ApplyString(t, ns, gatewayDeployment)
		t.Cleanup(func() {
			oc.DeleteFromString(t, ns, gatewayDeployment)
		})

		t.LogStep("Deploy the Gateway SMCP")
		if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
        spec:
          runtime:
            components:
              pilot:
                container:
                  env:
                    PILOT_ENABLE_GATEWAY_API: “true”
                    PILOT_ENABLE_GATEWAY_API_STATUS: “true”
                    PILOT_ENABLE_GATEWAY_API_DEPLOYMENT_CONTROLLER: “true”`)
			t.Cleanup(func() {
				oc.Patch(t, meshNamespace, "smcp", smcpName, "json",
					`[{"op": "remove", "path": "/spec/runtime"}]`)
			})

		} else {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
        spec:
          techPreview:
            gatewayAPI:
              enabled: true`)

			t.Cleanup(func() {
				oc.Patch(t, meshNamespace, "smcp", smcpName, "json",
					`[{"op": "remove", "path": "/spec/techPreview"}]`)
			})
		}

		t.LogStep("Deploy the Gateway API configuration including a single exposed route (i.e., /get)")
		oc.ApplyString(t, ns, GetGatewayAPI)
		t.Cleanup(func() {
			oc.DeleteFromString(t, ns, GetGatewayAPI)
		})

		t.LogStep("Verify the deployed gatewayApi")
		shell.Execute(t,
			fmt.Sprintf("oc wait --for=condition=Ready -n %s gateways.gateway.networking.k8s.io gateway", ns),
			assert.OutputContains("condition met",
				"Gateway is running",
				"Gateway is not running"))

		t.LogStep("Verfiy the GatewayApi access the httpbin service using curl")
		retry.UntilSuccess(t, func(t TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app=istio-ingressgateway", meshNamespace),
				"istio-proxy",
				"curl http://gateway.foo.svc.cluster.local:8080/get -H Host:httpbin.example.com -s -o /dev/null -w %{http_code}",
				assert.OutputContains("200",
					"Access the httpbin service with GatewayApi",
					"Unable to access the httpbin service with GatewayApi"))
		})
	})
}

const GetGatewayAPI = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: gateway
  namespace: foo
spec:
  gatewayClassName: istio
  listeners:
  - name: default
    hostname: "*.example.com"
    port: 8080
    protocol: HTTP
    allowedRoutes:
      namespaces:
        from: All
---
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: http
  namespace: foo
spec:
  parentRefs:
  - name: gateway
    namespace: foo
  hostnames: ["httpbin.example.com"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: httpbin
      port: 8000`

const gatewayDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway
  namespace: foo
spec:
  selector:
    matchLabels:
      istio: ingressgateway
  template:
    metadata:
      annotations:
        inject.istio.io/templates: gateway
      labels:
        istio: ingressgateway
        sidecar.istio.io/inject: "true"
        istio.io/gateway-name: gateway
    spec:
      containers:
        - name: istio-proxy
          image: auto
`
