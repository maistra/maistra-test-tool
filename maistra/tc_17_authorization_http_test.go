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
	"istio.io/istio/tests/util"
)

var (
	bookinfoRBACOn string
	bookinfoNamespacePolicy string
	bookinfoProductpagePolicy string
	bookinfoReviewPolicy string
	bookinfoRatingPolicy string
)

func updateYaml(namespace string) error {
	data, err := ioutil.ReadFile(bookinfoRBACOnTemplate)
	if err != nil {
		return err
	}	
	bookinfoRBACOn = strings.Replace(string(data), "\"default\"", "\"" + namespace + "\"", -1)

	data, err = ioutil.ReadFile(bookinfoNamespacePolicyTemplate)
	if err != nil {
		return err
	}	
	bookinfoNamespacePolicy = strings.Replace(string(data), "default", namespace, -1)

	data, err = ioutil.ReadFile(bookinfoProductpagePolicyTemplate)
	if err != nil {
		return err
	}
	bookinfoProductpagePolicy = strings.Replace(string(data), "default", namespace, -1)

	data, err = ioutil.ReadFile(bookinfoReviewPolicyTemplate)
	if err != nil {
		return err
	}
	bookinfoReviewPolicy = strings.Replace(string(data), "default", namespace, -1)

	data, err = ioutil.ReadFile(bookinfoRatingPolicyTemplate)
	if err != nil {
		return err
	}
	bookinfoRatingPolicy = strings.Replace(string(data), "default", namespace, -1)

	return nil
}


