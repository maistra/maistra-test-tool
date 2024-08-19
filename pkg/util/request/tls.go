// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package request

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"
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
	caCert, err := os.ReadFile(m.caCertFile)
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
