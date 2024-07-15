package injection

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestNativeSidecars(t *testing.T) {
	NewTest(t).Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		if version.ParseVersion(oc.GetOCPVersion(t)).LessThan(version.OCP_4_16) || env.GetSMCPVersion().LessThan(version.SMCP_2_6) {
			t.Skip("Native sidecars are only supported in OpenShift 4.16+ and OSSM 2.6+")
		}

		meshValues := map[string]interface{}{
			"Member":                ns.Foo,
			"NativeSidecarsEnabled": true,
			"Rosa":                  env.IsRosa(),
			"Version":               env.GetSMCPVersion().String(),
		}

		t.Cleanup(func() {
			oc.DeleteFromTemplate(t, meshNamespace, meshTmpl, meshValues)
			oc.RecreateNamespace(t, ns.Foo)
		})

		t.LogStep("Deploying SMCP")
		oc.ApplyTemplate(t, meshNamespace, meshTmpl, meshValues)
		oc.WaitSMCPReady(t, meshNamespace, "basic")

		t.LogStep("Install httpbin and sleep in mode")
		app.InstallAndWaitReady(t, app.HttpbinTproxy(ns.Foo), app.SleepTroxy(ns.Foo))

		t.NewSubTest("HTTP request from ingress gateway to httpbin in mode").Run(func(t TestHelper) {
			oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.6/samples/httpbin/httpbin-gateway.yaml")
			httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
			})
		})

		t.NewSubTest("HTTP request from sleep to httpbin").Run(func(t TestHelper) {
			app.ExecInSleepPod(t, ns.Foo,
				"curl http://httpbin.foo:8000/headers -s -o /dev/null -w %{http_code}",
				assert.OutputContains("200", "Request succeeded", "Unexpected response"))
		})
	})
}
