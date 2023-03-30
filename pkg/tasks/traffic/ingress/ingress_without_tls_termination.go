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
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

func cleanupIngressWithoutTLS() {
	log.Log.Info("Cleanup")
	util.KubeDeleteContents("bookinfo", nginxIngressGateway)
	nginx := examples.Nginx{"bookinfo"}
	nginx.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestIngressWithoutTLS(t *testing.T) {
	defer cleanupIngressWithoutTLS()
	defer util.RecoverPanic(t)

	log.Log.Info("TestIngressWithOutTLS Termination")
	nginx := examples.Nginx{"bookinfo"}
	nginx.Install("../testdata/examples/x86/nginx/nginx.conf")

	log.Log.Info("Verify NGINX server")
	pod, err := util.GetPodName("bookinfo", "run=my-nginx")
	cmd := fmt.Sprintf(`curl -sS -v -k --resolve nginx.example.com:8443:127.0.0.1 https://nginx.example.com:8443`)
	msg, err := util.PodExec("bookinfo", pod, "istio-proxy", cmd, true)
	util.Inspect(err, "failed to get response", "", t)
	if !strings.Contains(msg, "Welcome to nginx") {
		t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		log.Log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
	} else {
		log.Log.Infof("Success. Get expected response: %s", msg)
	}

	t.Run("TrafficManagement_ingress_configure_ingress_gateway_without_TLS_Termination", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Configure an ingress gateway")
		if err := util.KubeApplyContents("bookinfo", nginxIngressGateway); err != nil {
			t.Errorf("Failed to configure NGINX ingress gateway")
			log.Log.Errorf("Failed to configure NGINX ingress gateway")
		}
		time.Sleep(time.Duration(30) * time.Second)

		url := "https://nginx.example.com:" + secureIngressPort
		resp, err := util.CurlWithCA(url, gatewayHTTP, secureIngressPort, "nginx.example.com", nginxServerCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)

		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "Welcome to nginx") {
			log.Log.Info(string(bodyByte))
		} else {
			t.Errorf("Failed to get Welcome to nginx: %v", string(bodyByte))
		}
	})
}

const nginxIngressGateway = `
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: mygateway
spec:
  selector:
    istio: ingressgateway # use istio default ingress gateway
  servers:
  - port:
      number: 443
      name: https
      protocol: HTTPS
    tls:
      mode: PASSTHROUGH
    hosts:
    - nginx.example.com
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: nginx
spec:
  hosts:
  - nginx.example.com
  gateways:
  - mygateway
  tls:
  - match:
    - port: 443
      sniHosts:
      - nginx.example.com
    route:
    - destination:
        host: my-nginx
        port:
          number: 443`
