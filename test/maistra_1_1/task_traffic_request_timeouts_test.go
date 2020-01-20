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


func cleanupRequestTimeouts(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	cleanBookinfo(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestRequestTimeouts(t *testing.T) {
	defer cleanupRequestTimeouts(testNamespace)
	defer recoverPanic(t)

	log.Infof("# TC_07 Setting Request Timeouts")
	deployBookinfo(testNamespace, false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)

	if err := util.KubeApply(testNamespace, bookinfoAllv1Yaml, kubeconfig); err != nil {
		t.Errorf("Failed to route traffic to all v1")
		log.Errorf("Failed to route traffic to all v1")
	}
	
	if err := util.KubeApplyContents(testNamespace, ratingsDelay2, kubeconfig); err != nil {
		t.Errorf("Failed to inject delay")
		log.Errorf("Failed to inject delay")
	}
	time.Sleep(time.Duration(waitTime) * time.Second)

	t.Run("Request_timeouts", func(t *testing.T) {
		defer recoverPanic(t)

		if err := util.KubeApplyContents(testNamespace, reviewTimeout, kubeconfig); err != nil {
			t.Errorf("Failed to set timeouts")
			log.Errorf("Failed to set timeouts")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)

		resp, duration, err := util.GetHTTPResponse(productpageURL, nil)
		defer util.CloseResponseBody(resp)
		log.Infof("bookinfo productpage returned in %d ms", duration)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-review-timeout.html"),
			"Didn't get expected response.",
			"Success. Request timeouts.",
			t)
	})

}
