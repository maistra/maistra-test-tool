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

package ingress

import (
	_ "embed"
	"fmt"
	"net/http"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	. "github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/request"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/httpbin-tls-gateway-https.yaml
	httpbinTLSGatewayHTTPS string

	//go:embed yaml/gateway-multiple-hosts.yaml
	gatewayMultipleHosts string

	//go:embed yaml/gateway-httpbin-mtls.yaml
	gatewayHttpbinMTLSYaml string

	//go:embed yaml/hello-world.yaml
	helloWorldYaml string

	helloWorldImages = map[string]string{
		"p":   "quay.io/maistra/helloworld-v1:0.0-ibm-p",
		"z":   "quay.io/maistra/helloworld-v1:0.0-ibm-z",
		"x86": "istio/examples-helloworld-v1",
	}
)

func TestSecureGateways(t *testing.T) {
	NewTest(t).Id("T9").Groups(Full, InterOp).Run(func(t TestHelper) {
		ns := "bookinfo"

		t.Cleanup(func() {
			oc.DeleteSecret(t, meshNamespace, "httpbin-credential")
			oc.DeleteSecret(t, meshNamespace, "helloworld-credential")
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t, app.Httpbin(ns))
		oc.ApplyString(t, ns, helloWorldYAML())
		oc.WaitDeploymentRolloutComplete(t, ns, "helloworld-v1")

		t.LogStep("Create TLS secrets")
		oc.CreateTLSSecret(t, meshNamespace, "httpbin-credential", httpbinSampleServerCertKey, httpbinSampleServerCert)
		oc.CreateTLSSecret(t, meshNamespace, "helloworld-credential", helloworldServerCertKey, helloworldServerCert)

		helloWorldURL := "https://helloworld-v1.example.com:" + secureIngressPort + "/hello"
		teapotURL := "https://httpbin.example.com:" + secureIngressPort + "/status/418"

		t.NewSubTest("tls_single_host").Run(func(t TestHelper) {
			t.LogStep("Configure a TLS ingress gateway for a single host")
			oc.ApplyString(t, ns, httpbinTLSGatewayHTTPS)

			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHTTP, secureIngressPort),
					assert.ResponseContains("-=[ teapot ]=-"),
				)
			})
		})

		t.NewSubTest("tls_multiple_hosts").Run(func(t TestHelper) {
			t.LogStep("configure Gateway with multiple TLS hosts")
			oc.ApplyString(t, ns, gatewayMultipleHosts)

			t.LogStep("check if helloworld-v1 responds with 200 OK")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					helloWorldURL,
					request.WithTLS(httpbinSampleCACert, "helloworld-v1.example.com", gatewayHTTP, secureIngressPort),
					assert.ResponseStatus(http.StatusOK))
			})

			t.LogStep("check if httpbin responds with teapot")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHTTP, secureIngressPort),
					assert.ResponseContains("-=[ teapot ]=-"))
			})
		})

		t.NewSubTest("mutual_tls").Run(func(t TestHelper) {
			t.LogStep("configure Gateway with tls.mode=Mutual")
			oc.CreateGenericSecretFromFiles(t, meshNamespace, "httpbin-credential",
				"tls.key="+httpbinSampleServerCertKey,
				"tls.crt="+httpbinSampleServerCert,
				"ca.crt="+httpbinSampleCACert)
			oc.ApplyString(t, ns, gatewayHttpbinMTLSYaml)

			t.LogStep("check if SSL handshake fails when no client certificate is given")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHTTP, secureIngressPort),
					assert.RequestFails(
						"request failed as expected",
						"expected request to fail because no client certificate was provided"))
			})

			t.LogStep("check if SSL handshake succeeds when client certificate is given")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.
						WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHTTP, secureIngressPort).
						WithClientCertificate(httpbinSampleClientCert, httpbinSampleClientCertKey),
					assert.ResponseContains("-=[ teapot ]=-"))
			})
		})
	})
}

func helloWorldYAML() string {
	arch := Getenv("SAMPLEARCH", "x86")
	image := helloWorldImages[arch]
	if image == "" {
		panic(fmt.Sprintf("unsupported SAMPLEARCH: %s", arch))
	}

	return fmt.Sprintf(helloWorldYaml, image)
}
