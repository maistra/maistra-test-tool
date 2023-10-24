package extensions

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/tasks/security/authentication"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/request"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"

	"testing"
)

func TestThreeScaleWasmPluginIngressGateway(t *testing.T) {
	test.NewTest(t).Groups(test.Full).Run(func(t test.TestHelper) {
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Foo)
			oc.RecreateNamespace(t, meshNamespace)
			oc.DeleteNamespace(t, threeScaleNs)
		})

		meshValues := map[string]string{
			"Name":    smcpName,
			"Version": env.GetSMCPVersion().String(),
			"Member":  ns.Foo,
		}

		t.LogStep("Deploying SMCP")
		oc.ApplyTemplate(t, meshNamespace, meshTmpl, meshValues)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Deploying 3scale mocks")
		oc.CreateNamespace(t, threeScaleNs)
		oc.ApplyString(t, threeScaleNs, threeScaleBackend)
		oc.ApplyString(t, meshNamespace, threeScaleBackendSvcEntry)
		oc.ApplyString(t, threeScaleNs, threeScaleSystem)
		oc.ApplyString(t, meshNamespace, threeScaleSystemSvcEntry)
		oc.WaitAllPodsReady(t, threeScaleNs)

		t.LogStep("Configuring authz and authn")
		oc.ApplyString(t, meshNamespace, authentication.JWTAuthPolicyForIngressGateway)

		t.LogStep("Applying 3scale WASM plugin to the ingress gateway")
		oc.ApplyString(t, meshNamespace, wasmPluginIngressGateway)

		t.LogStep("Deploying httpbin")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))
		oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/httpbin/httpbin-gateway.yaml")

		t.LogStep("Verify that a request with a valid token returns 200")
		ingressGatewayHost := istio.GetIngressGatewayHost(t, meshNamespace)
		headersURL := fmt.Sprintf("http://%s/headers", ingressGatewayHost)
		token := string(curl.Request(t, "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/demo.jwt", nil))
		token = strings.Trim(token, "\n")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, headersURL, request.WithHeader("Authorization", "Bearer "+token), require.ResponseStatus(http.StatusOK))
		})
	})
}
