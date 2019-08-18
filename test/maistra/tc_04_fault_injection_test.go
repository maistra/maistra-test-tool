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
	"maistra/util"
)

func cleanup04(namespace string, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	cleanBookinfo(namespace, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func setup04(namespace, kubeconfig string) error {
	if err := util.KubeApply(namespace, bookinfoAllv1Yaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply(namespace, bookinfoReviewTestv2Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func faultInject(namespace, kubeconfig string) error {
	log.Infof("# Inject HTTP delay fault")
	if err := util.KubeApply(namespace, bookinfoRatingDelayYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func faultFix(namespace, kubeconfig string) error {
	log.Infof("# Fixing HTTP delay fault")
	if err := util.KubeApply(namespace, bookinfoRatingDelayv2Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func abortInject(namespace, kubeconfig string) error {
	log.Infof("# Inject HTTP abort fault")
	if err := util.KubeApply(namespace, bookinfoRatingAbortYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func Test04(t *testing.T) {
	defer cleanup04(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_04 Fault injection")
	util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	ingress, err := util.GetOCPIngressgateway("app=istio-ingressgateway", meshNamespace, kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)

	util.Inspect(setup04(testNamespace, kubeconfigFile), "failed to apply rules", "", t)

	testUserJar := util.GetCookieJar(testUsername, "", "http://"+ingress)

	t.Run("delay_fault_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		util.Inspect(faultInject(testNamespace, kubeconfigFile), "failed to apply rules", "", t)

		minDuration := 5000
		maxDuration := 8000
		standby := 10

		for i := 0; i < testRetryTimes; i++ {
			resp, duration, err := util.GetHTTPResponse(productpageURL, testUserJar)
			defer util.CloseResponseBody(resp)
			log.Infof("bookinfo productpage returned in %d ms", duration)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-test-user-v2-review-timeout.html"),
				"Didn't get expected response.",
				"Success. Response matches with expected.",
				t)

			if err == nil && duration >= minDuration && duration <= maxDuration {
				log.Info("Success. Fault delay as expected")
				break
			}
			if i == testRetryTimes-1 {
				t.Errorf("Fault delay failed. Delay in %d ms while expected between %d ms and %d ms, %s",
					duration, minDuration, maxDuration, err)
				break
			}
			time.Sleep(time.Duration(standby) * time.Second)
		}
	})

	t.Run("fix_fault_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		util.Inspect(faultFix(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		resp, duration, err := util.GetHTTPResponse(productpageURL, testUserJar)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "failed to get HTTP Response", "", t)
		log.Infof("bookinfo productpage returned in %d ms", duration)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-test-user-v2.html"),
			"Didn't get expected response.",
			"Success. Response matches with expected.",
			t)
	})

	t.Run("abort_fault_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		util.Inspect(abortInject(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		resp, duration, err := util.GetHTTPResponse(productpageURL, testUserJar)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "failed to get HTTP Response", "", t)
		log.Infof("bookinfo productpage returned in %d ms", duration)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-test-user-v2-rating-unavailable.html"),
			"Didn't get expected response.",
			"Success. Response abort matches with expected.",
			t)
	})

}
