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

package main

import "maistra/util"

const (
	bookinfoYaml 			= "samples/bookinfo/platform/kube/bookinfo.yaml"
	bookinfoGateway			= "samples/bookinfo/networking/bookinfo-gateway.yaml"
	bookinfoRuleAllYaml    	= "samples/bookinfo/networking/destination-rule-all.yaml"
	bookinfoRuleAllTLSYaml 	= "samples/bookinfo/networking/destination-rule-all-mtls.yaml"
	bookinfoAllv1Yaml       = "samples/bookinfo/networking/virtual-service-all-v1.yaml"
	bookinfoReviewV2Yaml    = "samples/bookinfo/networking/virtual-service-reviews-test-v2.yaml"
	bookinfoReview50V3Yaml  = "samples/bookinfo/networking/virtual-service-reviews-50-v3.yaml"
	bookinfoReviewV3Yaml    = "samples/bookinfo/networking/virtual-service-reviews-v3.yaml"

	bookinfoRatingDelayYaml = "samples/bookinfo/networking/virtual-service-ratings-test-delay.yaml"
	bookinfoRatingAbortYaml = "samples/bookinfo/networking/virtual-service-ratings-test-abort.yaml"

	bookinfoDBYaml          = "samples/bookinfo/platform/kube/bookinfo-db.yaml"

	httpbinYaml             = "samples/httpbin/httpbin-1.1.yaml"
	httpbinFortioYaml       = "samples/httpbin/sample-client/fortio-deploy.yaml"

	httpbinSampleServerCertKey 	= "samples/certs/httpbin.example.com/httpbin.example.com.key"
	httpbinSampleServerCert 	= "samples/certs/httpbin.example.com/httpbin.example.com.crt"
	httpbinSampleCACert        	= "samples/certs/httpbin.example.com/example.com.crt"

	echoYaml				= "samples/tcp-echo/tcp-echo-services.yaml"

	sleepYaml				= "samples/sleep/sleep.yaml"

	nginxYaml          		= "samples/https/nginx-app.yaml"
	nginxNoSidecarYaml 		= "samples/https/nginx-app-without-sidecar.yaml"
	

	kubeconfig 				= ""
	testNamespace 			= "bookinfo"
	testUsername   			= "jason"
	waitTime				= 5
	// KIND
	//gatewayHTTP 			= "localhost:8001/api/v1/namespaces/istio-system/services/istio-ingressgateway:80/proxy"
	
	meshNamespace 			= "service-mesh-1"

)

var (
	// OCP4.x
	gatewayHTTP, _ = util.ShellSilent("kubectl get routes -n %s istio-ingressgateway -o jsonpath='{.spec.host}'", meshNamespace)
	secureIngressPort, _ = util.GetSecureIngressPort(meshNamespace, "istio-ingressgateway", kubeconfig)

)
