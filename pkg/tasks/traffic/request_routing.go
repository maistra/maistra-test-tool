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
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupRequestRouting() {
	util.Log.Info("Cleanup")
	app := examples.Bookinfo{"bookinfo"}
	util.KubeDelete("bookinfo", bookinfoAllv1Yaml)
	app.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestRequestRouting(t *testing.T) {
	defer cleanupRequestRouting()
	defer util.RecoverPanic(t)

	util.Log.Info("TestRequestRouting")
	app := examples.Bookinfo{"bookinfo"}
	app.Install(false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)
	testUserJar := util.GetCookieJar(testUsername, "", "http://"+gatewayHTTP)

	t.Run("TrafficManagement_test_the_new_routing_configuration", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Routing traffic to all v1")
		if err := util.KubeApply("bookinfo", bookinfoAllv1Yaml); err != nil {
			t.Errorf("Failed to route traffic to all v1: %s", err)
			util.Log.Errorf("Failed to route traffic to all v1: %s", err)
		}
		time.Sleep(time.Duration(20) * time.Second)

		for i := 0; i <= 5; i++ {
			resp, duration, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "Failed to get HTTP Response", "", t)
			util.Log.Infof("bookinfo productpage returned in %d ms", duration)
			defer util.CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-normal-user-v1.html"),
				"Didn't get expected response.",
				"Success. Routing traffic to all v1.",
				t)
		}
	})

	t.Run("TrafficManagement_route_based_on_user_identity", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Traffic routing based on user identity")
		if err := util.KubeApply("bookinfo", bookinfoReviewV2Yaml); err != nil {
			t.Errorf("Failed to route traffic based on user: %s", err)
			util.Log.Errorf("Failed to route traffic based on user: %s", err)
		}
		time.Sleep(time.Duration(20) * time.Second)

		for i := 0; i <= 5; i++ {
			resp, duration, err := util.GetHTTPResponse(productpageURL, testUserJar)
			util.Inspect(err, "Failed to get HTTP Response", "", t)
			util.Log.Infof("bookinfo productpage returned in %d ms", duration)
			defer util.CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-test-user-v2.html"),
				"Didn't get expected response.",
				"Success. Route_based_on_user_identity.",
				t)
		}
	})
}
