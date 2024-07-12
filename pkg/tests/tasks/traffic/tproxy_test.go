package traffic

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestTproxy(t *testing.T) {
	NewTest(t).Id("T7").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		if env.GetSMCPVersion().LessThan(version.SMCP_2_5) {
			t.Skip("TPROXY is only supported in 2.5.3 and newer versions")
		}

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Foo)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Add privileged SCC to the app namespace")
		shell.Executef(t, "oc adm policy add-scc-to-group privileged system:serviceaccounts:%s", ns.Foo)

		t.LogStep("Install httpbin and sleep in tproxy mode")
		app.InstallAndWaitReady(t, app.HttpbinTproxy(ns.Foo), app.SleepTroxy(ns.Foo))

		t.NewSubTest("HTTP request from ingress gateway to httpbin in tproxy mode").Run(func(t TestHelper) {
			oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.6/samples/httpbin/httpbin-gateway.yaml")
			httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
			})
		})

		t.NewSubTest("HTTP request from tproxy sleep to tproxy httpbin").Run(func(t TestHelper) {
			app.ExecInSleepPod(t, ns.Foo,
				"curl http://httpbin.foo:8000/headers -s -o /dev/null -w %{http_code}",
				assert.OutputContains("200", "Request succeeded", "Unexpected response"))
		})
	})
}
