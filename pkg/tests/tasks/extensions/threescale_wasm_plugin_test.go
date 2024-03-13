package extensions

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/request"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"

	"testing"
)

const (
	tokenURL = "https://raw.githubusercontent.com/istio/istio/release-1.19/security/tools/jwt/samples/demo.jwt"
)

func TestThreeScaleWasmPlugin(t *testing.T) {
	test.NewTest(t).Groups(test.Full, test.ARM).Run(func(t test.TestHelper) {

		if env.GetArch() == "z" || env.GetArch() == "p" {
			t.Skip("Web Assembly is not supported for IBM Z&P")
		}

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Foo)
			oc.RecreateNamespace(t, meshNamespace)
			oc.DeleteNamespace(t, threeScaleNs)
		})

		t.LogStep("Deploy SMCP")
		smcpValues := map[string]interface{}{
			"Name":             smcpName,
			"Version":          env.GetSMCPVersion().String(),
			"Member":           ns.Foo,
			"ClusterWideProxy": false,
		}

		//If there is a cluster-wide proxy in front of OCP, it needs to be set in istio-proxy to be able to download the 3scale plugin from Quay
		clusterWideProxy := oc.GetProxy(t)
		if clusterWideProxy != nil {
			smcpValues["ClusterWideProxy"] = "true"
			smcpValues["HttpProxy"] = clusterWideProxy.HTTPProxy
			smcpValues["HttpsProxy"] = clusterWideProxy.HTTPSProxy
			smcpValues["NoProxy"] = clusterWideProxy.NoProxy
		}

		oc.ApplyTemplate(t, meshNamespace, meshTmpl, smcpValues)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Deploy 3scale mocks")
		oc.CreateNamespace(t, threeScaleNs)
		oc.ApplyString(t, threeScaleNs, threeScaleBackend)
		oc.ApplyString(t, meshNamespace, threeScaleBackendSvcEntry)
		oc.ApplyString(t, threeScaleNs, threeScaleSystem)
		oc.ApplyString(t, meshNamespace, threeScaleSystemSvcEntry)
		oc.WaitAllPodsReady(t, threeScaleNs)

		t.LogStep("Configure JWT authn")
		oc.ApplyTemplate(t, meshNamespace, jwtAuthnTmpl, map[string]interface{}{
			"AppLabel":     "istio-ingressgateway",
			"ForwardToken": true,
		})

		t.LogStep("Apply 3scale WASM plugin to the ingress gateway")
		oc.ApplyTemplate(t, meshNamespace, wasmPluginTmpl, map[string]interface{}{"AppLabel": "istio-ingressgateway"})

		t.LogStep("Deploy httpbin and configure its gateway and routing")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))
		oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/httpbin/httpbin-gateway.yaml")

		t.LogStep("Verify that a request to the ingress gateway with token returns 200")
		ingressGatewayHost := istio.GetIngressGatewayHost(t, meshNamespace)
		headersURL := fmt.Sprintf("http://%s/headers", ingressGatewayHost)
		token := string(curl.Request(t, tokenURL, nil))
		token = strings.Trim(token, "\n")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, headersURL, request.WithHeader("Authorization", "Bearer "+token), require.ResponseStatus(http.StatusOK))
		})

		t.LogStep("Apply JWT config and 3scale plugin to httpbin")
		oc.ApplyTemplate(t, ns.Foo, jwtAuthnTmpl, map[string]interface{}{"AppLabel": "httpbin"})
		oc.ApplyTemplate(t, ns.Foo, wasmPluginTmpl, map[string]interface{}{"AppLabel": "httpbin"})

		// This step would fail if the ingress gateway did not forward Authorization header to httpbin
		t.LogStep("Verify that a request to the ingress gateway with token returns 200")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, headersURL, request.WithHeader("Authorization", "Bearer "+token), require.ResponseStatus(http.StatusOK))
		})

		t.LogStep("Deploy sleep app")
		app.InstallAndWaitReady(t, app.Sleep(ns.Foo))
		httpbinUrl := "http://httpbin:8000/headers"
		CurlOpts := app.CurlOpts{Headers: []string{"Authorization: Bearer " + token}}

		t.LogStep("Verify that a request from sleep to httpbin with token returns 200")
		app.AssertSleepPodRequestSuccess(t, ns.Foo, httpbinUrl, CurlOpts)

		t.LogStep("Apply JWT config and 3scale plugin to sleep")
		oc.ApplyTemplate(t, ns.Foo, jwtAuthnTmpl, map[string]interface{}{"AppLabel": "sleep"})
		oc.ApplyTemplate(t, ns.Foo, wasmPluginTmpl, map[string]interface{}{"AppLabel": "sleep"})

		if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_3) {
			// A request should fail, because in 2.3+, WASM plugins are applied to inbound and outbound listeners.
			// JWT authentication filter is applied only to inbound listeners, so 3scale plugin configured
			// to use JWT filter metadata always fails on outbound.
			t.LogStep("Verify that a request from sleep to httpbin returns 403")
			app.AssertSleepPodRequestForbidden(t, ns.Foo, httpbinUrl, CurlOpts)
		} else {
			t.LogStep("Verify that a request from sleep to httpbin returns 200")
			app.AssertSleepPodRequestSuccess(t, ns.Foo, httpbinUrl, CurlOpts)
		}

		if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_3) {
			t.LogStep("Set flag APPLY_WASM_PLUGINS_TO_INBOUND_ONLY in istiod and send a request again")

			smcpValues["ApplyWasmPluginsToInboundOnly"] = true
			oc.ApplyTemplate(t, meshNamespace, meshTmpl, smcpValues)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			// SMCP v2.5 no longer supports APPLY_WASM_PLUGINS_TO_INBOUND_ONLY
			if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_5) {
				t.LogStep("Verify that a request from sleep to httpbin returns 403, for SMCP v2.5+")
				app.AssertSleepPodRequestForbidden(t, ns.Foo, httpbinUrl, CurlOpts)

			} else {
				t.LogStep("Verify that a request from sleep to httpbin returns 200")
				app.AssertSleepPodRequestSuccess(t, ns.Foo, httpbinUrl, CurlOpts)
			}
		}

		if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_4) {
			t.LogStep("Disable APPLY_WASM_PLUGINS_TO_INBOUND_ONLY and make sure that 403 is returned again")
			smcpValues["ApplyWasmPluginsToInboundOnly"] = false
			oc.ApplyTemplate(t, meshNamespace, meshTmpl, smcpValues)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			t.LogStep("Verify that a request from sleep to httpbin returns 403")
			app.AssertSleepPodRequestForbidden(t, ns.Foo, httpbinUrl, CurlOpts)

			t.LogStep("Enable SERVER mode in the WASM plugin and check if returns 200")
			oc.ApplyTemplate(t, ns.Foo, wasmPluginTmpl, map[string]interface{}{
				"AppLabel":   "sleep",
				"ServerMode": true,
			})

			t.LogStep("Verify that a request from sleep to httpbin returns 200")
			app.AssertSleepPodRequestSuccess(t, ns.Foo, httpbinUrl, CurlOpts)
		}
	})
}

//fmt.Sprintf(`curl http://httpbin:8000/headers -H "Authorization: Bearer %s" -s -o /dev/null -w "%%{http_code}"`, token),
