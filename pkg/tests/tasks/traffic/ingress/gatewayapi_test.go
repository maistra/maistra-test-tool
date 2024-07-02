package ingress

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/gatewayapi"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/version"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

func TestGatewayApi(t *testing.T) {
	NewTest(t).Id("T41").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		if env.GetSMCPVersion().LessThan(version.SMCP_2_3) {
			t.Skip("TestGatewayApi was added in v2.3")
		}

		smcpName := env.GetDefaultSMCPName()
		istiodDeployment := fmt.Sprintf("istiod-%s", smcpName)

		ossm.DeployControlPlane(t)

		t.LogStep("Install Gateway API CRD's")
		gatewayapi.InstallSupportedVersion(t, env.GetSMCPVersion())
		t.Cleanup(func() {
			gatewayapi.UninstallSupportedVersion(t, env.GetSMCPVersion())
		})

		oc.CreateNamespace(t, ns.Foo)

		t.NewSubTest("Check default Gateway API settings").Run(func(t TestHelper) {
			t.Log("Check Gateway API is disabled by default")

			t.LogStep("Check istiod deployment environment variables")
			expectedValue := "false"
			istiodEnvs := oc.GetJson(t, meshNamespace, "deployment", istiodDeployment, `{.spec.template.spec.containers[0].env}`)
			var istiodEnvVars []EnvVar
			if err := json.Unmarshal([]byte(istiodEnvs), &istiodEnvVars); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			checkedVars := 0
			for _, envVar := range istiodEnvVars {
				if envVar.Name == "PILOT_ENABLE_GATEWAY_API" ||
					envVar.Name == "PILOT_ENABLE_GATEWAY_API_STATUS" ||
					envVar.Name == "PILOT_ENABLE_GATEWAY_API_DEPLOYMENT_CONTROLLER" {
					checkedVars++
					if envVar.Value != expectedValue {
						t.Errorf("Expected %s to be %s, got %s", envVar.Name, expectedValue, envVar.Value)
					} else {
						t.Logf("Env %s is set to %s", envVar.Name, envVar.Value)
					}
					if checkedVars == 3 {
						break
					}
				}
			}

			if checkedVars != 3 {
				t.Errorf("Expected 3 PILOT_ENABLE_GATEWAY_API vars to be checked, got %d", checkedVars)
			}
		})

		t.NewSubTest("Deploy the Kubernetes Gateway API").Run(func(t TestHelper) {

			t.Cleanup(func() {
				oc.RecreateNamespace(t, meshNamespace)
				oc.RecreateNamespace(t, ns.Foo)
			})

			t.LogStep("Install httpbin")
			app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))

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
			oc.ApplyTemplate(t, ns.Foo, gatewayapi.GatewayAndRouteYAML, map[string]string{"GatewayClassName": "istio"})
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns.Foo, gatewayapi.GatewayAndRouteYAML, map[string]string{"GatewayClassName": "istio"})
			})

			t.LogStep("Wait for Gateway to be ready")
			oc.WaitCondition(t, ns.Foo, "Gateway", "gateway", gatewayapi.GetWaitingCondition(env.GetSMCPVersion()))

			t.LogStep("Verfiy the GatewayApi access the httpbin service using curl")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=istio-ingressgateway", meshNamespace),
					"istio-proxy",
					fmt.Sprintf("curl http://%s.foo.svc.cluster.local:8080/get -H Host:httpbin.example.com -s -o /dev/null -w %%{http_code}", gatewayapi.GetDefaultServiceName(env.GetSMCPVersion(), "gateway", "istio")),
					assert.OutputContains("200",
						"Access the httpbin service with GatewayApi",
						"Unable to access the httpbin service with GatewayApi"))
			})
		})

		t.NewSubTest("Deploy the Gateway-Controller Profile").Run(func(t TestHelper) {
			if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
				t.Skip("Gateway-Controller Profile was added in v2.4")
			}

			t.Cleanup(func() {
				oc.RecreateNamespace(t, meshNamespace)
				oc.RecreateNamespace(t, ns.Foo)
			})

			t.LogStep("Install httpbin")
			app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))

			t.LogStep("Deploy SMCP with the profile")
			oc.ApplyTemplate(t,
				meshNamespace,
				gatewayControllerProfile,
				map[string]string{"Name": "basic", "Version": env.GetSMCPVersion().String()})
			oc.WaitSMCPReady(t, meshNamespace, "basic")

			t.LogStep("delete default SMMR and create custom SMMR")
			oc.DeleteFromString(t, meshNamespace, defaultSMMR)
			oc.ApplyTemplate(t, meshNamespace, createSMMR, map[string]string{"Member": ns.Foo})
			oc.WaitSMMRReady(t, meshNamespace)

			t.LogStep("Deploy the Gateway API configuration including a single exposed route (i.e., /get)")
			oc.ApplyTemplate(t, ns.Foo, gatewayapi.GatewayAndRouteYAML, map[string]string{"GatewayClassName": "ocp"})
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns.Foo, gatewayapi.GatewayAndRouteYAML, map[string]string{"GatewayClassName": "ocp"})
			})

			t.LogStep("Wait for Gateway to be ready")
			oc.WaitCondition(t, ns.Foo, "Gateway", "gateway", gatewayapi.GetWaitingCondition(env.GetSMCPVersion()))

			t.LogStep("Verify the Gateway-Controller Profile access the httpbin service using curl")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=istiod", meshNamespace),
					"discovery",
					fmt.Sprintf("curl http://%s.foo.svc.cluster.local:8080/get -H Host:httpbin.example.com -s -o /dev/null -w %%{http_code}", gatewayapi.GetDefaultServiceName(env.GetSMCPVersion(), "gateway", "ocp")),
					assert.OutputContains("200",
						"Access the httpbin service with GatewayApi",
						"Unable to access the httpbin service with GatewayApi"))
			})
		})

	})
}

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
