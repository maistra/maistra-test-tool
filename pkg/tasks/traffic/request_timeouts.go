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
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

func cleanupRequestTimeouts() {
	log.Log.Info("Cleanup")
	app := examples.Bookinfo{"bookinfo"}
	util.KubeDelete("bookinfo", bookinfoAllv1Yaml)
	app.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestRequestTimeouts(t *testing.T) {
	defer cleanupRequestTimeouts()
	defer util.RecoverPanic(t)

	log.Log.Infof("TestRequestTimeouts")
	app := examples.Bookinfo{"bookinfo"}
	app.Install(false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)

	if err := util.KubeApply("bookinfo", bookinfoAllv1Yaml); err != nil {
		t.Errorf("Failed to route traffic to all v1")
		log.Log.Errorf("Failed to route traffic to all v1")
	}

	t.Run("TrafficManagement_request_delay", func(t *testing.T) {
		defer util.RecoverPanic(t)
		if err := util.KubeApplyContents("bookinfo", ratingsDelay2); err != nil {
			t.Errorf("Failed to inject delay")
			log.Log.Errorf("Failed to inject delay")
		}
		time.Sleep(time.Duration(5) * time.Second)
	})

	t.Run("TrafficManagement_request_timeouts", func(t *testing.T) {
		defer util.RecoverPanic(t)
		if err := util.KubeApplyContents("bookinfo", reviewTimeout); err != nil {
			t.Errorf("Failed to set timeouts")
			log.Log.Errorf("Failed to set timeouts")
		}
		time.Sleep(time.Duration(5) * time.Second)

		resp, duration, err := util.GetHTTPResponse(productpageURL, nil)
		defer util.CloseResponseBody(resp)
		log.Log.Infof("bookinfo productpage returned in %d ms", duration)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-review-timeout.html"),
			"Didn't get expected response.",
			"Success. Request timeouts.",
			t)
	})
}
