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
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestExternalCertificate(t *testing.T) {
	test.NewTest(t).Id("T17").Groups(test.Full, test.ARM, test.InterOp).Run(func(t test.TestHelper) {
		const ns = "bookinfo"

		t.Cleanup(func() {
			t.Logf("Recreate namespace %s", ns)
			oc.RecreateNamespace(t, ns, meshNamespace)
		})

		meshValues := map[string]interface{}{
			"Name":    smcpName,
			"Version": env.GetSMCPVersion().String(),
			"ROSA":    ROSA,
		}

		t.LogStep("Uninstall existing SMCP")
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Create cacerts secret")
		oc.CreateGenericSecretFromFiles(t, meshNamespace, "cacerts",
			"ca-cert.pem="+sampleCACert,
			"ca-key.pem="+sampleCAKey,
			"root-cert.pem="+sampleCARoot,
			"cert-chain.pem="+sampleCAChain)
		rootCert := readPemCertificatesFromFile(t, sampleCARoot)[0]
		chainCerts := readPemCertificatesFromFile(t, sampleCAChain)

		t.LogStep("Apply SMCP to configure certificate authority to use cacerts secret")
		oc.ApplyTemplate(t, meshNamespace, SMCPWithCustomCA, meshValues)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Install bookinfo")
		app.InstallAndWaitReady(t, app.BookinfoWithMTLS(ns))

		t.LogStep("Wait for response from productpage app")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t,
				app.BookinfoProductPageURL(t, meshNamespace), nil,
				assert.ResponseStatus(200))
		})

		t.LogStep("Retrieve certificates from bookinfo details service")

		var returnedCerts []*x509.Certificate
		retry.UntilSuccess(t, func(t test.TestHelper) {
			opensslOutput := oc.Exec(t,
				pod.MatchingSelector("app=productpage", ns), "istio-proxy",
				`openssl s_client -showcerts -connect details:9080 || true`) // the "|| true" is needed until oc.Exec can ignore return codes
			returnedCerts = readPemCertificatesFromText(t, opensslOutput)
		})
		certToVerify := returnedCerts[0]
		otherCerts := returnedCerts[1:]

		verifyContainsCerts(t, otherCerts, chainCerts,
			"The cert-chain certificates are present in the certificates sent by the tested service",
			"The cert-chain certificates were not found in the certificates sent by the tested service")

		verifyCertificate(t, certToVerify, []*x509.Certificate{rootCert}, otherCerts,
			"Certificate trust chain successfully verified",
			"Unable to verify certificate trust chain")
	})
}

func verifyContainsCerts(t test.TestHelper, actualCerts, expectedCerts []*x509.Certificate, successMsg, failureMsg string) {
	t.T().Helper()

	// Make a copy of expectedCerts because we will clobber it
	certsToFind := make([]*x509.Certificate, len(expectedCerts), len(expectedCerts))
	copy(certsToFind, expectedCerts)

	for _, certA := range actualCerts {
		for idxB, certB := range certsToFind {
			if certA.Equal(certB) {
				if len(certsToFind) == 1 {
					t.LogSuccess(successMsg)
					return
				}
				// "Remove" certB from certsToFind
				certsToFind[idxB] = certsToFind[len(certsToFind)-1] // overwrite certB with last element
				certsToFind[len(certsToFind)-1] = nil               // remove stale ref to last element
				certsToFind = certsToFind[:len(certsToFind)-1]      // resize slice
				break
			}
		}
	}
	t.Error(failureMsg)
}

func verifyCertificate(t test.TestHelper, cert *x509.Certificate, rootCerts, intermediateCerts []*x509.Certificate, successMsg, failureMsg string) {
	t.T().Helper()
	rootCertPool := x509.NewCertPool()
	for _, c := range rootCerts {
		rootCertPool.AddCert(c)
	}

	intermediateCertPool := x509.NewCertPool()
	for _, c := range intermediateCerts {
		intermediateCertPool.AddCert(c)
	}

	chains, err := cert.Verify(x509.VerifyOptions{
		Roots:         rootCertPool,
		Intermediates: intermediateCertPool})

	if err != nil {
		// x509.UnknownAuthorityError will be generated during a non-exceptional failure and can be ignored here
		_, isUnkAuthErr := err.(x509.UnknownAuthorityError)
		if !isUnkAuthErr {
			t.Fatalf(`Error while verifying certificate: %v`, err)
		}
	}

	if len(chains) > 0 {
		t.LogSuccess(successMsg)
	} else {
		t.Error(failureMsg)
	}
}

func readPemCertificatesFromFile(t test.TestHelper, path string) []*x509.Certificate {
	t.T().Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(`Error reading certificates: Unable to open "%s" for reading: %v`, path, err)
	}
	return readPemCertificates(t, bytes)
}

func readPemCertificatesFromText(t test.TestHelper, text string) []*x509.Certificate {
	t.T().Helper()

	return readPemCertificates(t, []byte(text))
}

func readPemCertificates(t test.TestHelper, pemData []byte) []*x509.Certificate {
	t.T().Helper()
	var certificates []*x509.Certificate
	for {
		block, rest := pem.Decode(pemData)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("failed to parse certificate: %w", err)
		}
		certificates = append(certificates, cert)
		pemData = rest
	}
	if len(certificates) == 0 {
		t.Fatal("Failed to read certificates: No certificates found in text.")
	}
	return certificates
}

const (
	SMCPWithCustomCA = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: {{ .Version }}
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
    dataPlane:
      mtls: true
    certificateAuthority:
      type: Istiod
      istiod:
        type: PrivateKey
        privateKey:
          rootCADir: /etc/cacerts
  tracing:
    type: None
  {{ if .ROSA }} 
  security:
    identity:
      type: ThirdParty
  {{ end }}
---
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  - bookinfo`
)
