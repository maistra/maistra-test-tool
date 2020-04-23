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

func cleanupAuthorizationHTTP(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, detailsGETPolicy, kubeconfig)
	util.KubeDeleteContents(namespace, reviewsGETPolicy, kubeconfig)
	util.KubeDeleteContents(namespace, ratingsGETPolicy, kubeconfig)
	util.KubeDeleteContents(namespace, productpageGETPolicy, kubeconfig)
	util.KubeDeleteContents(namespace, denyAllPolicy, kubeconfig)
	cleanBookinfo(namespace)
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":false,\"mtls\":{\"enabled\":false}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)
}

func TestAuthorizationHTTP(t *testing.T) {
	defer cleanupAuthorizationHTTP(testNamespace)
	defer recoverPanic(t)

	log.Info("Authorization for HTTP traffic")

	// update mtls to true
	log.Info("Update SMCP mtls to true")
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":true,\"mtls\":{\"enabled\":true}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)

	deployBookinfo(testNamespace, true)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)

	t.Run("Security_authorization_rbac_deny_all_http", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Configure access control for workloads using HTTP traffic")
		util.KubeApplyContents(testNamespace, denyAllPolicy, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)
		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		util.Inspect(err, "Failed to get HTTP Response", "", t)

		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		if strings.Contains(string(body), "RBAC: access denied") {
			log.Infof("Got access denied as expected: %s", string(body))
		} else {
			t.Errorf("RBAC deny all failed. Got response: %s", string(body))
			log.Errorf("RBAC deny all failed. Got response: %s", string(body))
		}
		util.CloseResponseBody(resp)
	})

	t.Run("Security_authorization_rbac_allow_GET_http", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents(testNamespace, productpageGETPolicy, kubeconfig)
		time.Sleep(time.Duration(waitTime*10) * time.Second)
		util.GetHTTPResponse(productpageURL, nil) // dummy request to refresh previous page

		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		util.Inspect(err, "Failed to get HTTP Response", "", t)

		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		if strings.Contains(string(body), "Error fetching product details") && strings.Contains(string(body), "Error fetching product reviews") {
			log.Infof("Got expected page with Error fetching product details and Error fetching product reviews")
		} else {
			t.Errorf("Productpage GET policy failed. Got response: %s", string(body))
			log.Errorf("Productpage GET policy failed. Got response: %s", string(body))
		}
		util.CloseResponseBody(resp)

		util.KubeApplyContents(testNamespace, detailsGETPolicy, kubeconfig)
		util.KubeApplyContents(testNamespace, reviewsGETPolicy, kubeconfig)
		util.KubeApplyContents(testNamespace, ratingsGETPolicy, kubeconfig)
		time.Sleep(time.Duration(waitTime*10) * time.Second)
		util.GetHTTPResponse(productpageURL, nil) // dummy request to refresh previous page

		resp, _, err = util.GetHTTPResponse(productpageURL, nil)
		util.Inspect(err, "Failed to get HTTP Response", "", t)

		body, err = ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		if strings.Contains(string(body), "Error fetching product details") || strings.Contains(string(body), "Error fetching product reviews") || strings.Contains(string(body), "Ratings service currently unavailable") {
			t.Errorf("GET policy failed. Got response: %s", string(body))
			log.Errorf("GET policy failed. Got response: %s", string(body))
		} else {
			log.Infof("Got expected page.")
		}
		util.CloseResponseBody(resp)
	})
}
