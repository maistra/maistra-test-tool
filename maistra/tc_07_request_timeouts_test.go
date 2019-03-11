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
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)

func cleanup07(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	cleanBookinfo(namespace, kubeconfig)
}

func setup07(namespace, kubeconfig string) error {
	if err := util.KubeApply(namespace, bookinfoAllv1Yaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply(namespace, bookinfoRatingDelayv2Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func setTimeout(namespace, kubeconfig string) error {
	log.Infof("# Set request timeouts")
	if err := util.KubeApply(namespace, bookinfoReviewTimeoutYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func Test07(t *testing.T) {
	defer cleanup07(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			log.Infof("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_07 Setting Request Timeouts")
	Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	ingress, err := GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)

	testUserJar	:= GetCookieJar(testUsername, "", "http://" + ingress)

	Inspect(setup07(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
	t.Run("timout", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()
		
		Inspect(setTimeout(testNamespace, kubeconfigFile), "failed to apply rules", "", t)

		resp, duration, err := GetHTTPResponse(productpageURL, testUserJar)
		defer CloseResponseBody(resp)
		Inspect(err, "failed to get HTTP Response", "", t)
		log.Infof("bookinfo productpage returned in %d ms", duration)
		body, err := ioutil.ReadAll(resp.Body)
		Inspect(err, "failed to read response body", "", t)
		Inspect(
			CompareHTTPResponse(body, "productpage-test-user-v2-review-timeout.html"),
			"Didn't get expected response.",
			"Success. Response timeout matches with expected.",
			t)
	})
	
}

