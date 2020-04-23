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
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/istio/pkg/log"
)

func cleanupDenials(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDelete(namespace, mixerRuleDenyIP, kubeconfig)
	util.KubeDelete(namespace, mixerRuleDenyWhitelist, kubeconfig)
	util.KubeDelete(namespace, mixerRuleDenyLabel, kubeconfig)
	util.KubeDelete(namespace, bookinfoReviewv2v3Yaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	cleanBookinfo(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestDenials(t *testing.T) {
	defer cleanupDenials(testNamespace)
	defer recoverPanic(t)

	log.Info("Denials and White/Black Listing")
	log.Info("Enabling Policy Enforcement")
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"disablePolicyChecks\":false}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)

	deployBookinfo(testNamespace, false)
	util.KubeApply(testNamespace, bookinfoAllv1Yaml, kubeconfig)
	util.KubeApply(testNamespace, bookinfoReviewv2v3Yaml, kubeconfig)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)
	testUserJar := util.GetCookieJar(testUsername, "", "http://"+gatewayHTTP)

	t.Run("Policies_simple_denials", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Simple denials")
		if err := util.KubeApply(testNamespace, mixerRuleDenyLabel, kubeconfig); err != nil {
			t.Errorf("Failed to apply mixer deny rule.")
			log.Errorf("Failed to apply mixer deny rule.")
		}
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		log.Info("Check productpage. Without login review shows ratings service is unavailable. Login user jason and see review black stars.")
		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-normal-user-rating-unavailable.html"),
			"Didn't get expected response.",
			"Success. Response matches with expected.",
			t)
		util.CloseResponseBody(resp)

		resp, _, err = util.GetHTTPResponse(productpageURL, testUserJar)
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		body, err = ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-test-user-v2.html"),
			"Didn't get expected response.",
			"Success. Response matches with expected.",
			t)
		util.CloseResponseBody(resp)
		util.KubeDelete(testNamespace, mixerRuleDenyLabel, kubeconfig)
		time.Sleep(time.Duration(waitTime*4) * time.Second)
	})

	t.Run("Policies_attribute_white_black_lists", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Attribute-based whitelists or blacklists")
		if err := util.KubeApply(testNamespace, mixerRuleDenyWhitelist, kubeconfig); err != nil {
			t.Errorf("Failed to apply mixer deny rule.")
			log.Errorf("Failed to apply mixer deny rule.")
		}
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		log.Info("Check productpage. Without login review shows ratings service is unavailable. Login user jason and see review black stars.")
		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-normal-user-rating-unavailable.html"),
			"Didn't get expected response.",
			"Success. Response matches with expected.",
			t)
		util.CloseResponseBody(resp)

		resp, _, err = util.GetHTTPResponse(productpageURL, testUserJar)
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		body, err = ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-test-user-v2.html"),
			"Didn't get expected response.",
			"Success. Response matches with expected.",
			t)
		util.CloseResponseBody(resp)
		util.KubeDelete(testNamespace, mixerRuleDenyWhitelist, kubeconfig)
		time.Sleep(time.Duration(waitTime*4) * time.Second)
	})

	t.Run("Policies_ip-based_white_black_lists", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("IP-based whitelists or blacklists")

		if err := util.KubeApply(testNamespace, mixerRuleDenyIP, kubeconfig); err != nil {
			t.Errorf("Failed to apply mixer deny rule.")
			log.Errorf("Failed to apply mixer deny rule.")
		}
		log.Info("Waiting for rules to propagate. Sleep 100 seconds...")
		time.Sleep(time.Duration(waitTime*20) * time.Second)

		log.Info("Check productpage. Get expected error: PERMISSION_DENIED:staticversion. *:<your mesh source ip> is not whitelisted")
		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		if !strings.Contains(string(body), "PERMISSION_DENIED") {
			t.Errorf("Wrong response: %s", string(body))
			log.Errorf("Wrong response: %s", string(body))
		} else {
			log.Infof("Success. Response matches with expected: %s", string(body))
		}
		util.CloseResponseBody(resp)
	})
}
