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

	bookinfoServerCert    = "sampleCerts/bookinfo.com/bookinfo.com.crt"
	bookinfoServerCertKey = "sampleCerts/bookinfo.com/bookinfo.com.key"
	bookinfoSampleCACert  = "sampleCerts/bookinfo.com/example.com.crt"

	bookinfoMetrics = "samples/bookinfo/telemetry/metrics.yaml"

	httpbinYaml       = "samples/httpbin/httpbin.yaml"
	httpbinFortioYaml = "samples/httpbin/sample-client/fortio-deploy.yaml"
	httpbinLegacyYaml = "samples_legacy/httpbin.yaml"

	httpbinSampleServerCertKey = "sampleCerts/httpbin.example.com/httpbin.example.com.key"
	httpbinSampleServerCert    = "sampleCerts/httpbin.example.com/httpbin.example.com.crt"
	httpbinSampleCACert        = "sampleCerts/httpbin.example.com/example.com.crt"
	httpbinSampleClientCert    = "sampleCerts/httpbin.example.com/httpbin-client.example.com.crt"
	httpbinSampleClientCertKey = "sampleCerts/httpbin.example.com/httpbin-client.example.com.key"

	helloworldServerCertKey = "sampleCerts/helloworldv1/helloworld-v1.example.com.key"
	helloworldServerCert    = "sampleCerts/helloworldv1/helloworld-v1.example.com.crt"

	echoYaml      = "samples/tcp-echo/tcp-echo-services.yaml"
	echoAllv1Yaml = "samples/tcp-echo/tcp-echo-all-v1.yaml"
	echo20v2Yaml  = "samples/tcp-echo/tcp-echo-20-v2.yaml"
	echoWithProxy = "samples/tcp-echo/tcp-echo.yaml"

	sleepYaml       = "samples/sleep/sleep.yaml"
	sleepLegacyYaml = "samples_legacy/sleep.yaml"

	nginxYaml          = "samples/https/nginx-app.yaml"
	nginxNoSidecarYaml = "samples_legacy/nginx-app.yaml"
	nginxDefaultConfig = "samples/https/default.conf"

	nginxServerCertKey = "sampleCerts/nginx.example.com/nginx.example.com.key"
	nginxServerCert    = "sampleCerts/nginx.example.com/nginx.example.com.crt"
	nginxServerCACert  = "sampleCerts/nginx.example.com/example.com.crt"
	nginxClientCertKey = "sampleCerts/nginx.example.com/nginx-client.example.com.key"
	nginxClientCert    = "sampleCerts/nginx.example.com/nginx-client.example.com.crt"

	caCert      = "sampleCerts/ca-cert.pem"
	caCertKey   = "sampleCerts/ca-key.pem"
	caRootCert  = "sampleCerts/root-cert.pem"
	caCertChain = "sampleCerts/cert-chain.pem"

	keyvaltemplate = "samples/httpbin/policy/keyval-template.yaml"
	keyvalYaml     = "samples/httpbin/policy/keyval.yaml"
	keyvalImage    = "gcr.io/istio-testing/keyval:release-1.1"

	mixerRuleProductpageRateLimit = "samples/bookinfo/policy/mixer-rule-productpage-ratelimit.yaml"
	mixerRuleDenyLabel            = "samples/bookinfo/policy/mixer-rule-deny-label.yaml"
	mixerRuleDenyWhitelist        = "samples/bookinfo/policy/mixer-rule-deny-whitelist.yaml"
	mixerRuleDenyIP               = "samples/bookinfo/policy/mixer-rule-deny-ip.yaml"

	kubeconfig    = ""
	testNamespace = "bookinfo"
	testUsername  = "jason"
	waitTime      = 5

	jmeterURL = "https://mirrors.ocf.berkeley.edu/apache//jmeter/binaries/apache-jmeter-5.3.tgz"

	meshNamespace     = "istio-system"
	smcpName          = "basic"
	smcpv1API         = "smcp.v1.maistra.io"
	invalidSMCPFields = "config/smcp-invalid-fields/smcp.yaml"
	goldPandaResource = "config/goldPanda/resource.yaml"

	excludeOutboundPortsAnnotation = "config/excludeOutboundPortsAnnotation/app-v2.yaml"

	mustGatherImage = "registry.redhat.io/openshift-service-mesh/istio-must-gather-rhel7"
	smmrTest        = "config/smmrTest.yaml"
)

var (
	// OCP4.x
	gatewayHTTP, _       = util.ShellSilent(`kubectl get routes -n %s istio-ingressgateway -o jsonpath='{.spec.host}'`, meshNamespace)
	ingressHTTPPort, _   = util.ShellSilent(`kubectl -n %s get service %s -o jsonpath='{.spec.ports[?(@.name=="http2")].port}'`, meshNamespace, "istio-ingressgateway")
	secureIngressPort, _ = util.ShellSilent(`kubectl -n %s get service %s -o jsonpath='{.spec.ports[?(@.name=="https")].port}'`, meshNamespace, "istio-ingressgateway")
)
