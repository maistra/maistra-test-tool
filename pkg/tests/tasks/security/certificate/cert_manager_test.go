package certificate

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestCertManager(t *testing.T) {
	test.NewTest(t).Id("T38").Groups(test.Full, test.ARM, test.InterOp).Run(func(t test.TestHelper) {

		ns := "bookinfo"
		cert := "cert-manager"
		t.Cleanup(func() {
			oc.DeleteNamespace(t, cert)
			oc.RecreateNamespace(t, ns)
		})

		t.LogStep("uninstall the SMCP")
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Install cert-manager")
		shell.Execute(t,
			fmt.Sprintf(`helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --version v1.11.0 --set installCRDs=true`),
			assert.OutputContains("cert-manager v1.11.0 has been deployed successfully",
				"Successfully installed cert-manager",
				"Failed to installed cert-manager"))

		t.LogStep("Provision certificates")
		oc.ApplyString(t, meshNamespace, SelfSignedCa)

		t.LogStep("Install cert-manager istio-csr Service")
		shell.Execute(t,
			fmt.Sprintf(`helm install -n cert-manager cert-manager-istio-csr jetstack/cert-manager-istio-csr -f https://raw.githubusercontent.com/maistra/istio-operator/maistra-2.4/deploy/examples/cert-manager/istio-csr-helm-values.yaml`),
			assert.OutputContains("STATUS: deployed",
				"Successfully, installed cert-manager-istio-csr",
				"Failed to installed cert-manager-istio-csr"))

		t.LogStep("deploy the cert-manager in SMCP")
		oc.ApplyString(t, meshNamespace, certManagerSMCP)

		t.LogStep("Install bookinfo app")
		app.InstallAndWaitReady(t, app.Bookinfo(ns))

		t.NewSubTest("Validation of Cert-Manager").Run(func(t test.TestHelper) {

			t.LogStep("Check istiod certificate")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=productpage", "bookinfo"),
					"istio-proxy",
					fmt.Sprint(`openssl s_client -CAfile /var/run/secrets/istio/root-cert.pem -showcerts -connect istiod-basic.istio-system:15012`),
					assert.OutputContains(
						"Verify return code: 0 (ok)",
						"Successfully verified the istiod certificate",
						"Failed to verified the istiod certificate"))
			})

			t.LogStep("Check app certificate")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=productpage", "bookinfo"),
					"istio-proxy",
					fmt.Sprint(`openssl s_client -CAfile /var/run/secrets/istio/root-cert.pem -showcerts -connect details.bookinfo:9080`),
					assert.OutputContains(
						"Verify return code: 0 (ok)",
						"Successfully verified the Check app certificate",
						"Failed to verified the Check app certificate"))
			})

		})

	})

}
