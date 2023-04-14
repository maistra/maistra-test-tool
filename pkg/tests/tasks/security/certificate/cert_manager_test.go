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
		fooNs := "foo"

		t.Cleanup(func() {
			shell.ExecuteIgnoreError(t, "helm uninstall istio-csr -n istio-system")
			shell.ExecuteIgnoreError(t, "helm uninstall cert-manager -n cert-manager")
			oc.DeleteNamespace(t, "cert-manager")
			oc.RecreateNamespace(t, fooNs)
		})

		t.LogStep("Uninstall the SMCP")
		oc.RecreateNamespace(t, "istio-system")

		t.LogStep("Add jetstack repo to helm")
		shell.Execute(t, `helm repo add jetstack https://charts.jetstack.io`)

		t.LogStep("Install cert-manager")
		shell.Execute(t,
			"helm install cert-manager jetstack/cert-manager -n cert-manager --create-namespace --version v1.11.0 --set installCRDs=true",
			assert.OutputContains("cert-manager v1.11.0 has been deployed successfully",
				"Successfully installed cert-manager",
				"Failed to installed cert-manager"))

		t.LogStep("Provision root certificate")
		oc.ApplyFile(t, "cert-manager", "https://raw.githubusercontent.com/maistra/istio-operator/maistra-2.4/deploy/examples/cert-manager/istio-csr/selfsigned-ca.yaml")

		t.LogStep("Provision Istio certificate")
		oc.ApplyFile(t, "istio-system", "https://raw.githubusercontent.com/maistra/istio-operator/maistra-2.4/deploy/examples/cert-manager/istio-csr/istio-ca.yaml")

		t.LogStep("Install cert-manager-istio-csr")
		shell.Execute(t,
			"helm install istio-csr jetstack/cert-manager-istio-csr -n istio-system "+
				"-f https://raw.githubusercontent.com/maistra/istio-operator/maistra-2.4/deploy/examples/cert-manager/istio-csr/istio-csr.yaml",
			assert.OutputContains("STATUS: deployed",
				"Successfully, installed cert-manager-istio-csr",
				"Failed to installed cert-manager-istio-csr"))
		oc.WaitPodsReady(t, "istio-system", "app=cert-manager-istio-csr")

		t.LogStep("Deploy the cert-manager in SMCP")
		oc.ApplyString(t, "istio-system", createSMCPWithCertManager(smcpName, fooNs))
		oc.WaitSMCPReady(t, "istio-system", smcpName)

		t.LogStep("Install httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(fooNs), app.Sleep(fooNs))

		t.LogStep("Check if httpbin returns 200 OK ")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app=sleep", fooNs),
				"sleep",
				`curl http://httpbin:8000/ip -s -o /dev/null -w "%{http_code}"`,
				assert.OutputContains(
					"200",
					"Got expected 200 OK from httpbin",
					"Expected 200 OK from httpbin, but got a different HTTP code"))
		})

	})
}

func createSMCPWithCertManager(smcpName, memberNs string) string {
	return fmt.Sprintf(`apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: %s
spec:
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  gateways:
    egress:
      enabled: false
    openshiftRoute:
      enabled: false
  security:
    certificateAuthority:
      cert-manager:
        address: cert-manager-istio-csr.istio-system.svc:443
      type: cert-manager
    dataPlane:
      mtls: true
    identity:
      type: ThirdParty
  tracing:
    type: None
  version: v2.3
---
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  - %s
`, smcpName, memberNs)
}
