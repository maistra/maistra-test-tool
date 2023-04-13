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

		ns := "foo"
		cert := "cert-manager"
		t.Cleanup(func() {
			oc.DeleteNamespace(t, cert)
			oc.RecreateNamespace(t, ns)
		})

		t.LogStep("uninstall the SMCP")
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Add jetstach repo to helm")
		shell.Execute(t, `helm repo add jetstack https://charts.jetstack.io`)

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
		oc.ApplyString(t, meshNamespace, CertManagerSMCP)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Install httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(ns), app.Sleep(ns))

		t.LogStep("Check if httpbin returns 200 OK ")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app=sleep", ns),
				"sleep",
				fmt.Sprintf(`curl http://httpbin.foo:8000/ip -s -o /dev/null -w "%%%%{http_code}\n"`),
				assert.OutputContains(
					"200",
					"Got expected 200 OK from httpbin",
					"Expected 200 OK from httpbin, but got a different HTTP code"))
		})

	})

}

const (
	SelfSignedCa = `
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: istio-ca
spec:
  isCA: true
  duration: 2160h # 90d
  secretName: istio-ca
  commonName: istio-ca
  subject:
    organizations:
      - cluster.local
      - cert-manager
  issuerRef:
    name: selfsigned
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: istio-ca
spec:
  ca:
    secretName: istio-ca
 `

	CertManagerSMCP = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: basic
spec:
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  security:
    certificateAuthority:
      cert-manager:
        address: cert-manager-istio-csr.cert-manager.svc:443
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
  - foo
 `
)
