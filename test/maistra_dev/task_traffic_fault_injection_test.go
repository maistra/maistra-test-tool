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

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)


func cleanupFaultInjection(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	cleanBookinfo(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestFaultInjection(t *testing.T) {
	defer cleanupFaultInjection(testNamespace)
	defer recoverPanic(t)

	log.Infof("# Fault injection")
	deployBookinfo(testNamespace, false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)
	testUserJar := util.GetCookieJar(testUsername, "", "http://"+ gatewayHTTP)

	if err := util.KubeApply(testNamespace, bookinfoAllv1Yaml, kubeconfig); err != nil {
		t.Errorf("Failed to route traffic to all v1")
		log.Errorf("Failed to route traffic to all v1")
	}
	if err := util.KubeApply(testNamespace, bookinfoReviewV2Yaml, kubeconfig); err != nil {
		t.Errorf("Failed to route traffic based on user")
		log.Errorf("Failed to route traffic based on user")
	}
	time.Sleep(time.Duration(waitTime) * time.Second)

	t.Run("TrafficManagement_injecting_an_HTTP_delay_fault", func(t *testing.T) {
		defer recoverPanic(t)

		if err := util.KubeApply(testNamespace, bookinfoRatingDelayYaml, kubeconfig); err != nil {
			t.Errorf("Failed to inject http delay fault")
			log.Errorf("Failed to inject http delay fault")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)

		minDuration := 5000
		maxDuration := 8000
		standby := 10

		for i := 0; i < 5; i++ {
			resp, duration, err := util.GetHTTPResponse(productpageURL, testUserJar)
			defer util.CloseResponseBody(resp)
			log.Infof("bookinfo productpage returned in %d ms", duration)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-test-user-v2-review-timeout.html"),
				"Didn't get expected response.",
				"Success. HTTP_delay_fault.",
				t)

			if err == nil && duration >= minDuration && duration <= maxDuration {
				log.Info("Success. Fault delay as expected")
				break
			} else if i >= 4 {
				t.Errorf("Fault delay failed. Delay in %d ms while expected between %d ms and %d ms, %s",
					duration, minDuration, maxDuration, err)
				log.Errorf("Fault delay failed. Delay in %d ms while expected between %d ms and %d ms, %s",
					duration, minDuration, maxDuration, err)
			}
			time.Sleep(time.Duration(standby) * time.Second)
		}
	})

	t.Run("TrafficManagement_injecting_an_HTTP_abort_fault", func(t *testing.T) {
		defer recoverPanic(t)

		if err := util.KubeApply(testNamespace, bookinfoRatingAbortYaml, kubeconfig); err != nil {
			t.Errorf("Failed to inject http abort fault")
			log.Errorf("Failed to inject http abort fault")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)

		resp, duration, err := util.GetHTTPResponse(productpageURL, testUserJar)
			defer util.CloseResponseBody(resp)
			log.Infof("bookinfo productpage returned in %d ms", duration)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-test-user-v2-rating-unavailable.html"),
				"Didn't get expected response.",
				"Success. HTTP_abort_fault.",
				t)
	})
}
