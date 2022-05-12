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
	"sync"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupTrafficShifting() {
	util.Log.Info("Cleanup")
	app := examples.Bookinfo{"bookinfo"}
	util.KubeDelete("bookinfo", bookinfoAllv1Yaml)
	app.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestTrafficShifting(t *testing.T) {
	defer cleanupTrafficShifting()
	defer util.RecoverPanic(t)

	util.Log.Info("TestTrafficShifting")
	app := examples.Bookinfo{"bookinfo"}
	app.Install(false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)

	if err := util.KubeApply("bookinfo", bookinfoAllv1Yaml); err != nil {
		t.Errorf("Failed to route traffic to all v1: %s", err)
		util.Log.Errorf("Failed to route traffic to all v1: %s", err)
	}
	time.Sleep(time.Duration(5) * time.Second)

	t.Run("TrafficManagement_shift_50_percent_v3_traffic", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("# Traffic shifting 50 percent v1 and 50 percent v3, tolerance 10 percent")
		if err := util.KubeApply("bookinfo", bookinfoReview50V3Yaml); err != nil {
			t.Errorf("Failed to route 50%% traffic to v3: %s", err)
			util.Log.Errorf("Failed to route 50%% traffic to v3: %s", err)
		}
		time.Sleep(time.Duration(5) * time.Second)

		tolerance := 0.40
		totalShot := 100
		once := sync.Once{}
		c1, cVersionToMigrate := 0, 0

		for i := 0; i < totalShot; i++ {
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "Failed to get response", "", t)
			if err := util.CheckHTTPResponse200(resp); err != nil {
				util.Log.Errorf("Unexpected response status %d", resp.StatusCode)
				continue
			}

			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)

			var c1CompareError, cVersionToMigrateError error

			if c1CompareError = util.CompareHTTPResponse(body, "productpage-normal-user-v1.html"); c1CompareError == nil {
				c1++
			} else if cVersionToMigrateError = util.CompareHTTPResponse(body, "productpage-normal-user-v3.html"); cVersionToMigrateError == nil {
				cVersionToMigrate++
			} else {
				util.Log.Errorf("Received unexpected version")
				once.Do(func() {
					util.Log.Infof("Comparing to the original version: %v", c1CompareError)
					util.Log.Infof("Comparing to the version to migrate to: %v", cVersionToMigrateError)
				})
			}
			util.CloseResponseBody(resp)
		}

		if util.IsWithinPercentage(c1, totalShot, 0.5, tolerance) && util.IsWithinPercentage(cVersionToMigrate, totalShot, 0.5, tolerance) {
			util.Log.Infof(
				"Success. Traffic shifting acts as expected for 50 percent. "+
					"old version hit %d of %d, new version hit %d of %d", c1, totalShot, cVersionToMigrate, totalShot)
		} else {
			t.Errorf(
				"Failed traffic shifting test for 50 percent. "+
					"old version hit %d of %d, new version hit %d of %d", c1, totalShot, cVersionToMigrate, totalShot)
			util.Log.Errorf(
				"Failed traffic shifting test for 50 percent. "+
					"old version hit %d of %d, new version hit %d of %d", c1, totalShot, cVersionToMigrate, totalShot)
		}
	})

	t.Run("TrafficManagement_shift_100_percent_v3_traffic", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("# Traffic shifting 100 percent v3, tolerance 0 percent")
		if err := util.KubeApply("bookinfo", bookinfoReviewV3Yaml); err != nil {
			t.Errorf("Failed to route traffic to v3: %s", err)
			util.Log.Errorf("Failed to route traffic to v3: %s", err)
		}
		time.Sleep(time.Duration(5) * time.Second)

		tolerance := 0.0
		totalShot := 10
		once := sync.Once{}
		cVersionToMigrate := 0

		for i := 0; i < totalShot; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "Failed to get response", "", t)
			if err := util.CheckHTTPResponse200(resp); err != nil {
				util.Log.Errorf("Unexpected response status %d", resp.StatusCode)
				continue
			}

			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)

			var cVersionToMigrateError error

			if cVersionToMigrateError = util.CompareHTTPResponse(body, "productpage-normal-user-v3.html"); cVersionToMigrateError == nil {
				cVersionToMigrate++
			} else {
				util.Log.Errorf("Received unexpected version")
				once.Do(func() {
					util.Log.Infof("Comparing to the version to migrate to: %v", cVersionToMigrateError)
				})
			}
			util.CloseResponseBody(resp)
		}

		if util.IsWithinPercentage(cVersionToMigrate, totalShot, 1, tolerance) {
			util.Log.Infof(
				"Success. Traffic shifting acts as expected for 100 percent. "+
					"new version hit %d of %d", cVersionToMigrate, totalShot)
		} else {
			t.Errorf(
				"Failed traffic shifting test for 100 percent. "+
					"new version hit %d of %d", cVersionToMigrate, totalShot)
			util.Log.Errorf(
				"Failed traffic shifting test for 100 percent. "+
					"new version hit %d of %d", cVersionToMigrate, totalShot)
		}
	})
}
