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

package tests

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupRateLimits(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDelete(meshNamespace, mixerRuleProductpageRateLimit, kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	cleanBookinfo(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestRateLimits(t *testing.T) {
	defer cleanupRateLimits(testNamespace)
	defer recoverPanic(t)

	log.Info("Enabling Rate Limits")
	log.Info("Enabling Policy Enforcement")
	util.ShellMuteOutput("kubectl patch -n %s %s/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"disablePolicyChecks\":false}}}}'", meshNamespace, smcpAPI, smcpName)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)

	deployBookinfo(testNamespace, false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)

	if err := util.KubeApply(testNamespace, bookinfoAllv1Yaml, kubeconfig); err != nil {
		t.Errorf("Failed to route traffic to all v1")
		log.Errorf("Failed to route traffic to all v1")
	}

	t.Run("Policies_rate_limits", func(t *testing.T) {
		defer recoverPanic(t)

		if err := util.KubeApply(meshNamespace, mixerRuleProductpageRateLimit, kubeconfig); err != nil {
			t.Errorf("Failed to apply mixer rule")
			log.Errorf("Failed to apply mixer rule")
		}
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		log.Info("productpage permits 2 requests every 5 seconds. Verify 'Quota is exhausted' message")
		i := 1
		startT := time.Now()
		for ; i < 10; i++ {
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "Failed to get HTTP Response", "", t)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)
			if strings.Contains(string(body), "RESOURCE_EXHAUSTED:Quota is exhausted") {
				duration := int(time.Since(startT) / time.Second)
				log.Infof("Response matches expected %s. Requests sent: %v times in %v seconds", string(body), i, duration)
				break
			}
			util.CloseResponseBody(resp)
			time.Sleep(time.Duration(waitTime) * time.Second)
		}
		if i > 3 {
			t.Errorf("Failed. Requests passed: %v times in 5 seconds", i)
			log.Errorf("Failed. Requests passed: %v times in 5 seconds", i)
		}
	})

	t.Run("Policies_conditional_rate_limits", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Conditional rate limits")
		testUserJar := util.GetCookieJar(testUsername, "", "http://"+gatewayHTTP)

		if err := util.KubeApplyContents(meshNamespace, mixerRuleConditional, kubeconfig); err != nil {
			t.Errorf("Failed to apply mixer rule")
			log.Errorf("Failed to apply mixer rule")
		}
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		log.Info("Login user jason should not see quota message")
		for i := 0; i < 10; i++ {
			resp, _, err := util.GetHTTPResponse(productpageURL, testUserJar)
			util.Inspect(err, "Failed to get HTTP Response", "", t)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)
			if strings.Contains(string(body), "RESOURCE_EXHAUSTED:Quota is exhausted") {
				// TBD
			} else {
				log.Info("Success. Login user got expected all v1 response.")
			}
			util.CloseResponseBody(resp)
			time.Sleep(time.Duration(waitTime*2) * time.Second)
		}

		log.Info("Logout user jason")
		util.GetHTTPResponse(fmt.Sprintf("http://%s/logout", gatewayHTTP), testUserJar)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		log.Info("productpage permits 2 requests every 5 seconds. Verify 'Quota is exhausted' message")
		i := 1
		startT := time.Now()
		for ; i < 10; i++ {
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "Failed to get HTTP Response", "", t)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)
			if strings.Contains(string(body), "RESOURCE_EXHAUSTED:Quota is exhausted") {
				duration := int(time.Since(startT) / time.Second)
				log.Infof("Success. Response matches expected %s. Requests sent: %v times in %v seconds", string(body), i, duration)
				break
			}
			util.CloseResponseBody(resp)
		}
		if i > 3 {
			t.Errorf("Failed. Requests passed: %v times in 5 seconds", i)
			log.Errorf("Failed. Requests passed: %v times in 5 seconds", i)
		}
	})
}