func cleanup17(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDeleteContents(namespace, bookinfoRatingPolicy, kubeconfig)
	util.KubeDeleteContents(namespace, bookinfoReviewPolicy, kubeconfig)
	util.KubeDeleteContents(namespace, bookinfoProductpagePolicy, kubeconfig)
	util.KubeDeleteContents(namespace, bookinfoNamespacePolicy, kubeconfig)
	util.KubeDeleteContents(namespace, bookinfoRBACOn, kubeconfig)
	
	util.KubeDelete(namespace, bookinfoReviewv3Yaml, kubeconfig)	
	util.ShellSilent("kubectl delete serviceaccount -n %s bookinfo-productpage", namespace)
	util.ShellSilent("kubectl delete serviceaccount -n %s bookinfo-reviews", namespace)

	util.KubeDelete(namespace, bookinfoRuleAllYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoGateway, kubeconfig)
	util.KubeDelete(namespace, bookinfoYaml, kubeconfig)
	
	util.ShellSilent("kubectl delete meshpolicy default")
	log.Info("Waiting... Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)	
}


func cleanupRbac() {
	log.Info("Cleanup old RBAC. Following error can be ignored ...")
	util.ShellSilent("kubectl delete authorization requestcontext -n istio-system")
	util.ShellSilent("kubectl delete rbac handler -n istio-system")
	util.ShellSilent("kubectl delete rule rbaccheck -n istio-system")

	util.ShellSilent("kubectl delete servicerole --all")
	util.ShellSilent("kubectl delete servicerolebinding --all")
	log.Info("Waiting for rules to be cleaned up. Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)
}

func createServiceAccount(namespace, kubeconfig string) error {
	OcGrantPermission("bookinfo-productpage", namespace, kubeconfig)
	OcGrantPermission("bookinfo-reviews", namespace, kubeconfig)
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
	if err := util.CheckPodRunning(namespace, "app=reviews,version=v3", kubeconfig); err != nil {
		return err
	}
	return nil
}

func Test17(t *testing.T) {
	log.Infof("# TC_17 Authorization for HTTP Services")
	Inspect(updateYaml(testNamespace), "failed to update yaml", "", t)
	cleanupRbac()
	log.Info("Enable mTLS")
	Inspect(util.KubeApplyContents("", meshPolicy, kubeconfigFile), "failed to apply MeshPolicy", "", t)
	log.Info("Waiting... Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)	

	log.Info("Deploy bookinfo")
	Inspect(deployBookinfo(testNamespace, kubeconfigFile, true), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	ingress, err := GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)

	log.Info("Create Service Accounts")
	Inspect(createServiceAccount(testNamespace, kubeconfigFile), "failed to create service account", "", t)
	Inspect(util.KubeApply(testNamespace, bookinfoReviewv3Yaml, kubeconfigFile), "failed to apply rule", "", t)
	log.Info("Waiting... Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)	
	
	t.Run("general_check", func(t *testing.T) {
		for i := 0; i <= testRetryTimes; i++ {
			resp, _, err := GetHTTPResponse(productpageURL, nil)
			Inspect(err, "failed to get HTTP Response", "", t)
			defer CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)
			Inspect(
				CompareHTTPResponse(body, "productpage-normal-user-v3.html"), 
				"Didn't get expected response.", 
				"Success. Response matches with expected.", 
				t)	
		}
	})
	
	t.Run("global_rbac", func(t *testing.T) {
		log.Info("Enabling Istio authorization")
		Inspect(util.KubeApplyContents(testNamespace, bookinfoRBACOn, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)	
		for i := 0; i <= testRetryTimes; i++ {
			resp, _, err := GetHTTPResponse(productpageURL, nil)
			Inspect(err, "failed to get HTTP Response", "", t)
			defer CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)
			if resp.StatusCode == 403 {
				log.Infof("Expected response 403: %s", string(body))
			} else {
				t.Errorf("Expected response 403; Got unexpected response: %d", resp.StatusCode)
				log.Errorf("Expected response 403; Got unexpected response: %d", resp.StatusCode)
			}
		}
	})

	t.Run("namespace_rbac", func(t *testing.T) {
		log.Info("Namespace-level access control")
		Inspect(util.KubeApplyContents(testNamespace, bookinfoNamespacePolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)	

		for i := 0; i <= testRetryTimes; i++ {
			resp, _, err := GetHTTPResponse(productpageURL, nil)
			Inspect(err, "failed to get HTTP Response", "", t)
			defer CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)
			Inspect(
				CompareHTTPResponse(body, "productpage-normal-user-v3.html"), 
				"Didn't get expected response.", 
				"Success. Response matches with expected.", 
				t)	
		}
		Inspect(util.KubeDeleteContents(testNamespace, bookinfoNamespacePolicy, kubeconfigFile), "failed to delete policy", "", t)
		log.Info("Waiting for rules to be cleaned up. Sleep 5 seconds...")
		time.Sleep(time.Duration(5) * time.Second)
	})

	t.Run("service_rbac", func(t *testing.T) {
		log.Info("Service-level access control")
		log.Info("Step 1. allowing access to the productpage")
		Inspect(util.KubeApplyContents(testNamespace, bookinfoProductpagePolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)	
		
		for i := 0; i <= testRetryTimes; i++ {
			resp, _, err := GetHTTPResponse(productpageURL, nil)
			Inspect(err, "failed to get HTTP Response", "", t)
			defer CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)
			Inspect(
				CompareHTTPResponse(body, "productpage-rbac-details-reviews-error.html"), 
				"Didn't get expected response.", 
				"Success. Response matches with expected.", 
				t)	
		}

		log.Info("Step 2. allowing access to the details and reviews")
		Inspect(util.KubeApplyContents(testNamespace, bookinfoReviewPolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)

		for i := 0; i <= testRetryTimes; i++ {
			resp, _, err := GetHTTPResponse(productpageURL, nil)
			Inspect(err, "failed to get HTTP Response", "", t)
			defer CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)
			Inspect(
				CompareHTTPResponse(body, "productpage-rbac-rating-error.html"), 
				"Didn't get expected response.", 
				"Success. Response matches with expected.", 
				t)	
		}

		log.Info("Step 3. allowing access to the ratings")
		Inspect(util.KubeApplyContents(testNamespace, bookinfoRatingPolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)
		for i := 0; i <= testRetryTimes; i++ {
			resp, _, err := GetHTTPResponse(productpageURL, nil)
			Inspect(err, "failed to get HTTP Response", "", t)
			defer CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)
			Inspect(
				CompareHTTPResponse(body, "productpage-normal-user-v3.html"), 
				"Didn't get expected response.", 
				"Success. Response matches with expected.", 
				t)	
		}
	})

	defer cleanup17(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			log.Infof("Test failed: %v", err)
		}
	}()
}
