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

package traffic

import (
	"github.com/maistra/maistra-test-tool/pkg/util"
)

const (
	bookinfoAllv1Yaml       = "../testdata/examples/x86/bookinfo/virtual-service-all-v1.yaml"
	bookinfoReviewV2Yaml    = "../testdata/examples/x86/bookinfo/virtual-service-reviews-test-v2.yaml"
	bookinfoRatingDelayYaml = "../testdata/examples/x86/bookinfo/virtual-service-ratings-test-delay.yaml"
	bookinfoRatingAbortYaml = "../testdata/examples/x86/bookinfo/virtual-service-ratings-test-abort.yaml"
	bookinfoReview50V3Yaml  = "../testdata/examples/x86/bookinfo/virtual-service-reviews-50-v3.yaml"
	bookinfoReviewV3Yaml    = "../testdata/examples/x86/bookinfo/virtual-service-reviews-v3.yaml"

	// OSSM need custom changes in VirtualService tcp-echo
	echoAllv1Yaml = "../testdata/examples/x86/tcp-echo/tcp-echo-all-v1.yaml"
	echo20v2Yaml  = "../testdata/examples/x86/tcp-echo/tcp-echo-20-v2.yaml"

	meshNamespace = "istio-system"
	smcpName      = "basic"
	testUsername  = "jason"
)

var (
	// OCP4.x
	gatewayHTTP, _       = util.ShellSilent(`kubectl get routes -n %s istio-ingressgateway -o jsonpath='{.spec.host}'`, meshNamespace)
	ingressHTTPPort, _   = util.ShellSilent(`kubectl -n %s get service %s -o jsonpath='{.spec.ports[?(@.name=="http2")].port}'`, meshNamespace, "istio-ingressgateway")
	secureIngressPort, _ = util.ShellSilent(`kubectl -n %s get service %s -o jsonpath='{.spec.ports[?(@.name=="https")].port}'`, meshNamespace, "istio-ingressgateway")
)
