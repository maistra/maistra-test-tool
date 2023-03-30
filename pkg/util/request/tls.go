package request

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/curl"
)

func WithTLS(cacertFile string, host, ingressHost string, secureIngressPort string) TLSRequestOption {
	return TLSRequestOption{
		caCertFile:        cacertFile,
		host:              host,
		ingressHost:       ingressHost,
		secureIngressPort: secureIngressPort,
	}
}

type TLSRequestOption struct {
	caCertFile        string
	clientCertFile    string
	clientKeyFile     string
	host              string
	ingressHost       string
	secureIngressPort string
}

var _ curl.RequestOption = headerModifier{}

func (m TLSRequestOption) WithClientCertificate(clientCertFile, clientKeyFile string) TLSRequestOption {
	m.clientCertFile = clientCertFile
	m.clientKeyFile = clientKeyFile
	return m
}

func (m TLSRequestOption) ApplyToRequest(req *http.Request) error {
	req.Host = m.host
	return nil
}

func (m TLSRequestOption) ApplyToClient(client *http.Client) error {
	// Load CA cert
	caCert, err := ioutil.ReadFile(m.caCertFile)
	if err != nil {
		return err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS transport
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	// add client cert if provided
	if m.clientCertFile != "" {
		clientCert, err := tls.LoadX509KeyPair(m.clientCertFile, m.clientKeyFile)
		if err != nil {
			return err
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	tlsConfig.BuildNameToCertificate()

	// Custom DialContext
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == m.host+":"+m.secureIngressPort {
			addr = m.ingressHost + ":" + m.secureIngressPort
		}
		return dialer.DialContext(ctx, network, addr)
	}

	client.Transport = transport
	return nil
}
