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
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupExternalCert() {
	util.Log.Info("Cleanup")

	bookinfo := examples.Bookinfo{Namespace: "bookinfo"}
	bookinfo.Uninstall()

	util.Shell(`kubectl -n %s delete secret cacerts`, meshNamespace)
	util.Shell(`kubectl -n %s patch smcp/%s --type=json -p='[{"op": "remove", "path": "/spec/security/certificateAuthority"}, {"op": "remove", "path": "/spec/security/dataPlane"}]'`, meshNamespace, smcpName)
	util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcpName)
	time.Sleep(time.Duration(60) * time.Second)
}

func TestExternalCert(t *testing.T) {
	defer cleanupExternalCert()

	util.Log.Info("Test External Certificates")
	util.Log.Info("Enable Control Plane MTLS")

	t.Run("Security_plugging_external_cert_test", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Adding an external CA")
		util.Shell(`kubectl create -n %s secret generic cacerts --from-file=%s --from-file=%s --from-file=%s --from-file=%s`,
			meshNamespace, sampleCACert, sampleCAKey, sampleCARoot, sampleCAChain)

		util.Shell(`kubectl -n %s patch smcp/%s --type=merge --patch="%s"`, meshNamespace, smcpName, CertSMCPPath)
		util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcpName)
		time.Sleep(time.Duration(60) * time.Second)

		bookinfo := examples.Bookinfo{Namespace: "bookinfo"}
		bookinfo.Install(true)

		productPod, err := util.GetPodName("bookinfo", "app=productpage")
		util.Inspect(err, "Failed to get productpage pod name", "", t)

		tmpDir, err := ioutil.TempDir("", "cacerts")
		util.Inspect(err, "Failed to create temp dir", "", t)
		defer os.RemoveAll(tmpDir)

		if util.Getenv("SAMPLEARCH", "x86") == "p" || util.Getenv("SAMPLEARCH", "x86") == "z" {
			gatewayHTTP, _ := util.ShellSilent(`kubectl get routes -n %s istio-ingressgateway -o jsonpath='{.spec.host}'`, meshNamespace)
			productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "Failed to get HTTP Response", "", t)
			defer util.CloseResponseBody(resp)
		} else {
			util.Log.Info("Verify the new certificates")

			// Generate the cert files
			util.ShellMuteOutput(`oc -n bookinfo exec %s -c istio-proxy -- openssl s_client -showcerts -connect details:9080 > %s/bookinfo-proxy-cert.txt`, productPod, tmpDir)
			_, err = util.ShellMuteOutput(`sed -n '/-----BEGIN CERTIFICATE-----/{:start /-----END CERTIFICATE-----/!{N;b start};/.*/p}' %s/bookinfo-proxy-cert.txt > %s/certs.pem`, tmpDir, tmpDir)
			util.Inspect(err, "Failed to parse 'openssl s_client' output", "", t)
			_, err = util.ShellMuteOutput(`awk 'BEGIN {counter=0;} /BEGIN CERT/{counter++} { print > "%s/proxy-cert-" counter ".pem"}' < %s/certs.pem`, tmpDir, tmpDir)
			util.Inspect(err, "Failed to split certs into separate files", "", t)

			// Compare them with the original certs
			util.Log.Info("Verifying the root certificate")
			_, err = util.ShellMuteOutput(`openssl x509 -in %s -text -noout > %s/root-cert.crt.txt`, sampleCARoot, tmpDir)
			util.Inspect(err, "Failed to print cert", "", t)
			_, err = util.ShellMuteOutput(`openssl x509 -in %s/proxy-cert-3.pem -text -noout > %s/pod-root-cert.crt.txt`, tmpDir, tmpDir)
			util.Inspect(err, "Failed to print cert", "", t)
			if err := util.CompareFiles(fmt.Sprintf("%s/root-cert.crt.txt", tmpDir), fmt.Sprintf("%s/pod-root-cert.crt.txt", tmpDir)); err != nil {
				t.Errorf("Root certs do not match: %v", err)
			}

			util.Log.Info("Verifying the CA certificate")
			_, err = util.ShellMuteOutput(`openssl x509 -in %s -text -noout > %s/ca-cert.crt.txt`, sampleCACert, tmpDir)
			util.Inspect(err, "Failed to print cert", "", t)
			_, err = util.ShellMuteOutput(`openssl x509 -in %s/proxy-cert-2.pem -text -noout > %s/pod-cert-chain-ca.crt.txt`, tmpDir, tmpDir)
			util.Inspect(err, "Failed to print cert", "", t)
			if err := util.CompareFiles(fmt.Sprintf("%s/ca-cert.crt.txt", tmpDir), fmt.Sprintf("%s/pod-cert-chain-ca.crt.txt", tmpDir)); err != nil {
				t.Errorf("CA certs do not match: %v", err)
			}

			util.Log.Info("Verifying the certificate chain")
			output, err := util.ShellMuteOutput(`/bin/bash -c "openssl verify -CAfile <(cat %s %s) %s/proxy-cert-1.pem"`, sampleCACert, sampleCARoot, tmpDir)
			util.Inspect(err, "Failed to verify the certificate chain", "", t)
			expected := []byte(fmt.Sprintf("%s/proxy-cert-1.pem: OK", tmpDir))
			if err := util.Compare([]byte(output), expected); err != nil {
				t.Errorf("unexpected output while verifying cert chain: %v", err)
			}
		}
	})
}
