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
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupIngressGateways() {
	util.Log.Info("Cleanup")
	httpbin := examples.Httpbin{"bookinfo"}
	util.KubeDeleteContents("bookinfo", httpbinGateway1)
	httpbin.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestIngressGateways(t *testing.T) {
	defer cleanupIngressGateways()
	defer util.RecoverPanic(t)

	util.Log.Info("TestIngressGateways")
	httpbin := examples.Httpbin{"bookinfo"}
	httpbin.Install()

	t.Run("TrafficManagement_ingress_status_200_test", func(t *testing.T) {
		defer util.RecoverPanic(t)

		if err := util.KubeApplyContents("bookinfo", httpbinGateway1); err != nil {
			t.Errorf("Failed to configure Gateway")
			util.Log.Errorf("Failed to configure Gateway")
		}
		time.Sleep(time.Duration(20) * time.Second)

		resp, err := util.GetWithHost(fmt.Sprintf("http://%s/status/200", gatewayHTTP), "httpbin.example.com")
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)
		util.Inspect(util.CheckHTTPResponse200(resp), "Failed to get HTTP 200", resp.Status, t)
	})

	t.Run("TrafficManagement_ingress_headers_test", func(t *testing.T) {
		defer util.RecoverPanic(t)

		if err := util.KubeApplyContents("bookinfo", httpbinGateway2); err != nil {
			t.Errorf("Failed to configure Gateway")
			util.Log.Errorf("Failed to configure Gateway")
		}
		time.Sleep(time.Duration(10) * time.Second)

		resp, duration, err := util.GetHTTPResponse(fmt.Sprintf("http://%s/headers", gatewayHTTP), nil)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		util.Log.Infof("httpbin headers page returned in %d ms", duration)
		util.Inspect(util.CheckHTTPResponse200(resp), "Failed to get HTTP 200", resp.Status, t)
	})
}
