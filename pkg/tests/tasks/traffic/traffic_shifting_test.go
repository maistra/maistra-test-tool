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
	_ "embed"
	"net/http"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	. "github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestTrafficShifting(t *testing.T) {
	NewTest(t).Id("T3").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		ns := "bookinfo"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t, app.Bookinfo(ns))
		productpageURL := app.BookinfoProductPageURL(t, meshNamespace)

		oc.ApplyString(t, ns, bookinfoVirtualServicesAllV1)

		t.NewSubTest("50 percent to v3").Run(func(t TestHelper) {
			t.LogStep("configure VirtualService to split traffic 50% to v1 and 50% to v3")
			oc.ApplyString(t, ns, splitReviews5050BetweenV1andV3)

			t.LogStep("Make 100 requests and check if v1 and v3 get 50% of requests each (tolerance: 20%)")
			retry.UntilSuccess(t, func(t TestHelper) {
				tolerance := 0.20
				checkTrafficRatio(t, productpageURL, 100, tolerance, map[string]float64{
					"productpage-normal-user-v1.html": 0.5,
					"productpage-normal-user-v3.html": 0.5,
				})
			})
		})

		t.NewSubTest("100 percent to v3").Run(func(t TestHelper) {
			t.LogStep("configure VirtualService to send all traffic to v3")
			oc.ApplyString(t, ns, allReviewsToV3)

			t.LogStep("Make 100 requests and check if all of them go to v3 (tolerance: 0%)")
			retry.UntilSuccess(t, func(t TestHelper) {
				tolerance := 0.0
				checkTrafficRatio(t, productpageURL, 100, tolerance, map[string]float64{
					"productpage-normal-user-v1.html": 0.0,
					"productpage-normal-user-v3.html": 1.0,
				})
			})
		})
	})
}

func checkTrafficRatio(t TestHelper, url string, numberOfRequests int, tolerance float64, ratios map[string]float64) {
	counts := map[string]int{}
	for i := 0; i < numberOfRequests; i++ {
		curl.Request(t,
			url, nil,
			assert.ResponseStatus(http.StatusOK),
			func(t TestHelper, response *http.Response, responseBody []byte, duration time.Duration) {
				comparisonErrors := map[string]error{}
				matched := false
				for file := range ratios {
					err := CompareHTTPResponse(responseBody, file)
					if err == nil {
						matched = true
						counts[file]++
					} else {
						comparisonErrors[file] = err
					}
				}
				if !matched {
					// for file, err := range comparisonErrors {
					// 	t.Logf("Diff with %s: %v", file, err)
					// }
					matchedFile := app.FindBookinfoProductPageResponseFile(responseBody)
					if matchedFile == "" {
						t.Fatal("Response did not match any expected value and also didn't match any standard bookinfo productpage responses")
					} else {
						t.Fatalf("Response did not match any expected value, but matched file %q", matchedFile)
					}
				}
			},
		)
	}

	for file, count := range counts {
		expectedRate := ratios[file]
		actualRate := float64(count) / float64(numberOfRequests)
		if IsWithinPercentage(count, numberOfRequests, expectedRate, tolerance) {
			t.Logf("success: %d/%d responses matched %s (actual rate %f, expected %f, tolerance %f)", count, numberOfRequests, file, actualRate, expectedRate, tolerance)
		} else {
			t.Errorf("failure: %d/%d responses matched %s (actual rate %f, expected %f, tolerance %f)", count, numberOfRequests, file, actualRate, expectedRate, tolerance)
		}
	}
}

var (
	//go:embed yaml/virtualservice-reviews-split-v1-v3-50-50.yaml
	splitReviews5050BetweenV1andV3 string

	//go:embed yaml/virtualservice-reviews-reviews-v3.yaml
	allReviewsToV3 string
)
