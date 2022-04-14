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

func cleanupFaultInjection() {
	util.Log.Info("Cleanup")
	app := examples.Bookinfo{"bookinfo"}
	util.KubeDelete("bookinfo", bookinfoAllv1Yaml)
	app.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestFaultInjection(t *testing.T) {
	defer cleanupFaultInjection()
	defer util.RecoverPanic(t)

	util.Log.Info("TestFaultInjection")
	app := examples.Bookinfo{"bookinfo"}
	app.Install(false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)
	testUserJar := util.GetCookieJar(testUsername, "", "http://"+gatewayHTTP)

	if err := util.KubeApply("bookinfo", bookinfoAllv1Yaml); err != nil {
		t.Errorf("Failed to route traffic to all v1: %s", err)
		util.Log.Errorf("Failed to route traffic to all v1: %s", err)
	}
	if err := util.KubeApply("bookinfo", bookinfoReviewV2Yaml); err != nil {
		t.Errorf("Failed to route traffic based on user: %s", err)
		util.Log.Errorf("Failed to route traffic based on user: %s", err)
	}
	time.Sleep(time.Duration(5) * time.Second)

	t.Run("TrafficManagement_injecting_an_HTTP_delay_fault", func(t *testing.T) {
		defer util.RecoverPanic(t)

		if err := util.KubeApply("bookinfo", bookinfoRatingDelayYaml); err != nil {
			t.Errorf("Failed to inject http delay fault: %s", err)
			util.Log.Errorf("Failed to inject http delay fault: %s", err)
		}
		time.Sleep(time.Duration(5) * time.Second)

		minDuration := 4000
		maxDuration := 14000
		standby := 10

		for i := 0; i < 5; i++ {
			resp, duration, err := util.GetHTTPResponse(productpageURL, testUserJar)
			defer util.CloseResponseBody(resp)
			util.Log.Infof("bookinfo productpage returned in %d ms", duration)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-test-user-v2-review-timeout.html"),
				"Didn't get expected response.",
				"Success. HTTP_delay_fault.",
				t)

			if err == nil && duration >= minDuration && duration <= maxDuration {
				util.Log.Info("Success. Fault delay as expected")
				break
			} else if i >= 4 {
				t.Errorf("Fault delay failed. Delay in %d ms while expected between %d ms and %d ms, %s",
					duration, minDuration, maxDuration, err)
				util.Log.Errorf("Fault delay failed. Delay in %d ms while expected between %d ms and %d ms, %s",
					duration, minDuration, maxDuration, err)
			}
			time.Sleep(time.Duration(standby) * time.Second)
		}
	})

	t.Run("TrafficManagement_injecting_an_HTTP_abort_fault", func(t *testing.T) {
		defer util.RecoverPanic(t)

		if err := util.KubeApply("bookinfo", bookinfoRatingAbortYaml); err != nil {
			t.Errorf("Failed to inject http abort fault: %s", err)
			util.Log.Errorf("Failed to inject http abort fault: %s", err)
		}
		time.Sleep(time.Duration(5) * time.Second)

		resp, duration, err := util.GetHTTPResponse(productpageURL, testUserJar)
		defer util.CloseResponseBody(resp)
		util.Log.Infof("bookinfo productpage returned in %d ms", duration)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-test-user-v2-rating-unavailable.html"),
			"Didn't get expected response.",
			"Success. HTTP_abort_fault.",
			t)
	})
}
