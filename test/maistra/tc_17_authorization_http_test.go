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

func cleanup17(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDeleteContents(namespace, bookinfoRatingPolicy, kubeconfig)
	util.KubeDeleteContents(namespace, bookinfoReviewPolicy, kubeconfig)
	util.KubeDeleteContents(namespace, bookinfoProductpagePolicy, kubeconfig)
	util.KubeDeleteContents(namespace, bookinfoNamespacePolicy, kubeconfig)
	util.KubeDeleteContents(namespace, bookinfoRBACOn, kubeconfig)

	util.KubeDelete(namespace, bookinfoReviewv3Yaml, kubeconfig)
	util.ShellMuteOutput("kubectl delete serviceaccount -n %s bookinfo-productpage", namespace)
	util.ShellMuteOutput("kubectl delete serviceaccount -n %s bookinfo-reviews", namespace)

	util.KubeDelete(namespace, bookinfoRuleAllTLSYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoGateway, kubeconfig)
	util.KubeDelete(namespace, bookinfoYaml, kubeconfig)

	util.ShellMuteOutput("kubectl delete meshpolicy default")
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}


func setup17(namespace, kubeconfig string) error {
	util.OcGrantPermission("bookinfo-productpage", namespace, kubeconfig)
	util.OcGrantPermission("bookinfo-reviews", namespace, kubeconfig)
	if err := util.KubeApply(namespace, bookinfoAddServiceAccountYaml, kubeconfig); err != nil {
		return err
	}

	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(namespace, "app=productpage", kubeconfig); err != nil {
		return err
	}
	if err := util.CheckPodRunning(namespace, "app=reviews,version=v2", kubeconfig); err != nil {
		return err
	}
	err := util.CheckPodRunning(namespace, "app=reviews,version=v3", kubeconfig)
	log.Info("Waiting for rules to propagate. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
	return err
}

func Test17(t *testing.T) {
	defer cleanup17(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	Retry := 3

	log.Infof("# TC_17 Authorization for HTTP Services")
	updateYaml(testNamespace)
	
	log.Info("Enable mTLS")
	util.Inspect(util.KubeApplyContents("", meshPolicy, kubeconfigFile), "failed to apply MeshPolicy", "", t)
	log.Info("Waiting... Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)

	log.Info("Deploy bookinfo")
	util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, true), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	ingress, err := util.GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)

	log.Info("Create Service Accounts")
	util.Inspect(setup17(testNamespace, kubeconfigFile), "failed to create service account", "", t)
	util.Inspect(util.KubeApply(testNamespace, bookinfoReviewv3Yaml, kubeconfigFile), "failed to apply rule", "", t)
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)

	t.Run("verify_setup_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		for i := 0; i <= Retry; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "failed to get HTTP Response", "", t)
			defer util.CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-normal-user-v3.html"),
				"Didn't get expected response.",
				"Success. Response matches with expected.",
				t)
		}
	})

	t.Run("global_rbac_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Enabling Istio authorization")
		util.Inspect(util.KubeApplyContents(testNamespace, bookinfoRBACOn, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)
		for i := 0; i <= Retry; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "failed to get HTTP Response", "", t)
			defer util.CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)
			if resp.StatusCode == 403 {
				log.Infof("Expected response 403: %s", string(body))
			} else {
				t.Errorf("Expected response 403; Got unexpected response: %d", resp.StatusCode)
				log.Errorf("Expected response 403; Got unexpected response: %d", resp.StatusCode)
			}
		}
	})

	t.Run("namespace_rbac_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Namespace-level access control")
		util.Inspect(util.KubeApplyContents(testNamespace, bookinfoNamespacePolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 60 seconds...")
		time.Sleep(time.Duration(60) * time.Second)

		for i := 0; i <= Retry; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "failed to get HTTP Response", "", t)
			defer util.CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-normal-user-v3.html"),
				"Didn't get expected response.",
				"Success. Response matches with expected.",
				t)
		}
		util.Inspect(util.KubeDeleteContents(testNamespace, bookinfoNamespacePolicy, kubeconfigFile), "failed to delete policy", "", t)
		log.Info("Waiting for rules to be cleaned up. Sleep 5 seconds...")
		time.Sleep(time.Duration(5) * time.Second)
	})

	t.Run("service_rbac_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Service-level access control")
		log.Info("Step 1. allowing access to the productpage")
		util.Inspect(util.KubeApplyContents(testNamespace, bookinfoProductpagePolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)

		for i := 0; i <= Retry; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "failed to get HTTP Response", "", t)
			defer util.CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-rbac-details-reviews-error.html"),
				"Didn't get expected response.",
				"Success. Response matches with expected.",
				t)
		}

		log.Info("Step 2. allowing access to the details and reviews")
		util.Inspect(util.KubeApplyContents(testNamespace, bookinfoReviewPolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)

		for i := 0; i <= Retry; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "failed to get HTTP Response", "", t)
			defer util.CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-rbac-rating-error.html"),
				"Didn't get expected response.",
				"Success. Response matches with expected.",
				t)
		}

		log.Info("Step 3. allowing access to the ratings")
		util.Inspect(util.KubeApplyContents(testNamespace, bookinfoRatingPolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)
		for i := 0; i <= Retry; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "failed to get HTTP Response", "", t)
			defer util.CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-normal-user-v3.html"),
				"Didn't get expected response.",
				"Success. Response matches with expected.",
				t)
		}
	})

}
