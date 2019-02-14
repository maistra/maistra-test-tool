// Copyright 2019 Red Hat, Inc.
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

package maistra

import (
	"io/ioutil"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)


func cleanup03(namespace string, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoReviewTestv2Yaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	cleanBookinfo(namespace, kubeconfig)
}

func routeTraffic(namespace string, kubeconfig string) error {
	log.Infof("# Routing traffic to all v1")
	if err := util.KubeApply(namespace, bookinfoAllv1Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func routeTrafficUser(namespace string, kubeconfig string) error {
	log.Infof("# Traffic routing based on user identity")
	if err := util.KubeApply(namespace, bookinfoReviewTestv2Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func Test03(t *testing.T) {
	log.Infof("# TC_03 Traffic Routing")
	Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	
	testUserJar	:= GetCookieJar(testUsername, "", "http://" + ingressURL)

	t.Run("general_route", func(t *testing.T) {
		Inspect(routeTraffic(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		for i := 0; i <= testRetryTimes; i++ {
			resp, duration, err := GetHTTPResponse(productpageURL, nil)
			Inspect(err, "failed to get HTTP Response", "", t)
			log.Infof("bookinfo productpage returned in %d ms", duration)
			defer CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)
			Inspect(
				CompareHTTPResponse(body, "productpage-normal-user-v1.html"), 
				"Didn't get expected response.", 
				"Success. Response matches with expected.", 
				t)	
		}
	})
	t.Run("user_route", func(t *testing.T) {
		Inspect(routeTrafficUser(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		for i := 0; i <= testRetryTimes; i++ {
			resp, duration, err := GetHTTPResponse(productpageURL, testUserJar)
			Inspect(err, "failed to get HTTP Response", "", t)
			log.Infof("bookinfo productpage returned in %d ms", duration)
			defer CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)
			Inspect(
				CompareHTTPResponse(body, "productpage-test-user-v2.html"), 
				"Didn't get expected response.", 
				"Success. Respones matches with expected.",
				t)
		}
	})
	defer cleanup03(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			log.Infof("Test failed: %v", err)
		}
	}()
}
