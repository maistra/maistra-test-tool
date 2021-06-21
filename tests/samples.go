// Copyright 2020 Red Hat, Inc.
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

package tests

import "maistra/util"

const (
	bookinfoYaml           = "samples/bookinfo/platform/kube/bookinfo.yaml"
	bookinfoGateway        = "samples/bookinfo/networking/bookinfo-gateway.yaml"
	bookinfoRuleAllYaml    = "samples/bookinfo/networking/destination-rule-all.yaml"
	bookinfoRuleAllTLSYaml = "samples/bookinfo/networking/destination-rule-all-mtls.yaml"
	bookinfoAllv1Yaml      = "samples/bookinfo/networking/virtual-service-all-v1.yaml"
	bookinfoReviewV2Yaml   = "samples/bookinfo/networking/virtual-service-reviews-test-v2.yaml"
	bookinfoReview50V3Yaml = "samples/bookinfo/networking/virtual-service-reviews-50-v3.yaml"
	bookinfoReviewV3Yaml   = "samples/bookinfo/networking/virtual-service-reviews-v3.yaml"
	bookinfoReviewv2v3Yaml = "samples/bookinfo/networking/virtual-service-reviews-jason-v2-v3.yaml"

	bookinfoRatingDelayYaml = "samples/bookinfo/networking/virtual-service-ratings-test-delay.yaml"
	bookinfoRatingAbortYaml = "samples/bookinfo/networking/virtual-service-ratings-test-abort.yaml"

	bookinfoDBYaml = "samples/bookinfo/platform/kube/bookinfo-db.yaml"

	bookinfoServerCert    = "samples/certs/bookinfo.com/bookinfo.com.crt"
	bookinfoServerCertKey = "samples/certs/bookinfo.com/bookinfo.com.key"
	bookinfoSampleCACert  = "samples/certs/bookinfo.com/example.com.crt"

	bookinfoMetrics = "samples/bookinfo/telemetry/metrics.yaml"

	httpbinYaml       = "samples/httpbin/httpbin-1.1.yaml"
	httpbinFortioYaml = "samples/httpbin/sample-client/fortio-deploy.yaml"
	httpbinLegacyYaml = "samples/httpbin/httpbin-1.1-legacy.yaml"

	httpbinSampleServerCertKey = "samples/certs/httpbin.example.com/httpbin.example.com.key"
	httpbinSampleServerCert    = "samples/certs/httpbin.example.com/httpbin.example.com.crt"
	httpbinSampleCACert        = "samples/certs/httpbin.example.com/example.com.crt"
	httpbinSampleClientCert    = "samples/certs/httpbin.example.com/httpbin-client.example.com.crt"
	httpbinSampleClientCertKey = "samples/certs/httpbin.example.com/httpbin-client.example.com.key"

	echoYaml      = "samples/tcp-echo/tcp-echo-services.yaml"
	echoAllv1Yaml = "samples/tcp-echo/tcp-echo-all-v1.yaml"
	echo20v2Yaml  = "samples/tcp-echo/tcp-echo-20-v2.yaml"

	sleepYaml       = "samples/sleep/sleep.yaml"
	sleepLegacyYaml = "samples/sleep/sleep-legacy.yaml"

	nginxYaml          = "samples/https/nginx-app.yaml"
	nginxNoSidecarYaml = "samples/https/nginx-app-without-sidecar.yaml"
	nginxDefaultConfig = "samples/https/default.conf"

	nginxServerCertKey = "samples/certs/nginx.example.com/nginx.example.com.key"
	nginxServerCert    = "samples/certs/nginx.example.com/nginx.example.com.crt"
	nginxServerCACert  = "samples/certs/nginx.example.com/example.com.crt"
	nginxClientCertKey = "samples/certs/nginx.example.com/nginx-client.example.com.key"
	nginxClientCert    = "samples/certs/nginx.example.com/nginx-client.example.com.crt"

	caCert      = "samples/certs/ca-cert.pem"
	caCertKey   = "samples/certs/ca-key.pem"
	caRootCert  = "samples/certs/root-cert.pem"
	caCertChain = "samples/certs/cert-chain.pem"

	keyvaltemplate = "samples/httpbin/policy/keyval-template.yaml"
	keyvalYaml     = "samples/httpbin/policy/keyval.yaml"

	mixerRuleProductpageRateLimit = "samples/bookinfo/policy/mixer-rule-productpage-ratelimit.yaml"
	mixerRuleDenyLabel            = "samples/bookinfo/policy/mixer-rule-deny-label.yaml"
	mixerRuleDenyWhitelist        = "samples/bookinfo/policy/mixer-rule-deny-whitelist.yaml"
	mixerRuleDenyIP               = "samples/bookinfo/policy/mixer-rule-deny-ip.yaml"

	kubeconfig    = ""
	testNamespace = "bookinfo"
	testUsername  = "jason"
	waitTime      = 5
	// KIND
	//gatewayHTTP 			= "localhost:8001/api/v1/namespaces/istio-system/services/istio-ingressgateway:80/proxy"

	jmeterURL = "https://apache.osuosl.org//jmeter/binaries/apache-jmeter-5.3.tgz"

	meshNamespace = "istio-system"
	smcpName      = "basic"
	smcpAPI       = "smcp.v1.maistra.io"

	invalidLogging = "config/smcp-invalid-logging/smcp.yaml"
)

var (
	// OCP4.x
	gatewayHTTP, _       = util.ShellSilent("kubectl get routes -n %s istio-ingressgateway -o jsonpath='{.spec.host}'", meshNamespace)
	ingressHTTPPort, _   = util.ShellSilent("kubectl -n %s get service %s -o jsonpath='{.spec.ports[?(@.name==\"http2\")].port}'", meshNamespace, "istio-ingressgateway")
	secureIngressPort, _ = util.ShellSilent("kubectl -n %s get service %s -o jsonpath='{.spec.ports[?(@.name==\"https\")].port}'", meshNamespace, "istio-ingressgateway")
)
