// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package certificate

import (
	"fmt"
	"os"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestExternalCertificate(t *testing.T) {
	test.NewTest(t).Id("T17").Groups(test.Full, test.ARM, test.InterOp).Run(func(t test.TestHelper) {
		const ns = "bookinfo"

		t.Log("Testing external certificates.")

		var tmpDir = ""

		t.Cleanup(func() {

			if tmpDir != "" {
				t.Log("Removing temporary directory")
				os.RemoveAll(tmpDir)
			}

			t.Log("Recreating bookinfo namespace.")
			oc.RecreateNamespace(t, ns)

			t.Log("Undoing SMCP patch.")
			oc.Patch(t, meshNamespace, "smcp", smcpName, "json", CertSMCPPathRemovalPatch)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
		})

		t.LogStep("Adding external CA.")
		oc.CreateGenericSecretFromFiles(t, meshNamespace, "cacerts",
			fmt.Sprintf("ca-cert.pem=%s", sampleCACert),
			fmt.Sprintf("ca-key.pem=%s", sampleCAKey),
			fmt.Sprintf("root-cert.pem=%s", sampleCARoot),
			fmt.Sprintf("cert-chain.pem=%s", sampleCAChain))

		t.LogStep("Enabling control plane MTLS.")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", CertSMCPPath)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Installing bookinfo.")
		app.InstallAndWaitReady(t, app.BookinfoWithMTLS(ns))

		if env.GetSampleArch() == "p" || env.GetSampleArch() == "z" {

			t.Log("Running Power/Z environment test.")

			t.LogStep("Checking response from productpage.")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				curl.Request(t, app.BookinfoProductPageURL(t, meshNamespace), curl.NilRequestOption{}, assert.ResponseStatus(200))
			})

		} else {

			t.LogStep("Retrieving certificates.")
			tmpDir = shell.CreateTempDir(t, "cacerts")
			oc.Exec(t, pod.MatchingSelector("app=productpage", ns), "istio-proxy",
				fmt.Sprintf(`openssl s_client -showcerts -connect details:9080 > '%s/bookinfo-proxy-cert.txt' || true`, tmpDir))

			t.LogStep("Extracting certificates from response.")
			shell.Executef(t,
				`sed -n '/-----BEGIN CERTIFICATE-----/{:start /-----END CERTIFICATE-----/!{N;b start};/.*/p}' '%s/bookinfo-proxy-cert.txt' > '%s/certs.pem'`,
				tmpDir, tmpDir)
			shell.Executef(t,
				`awk 'BEGIN {counter=0;} /BEGIN CERT/{counter++} { print > "%s/proxy-cert-" counter ".pem"}' < '%s/certs.pem'`,
				tmpDir, tmpDir)

			t.LogStep("Verifying certificates.")
			t.NewSubTest("Verify root certificate.").Run(func(t test.TestHelper) {

				shell.Executef(t, `openssl x509 -in '%s' -text -noout > '%s/root-cert.crt.txt'`, sampleCARoot, tmpDir)
				shell.Executef(t, `openssl x509 -in '%s/proxy-cert-3.pem' -text -noout > '%s/pod-root-cert.crt.txt'`, tmpDir, tmpDir)

				if err := util.CompareFiles(fmt.Sprintf("%s/root-cert.crt.txt", tmpDir), fmt.Sprintf("%s/pod-root-cert.crt.txt", tmpDir)); err != nil {
					t.Errorf("Root certs do not match: %v", err)
				}
			})

			t.NewSubTest("Verify CA certificate.").Run(func(t test.TestHelper) {

				shell.Executef(t, `openssl x509 -in '%s' -text -noout > '%s/ca-cert.crt.txt'`, sampleCACert, tmpDir)
				shell.Executef(t, `openssl x509 -in '%s/proxy-cert-2.pem' -text -noout > '%s/pod-cert-chain-ca.crt.txt'`, tmpDir, tmpDir)

				if err := util.CompareFiles(fmt.Sprintf("%s/ca-cert.crt.txt", tmpDir), fmt.Sprintf("%s/pod-cert-chain-ca.crt.txt", tmpDir)); err != nil {
					t.Errorf("CA certs do not match: %v", err)
				}
			})

			t.LogStep("Verifying certificate chain.")
			t.NewSubTest("Verify certificate chain.").Run(func(t test.TestHelper) {

				shell.Executef(t, `cat '%s' '%s' > '%s/sample-cert-and-root-cert.pem'`, sampleCACert, sampleCARoot, tmpDir)
				shell.Execute(t,
					fmt.Sprintf(`openssl verify -CAfile '%s/sample-cert-and-root-cert.pem' '%s/proxy-cert-1.pem'`, tmpDir, tmpDir),
					assert.OutputContains(fmt.Sprintf("%s/proxy-cert-1.pem: OK", tmpDir),
						"Certificate chain verified.",
						"Certificate chain could not be verified."))
			})
		}
	})
}
