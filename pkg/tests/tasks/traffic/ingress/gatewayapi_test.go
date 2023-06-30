package ingress

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestGatewayApi(t *testing.T) {
	NewTest(t).Id("T41").Groups(Full, InterOp).Run(func(t TestHelper) {
		if env.GetSMCPVersion().LessThan(version.SMCP_2_3) {
			t.Skip("TestGatewayApi was added in v2.3")
		}
		ns := "foo"

		smcpName := env.GetDefaultSMCPName()

		ossm.DeployControlPlane(t)

		t.LogStep("Install Gateway API CRD's")
		shell.Executef(t, "kubectl get crd gateways.gateway.networking.k8s.io &> /dev/null && echo 'Gateway API CRDs already installed' || kubectl apply -k github.com/kubernetes-sigs/gateway-api/config/crd/experimental?ref=v0.5.1")

		oc.CreateNamespace(t, ns)

		t.NewSubTest("Deploy the Kubernetes Gateway API").Run(func(t test.TestHelper) {

			t.Cleanup(func() {
				oc.RecreateNamespace(t, meshNamespace)
				oc.RecreateNamespace(t, ns)
			})

			t.LogStep("Install httpbin")
			app.InstallAndWaitReady(t, app.Httpbin(ns))

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
			oc.ApplyTemplate(t, ns, gatewayAndRouteYAML, map[string]string{"GatewayClassName": "istio"})
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, gatewayAndRouteYAML, map[string]string{"GatewayClassName": "istio"})
			})

			t.LogStep("Wait for Gateway to be ready")
			oc.WaitFor(t, ns, "Gateway", "gateway", "condition=Ready")

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

		t.NewSubTest("Deploy the Gateway-Controller Profile").Run(func(t test.TestHelper) {
			if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
				t.Skip("Gateway-Controller Profile was added in v2.4")
			}

			t.Cleanup(func() {
				oc.RecreateNamespace(t, meshNamespace)
				oc.RecreateNamespace(t, ns)
			})

			t.LogStep("Install httpbin")
			app.InstallAndWaitReady(t, app.Httpbin(ns))

			t.LogStep("Deploy SMCP with the profile")
			oc.ApplyTemplate(t,
				meshNamespace,
				gatewayControllerProfile,
				map[string]string{"Name": "basic", "Version": env.GetSMCPVersion().String()})
			oc.WaitSMCPReady(t, meshNamespace, "basic")

			t.LogStep("delete default SMMR and create custom SMMR")
			oc.DeleteFromString(t, meshNamespace, defaultSMMR)
			oc.ApplyTemplate(t, meshNamespace, createSMMR, map[string]string{"Member": ns})
			oc.WaitSMMRReady(t, meshNamespace)

			t.LogStep("Deploy the Gateway API configuration including a single exposed route (i.e., /get)")
			oc.ApplyTemplate(t, ns, gatewayAndRouteYAML, map[string]string{"GatewayClassName": "ocp"})
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, gatewayAndRouteYAML, map[string]string{"GatewayClassName": "ocp"})
			})

			t.LogStep("Wait for Gateway to be ready")
			oc.WaitFor(t, ns, "Gateway", "gateway", "condition=Ready")

			t.LogStep("Verify the Gateway-Controller Profile access the httpbin service using curl")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=istiod", meshNamespace),
					"discovery",
					"curl http://gateway.foo.svc.cluster.local:8080/get -H Host:httpbin.example.com -s -o /dev/null -w %{http_code}",
					assert.OutputContains("200",
						"Access the httpbin service with GatewayApi",
						"Unable to access the httpbin service with GatewayApi"))
			})
		})

	})
}

const gatewayAndRouteYAML = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: gateway
spec:
  gatewayClassName: {{ .GatewayClassName }}
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

const gatewayControllerProfile = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: {{ .Version }}
  profiles:
  - gateway-controller`

const createSMMR = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
    - {{ .Member }}`

const defaultSMMR = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  memberSelectors:
  - matchLabels:
      istio-injection: enabled`
