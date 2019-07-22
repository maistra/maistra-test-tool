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
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup25(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, mixerDenyPolicyYaml, kubeconfig)
	util.KubeDelete(namespace, policyDenyWhitelistYaml, kubeconfig)
	util.KubeDelete(namespace, policyDenyIPYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoReviewv2v3Yaml, kubeconfig)
	util.ShellSilent("rm -f /tmp/mesh.yaml")
	cleanBookinfo(namespace, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}


func Test25(t *testing.T) {
	defer cleanup25(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_25 Denials and White / Black Listing")
	util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	util.Inspect(util.KubeApply(testNamespace, bookinfoAllv1Yaml, kubeconfigFile), "failed to apply all v1 virtual service", "", t)
	util.Inspect(util.KubeApply(testNamespace, bookinfoReviewv2v3Yaml, kubeconfigFile), "failed to apply review v2 v3 virtual service", "", t)

	ingress, err := util.GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)
	testUserJar := util.GetCookieJar(testUsername, "", "http://"+ingress)

	fmt.Print(productpageURL, testUserJar)

	log.Info("Enable policy check")
	util.ShellMuteOutput("oc -n istio-system get cm istio -o jsonpath=\"{@.data.mesh}\" | sed -e \"s@disablePolicyChecks: true@disablePolicyChecks: false@\" > /tmp/mesh.yaml")
	util.ShellMuteOutput("oc -n istio-system create cm istio -o yaml --dry-run --from-file=mesh=/tmp/mesh.yaml | oc replace -f -")
	log.Info("Verify disablePolicyChecks should be false")
	util.Shell("oc -n istio-system get cm istio -o jsonpath=\"{@.data.mesh}\" | grep disablePolicyChecks")

	t.Run("simple_denials_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Explicitly deny access to v3 reviews")
		util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
		util.Inspect(util.KubeApply(testNamespace, bookinfoAllv1Yaml, kubeconfigFile), "failed to apply all v1 virtual service", "", t)
		util.Inspect(util.KubeApply(testNamespace, bookinfoReviewv2v3Yaml, kubeconfigFile), "failed to apply review v2 v3 virtual service", "", t)

		util.Inspect(util.KubeApply(testNamespace, mixerDenyPolicyYaml, kubeconfigFile), "failed to apply deny rule", "", t)
		time.Sleep(time.Duration(10) * time.Second)

		_, err := util.ShellSilent("oc get rule -n %s | grep denyreviewsv3", testNamespace)
		for err != nil {
			_, err = util.ShellSilent("oc get rule -n %s | grep denyreviewsv3", testNamespace)
			time.Sleep(time.Duration(5) * time.Second)
		}

		log.Info("Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)
		
		log.Info("Check productpage. Without login review shows ratings service is unavailable. Login user jason and see review black stars.")
		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		defer util.CloseResponseBody(resp)

		util.Inspect(err, "failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		util.Inspect(
					util.CompareHTTPResponse(body, "productpage-normal-user-rating-unavailable.html"),
					"Didn't get expected response.",
					"Success. Response matches with expected.",
					t)
			
		resp, _, err = util.GetHTTPResponse(productpageURL, testUserJar)
		defer util.CloseResponseBody(resp)

		util.Inspect(err, "failed to get HTTP Response", "", t)
		body, err = ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		util.Inspect(
					util.CompareHTTPResponse(body, "productpage-test-user-v2.html"),
					"Didn't get expected response.",
					"Success. Response matches with expected.",
					t)

		cleanup25(testNamespace, kubeconfigFile)
	})

	t.Run("attribute_whitelist_blacklist_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Attribute-based whitelists or blacklists")
		util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
		util.Inspect(util.KubeApply(testNamespace, bookinfoAllv1Yaml, kubeconfigFile), "failed to apply all v1 virtual service", "", t)
		util.Inspect(util.KubeApply(testNamespace, bookinfoReviewv2v3Yaml, kubeconfigFile), "failed to apply review v2 v3 virtual service", "", t)

		util.Inspect(util.KubeApply(testNamespace, policyDenyWhitelistYaml, kubeconfigFile), "failed to apply whitelist handler", "", t)
		log.Info("Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)

		log.Info("Check productpage. Without login review shows ratings service is unavailable. Login user jason and see review black stars.")
		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		defer util.CloseResponseBody(resp)

		util.Inspect(err, "failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		util.Inspect(
					util.CompareHTTPResponse(body, "productpage-normal-user-rating-unavailable.html"),
					"Didn't get expected response.",
					"Success. Response matches with expected.",
					t)
			
		resp, _, err = util.GetHTTPResponse(productpageURL, testUserJar)
		defer util.CloseResponseBody(resp)

		util.Inspect(err, "failed to get HTTP Response", "", t)
		body, err = ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		util.Inspect(
					util.CompareHTTPResponse(body, "productpage-test-user-v2.html"),
					"Didn't get expected response.",
					"Success. Response matches with expected.",
					t)
		
		cleanup25(testNamespace, kubeconfigFile)
	})

	t.Run("ip_whitelist_blacklist_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("IP-based whitelists/blacklists")
		util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
		util.Inspect(util.KubeApply(testNamespace, bookinfoAllv1Yaml, kubeconfigFile), "failed to apply all v1 virtual service", "", t)
		util.Inspect(util.KubeApply(testNamespace, bookinfoReviewv2v3Yaml, kubeconfigFile), "failed to apply review v2 v3 virtual service", "", t)

		util.Inspect(util.KubeApply(testNamespace, policyDenyIPYaml, kubeconfigFile), "failed to apply whitelist IP handler", "", t)
		log.Info("Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)

		log.Info("Check productpage. Get expected error: PERMISSION_DENIED:staticversion.istio-system:<your mesh source ip> is not whitelisted")
		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		defer util.CloseResponseBody(resp)

		util.Inspect(err, "failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		if !strings.Contains(string(body), "PERMISSION_DENIED") {
			t.Errorf("Wrong response: %s", string(body))
			log.Errorf("Wrong response: %s", string(body))
		} else {
			log.Infof("Success. Response matches with expected: %s",string(body))
		}
	})

}