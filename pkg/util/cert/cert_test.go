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

/* References:
https://gist.github.com/shaneutt/5e1995295cff6721c89a71d13a71c251
https://gist.github.com/Mattemagikern/328cdd650be33bc33105e26db88e487d
*/
