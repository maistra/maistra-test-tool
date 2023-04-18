// Copyright 2023 Red Hat, Inc.
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

package cert

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTLSCertConfig(t *testing.T) {
	// create new ca and server certificate
	newCA := new(CertBuilder).NewCACert()
	caCert, caPEM, caPrivateKey := newCA.GetCert(), newCA.GetCertPEM(), newCA.GetPrivateKey()

	newServer := new(CertBuilder).NewServerCert(caCert, caPrivateKey)
	certPEM, certPrivateKeyPEM := newServer.GetCertPEM(), newServer.GetPrivateKeyPEM()

	serverCert, err := tls.X509KeyPair(certPEM, certPrivateKeyPEM)
	if err != nil {
		t.Errorf("Failed to configure server cert:" + err.Error())
	}

	serverTLSConf := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caPEM)
	clientTLSConf := &tls.Config{
		RootCAs: certpool,
	}

	// set up the httptest.Server using our certificate signed by our CA
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "success!")
	}))
	server.TLS = serverTLSConf
	server.StartTLS()
	defer server.Close()

	// communicate with the server using an http.Client configured to trust our CA
	transport := &http.Transport{
		TLSClientConfig: clientTLSConf,
	}
	http := http.Client{
		Transport: transport,
	}
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Error(err)
	}

	// verify the response
	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	body := strings.TrimSpace(string(respBodyBytes[:]))
	if body == "success!" {
		fmt.Println("Got response success!")
		t.Logf("TestTLSCertConfig Pass: %s", body)
	} else {
		t.Errorf("Failed to get successful response.")
	}
}

func TestVerifyDCA(t *testing.T) {
	newCA := new(CertBuilder).NewCACert()
	caCert, caPrivateKey := newCA.GetCert(), newCA.GetPrivateKey()
	dcaCert := new(CertBuilder).NewServerCert(caCert, caPrivateKey).GetCert()
	err := verifyDCA(caCert, dcaCert)
	if err != nil {
		t.Errorf("Failed to verify certificate: Got error %s", err)
	}
}

func verifyDCA(root, dca *x509.Certificate) error {
	roots := x509.NewCertPool()
	roots.AddCert(root)
	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if _, err := dca.Verify(opts); err != nil {
		return fmt.Errorf("failed to verify certificate: " + err.Error())
	}
	fmt.Println("DCA verified")
	return nil
}
