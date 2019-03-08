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
	"fmt"
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)

func cleanup18(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDeleteContents(namespace, bookinfoMongodbPolicy, kubeconfig)
	util.KubeDeleteContents(namespace, bookinfoRBAConDB, kubeconfig)
	log.Info("Waiting... Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)	

	util.KubeDelete(namespace, bookinfoRatingv2Yaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoDBYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoRatingDBYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoRatingv2ServiceAccount, kubeconfig)

	util.KubeDelete(namespace, bookinfoRuleAllTLSYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoGateway, kubeconfig)
	util.KubeDelete(namespace, bookinfoYaml, kubeconfig)

	util.ShellMuteOutput("kubectl delete policy -n %s default", namespace)
	util.ShellMuteOutput("kubectl delete destinationrule -n %s default", namespace)
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)	
}

func setup18(namespace, kubeconfig string) error {
	OcGrantPermission("bookinfo-ratings-v2", namespace, kubeconfig)
	if err := util.KubeApply(namespace, bookinfoRatingv2ServiceAccount, kubeconfig); err != nil {
		return err
	}
	
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(namespace, "app=ratings,version=v2", kubeconfig); err != nil {
		return err
	}
	return nil
}

func Test18(t *testing.T) {
	log.Infof("# TC_18 Authorization for TCP Services")
	Inspect(updateYaml(testNamespace), "failed to update yaml", "", t)
	log.Info("Clean existing mesh policy")
	util.ShellSilent("kubectl delete meshpolicy default")
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)	

	log.Info("Enable mutual TLS")
	Inspect(util.KubeApplyContents(testNamespace, mtlsPolicy, kubeconfigFile), "failed to apply policy", "", t)
	mtlsRule := strings.Replace(mtlsRuleTemplate, "@token@", testNamespace, -1)
	Inspect(util.KubeApplyContents(testNamespace, mtlsRule, kubeconfigFile), "failed to apply rule", "", t)
	log.Info("Waiting... Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)	

	log.Info("Deploy bookinfo")
	Inspect(deployBookinfo(testNamespace, kubeconfigFile, true), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	ingress, err := GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)

	log.Info("Create Service Accounts")
	Inspect(setup18(testNamespace, kubeconfigFile), "failed to create service account", "", t)
	Inspect(util.KubeApply(testNamespace, bookinfoRuleAllTLSYaml, kubeconfigFile), "failed to apply rule", "", t)
	Inspect(util.KubeApply(testNamespace, bookinfoRatingDBYaml, kubeconfigFile), "failed to apply rule", "", t)
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)	

	t.Run("general_check", func(t *testing.T) {
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

		Inspect(deployMongoDB(testNamespace, kubeconfigFile), "failed to deploy mongoDB", "", t)
		log.Info("Waiting... Sleep 45 seconds...")
		time.Sleep(time.Duration(45) * time.Second)	
		resp, _, err = GetHTTPResponse(productpageURL, nil)
		CloseResponseBody(resp)
		resp, _, err = GetHTTPResponse(productpageURL, nil)
		CloseResponseBody(resp)
		resp, _, err = GetHTTPResponse(productpageURL, nil)
		Inspect(err, "failed to get HTTP Response", "", t)
		defer CloseResponseBody(resp)
		body, err = ioutil.ReadAll(resp.Body)
		Inspect(err, "failed to read response body", "", t)
		Inspect(
			CompareHTTPResponse(body, "productpage-normal-user-v3.html"), 
			"Didn't get expected response.", 
			"Success. Response matches with expected.", 
			t)
	})

	t.Run("enable_rbac", func(t *testing.T) {
		log.Info("Enable Istio Authorization")
		Inspect(util.KubeApplyContents(testNamespace, bookinfoRBAConDB, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 20 seconds...")
		time.Sleep(time.Duration(20) * time.Second)	
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
	})

	t.Run("service_rbac_pass", func(t *testing.T) {
		log.Info("Enforcing Service-level access control")
		Inspect(util.KubeApplyContents(testNamespace, bookinfoMongodbPolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)
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
	})

	t.Run("service_rbac_fail", func(t *testing.T) {
		util.KubeDelete(testNamespace, bookinfoRatingv2ServiceAccount, kubeconfigFile)
		Inspect(util.KubeApply(testNamespace, bookinfoRatingv2Yaml, kubeconfigFile), "failed to apply rule", "", t)
		log.Info("Waiting... Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)
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
	})

	defer cleanup18(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			log.Infof("Test failed: %v", err)
		}
	}()
}