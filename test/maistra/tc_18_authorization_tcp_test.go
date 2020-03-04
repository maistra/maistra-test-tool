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

func cleanup18(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDeleteContents(namespace, bookinfoMongodbPolicy, kubeconfig)
	util.KubeDeleteContents(meshNamespace, bookinfoRBAConDB, kubeconfig)
	log.Info("Waiting... Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)

	util.KubeDelete(namespace, bookinfoRatingv2Yaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoDBYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoRatingDBYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoRatingv2ServiceAccount, kubeconfig)

	util.KubeDelete(namespace, bookinfoRuleAllTLSYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoGateway, kubeconfig)
	util.KubeDelete(namespace, bookinfoYaml, kubeconfig)

	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}


func Test18mtls(t *testing.T) {
	defer cleanup18(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_18 Authorization for TCP Services")
	updateYaml()

	// update mtls to true
	log.Info("Update SMCP mtls to true")
	util.ShellMuteOutput("oc patch -n %s smcp/basic-install --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":true,\"mtls\":{\"enabled\":true}}}}}'", meshNamespace)
	time.Sleep(time.Duration(20) * time.Second)

	util.CreateNamespace(testNamespace, kubeconfigFile)
	//util.OcGrantPermission("default", testNamespace, kubeconfigFile)

	log.Info("Deploy bookinfo")
	util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	ingress, err := util.GetOCPIngressgateway("app=istio-ingressgateway", meshNamespace, kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)

	log.Info("Create Service Accounts")
	//util.KubeDeleteContents(meshNamespace, bookinfoRBAConDB, kubeconfigFile)
	if err := util.KubeApply(testNamespace, bookinfoRatingv2ServiceAccount, kubeconfigFile); err != nil {
		log.Errorf("failed to create service account")
	}
	//util.OcGrantPermission("bookinfo-ratings-v2", namespace, kubeconfig)
	log.Info("Waiting for rules to propagate. Sleep 30 seconds...")
	time.Sleep(time.Duration(30) * time.Second)
	util.CheckPodRunning(testNamespace, "app=ratings,version=v2", kubeconfigFile)

	util.Inspect(util.KubeApply(testNamespace, bookinfoRuleAllTLSYaml, kubeconfigFile), "failed to apply rule", "", t)

	log.Info("Deploy MongoDB")
	util.Inspect(util.KubeApply(testNamespace, bookinfoRatingDBYaml, kubeconfigFile), "failed to apply rule", "", t)

	util.Inspect(deployMongoDB(testNamespace, kubeconfigFile), "failed to deploy mongoDB", "", t)

	log.Info("Waiting... Sleep 40 seconds...")
	time.Sleep(time.Duration(40) * time.Second)

	/*
	t.Run("verify_setup_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		util.Inspect(err, "failed to get HTTP Response", "", t)
		defer util.CloseResponseBody(resp)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-normal-user-mongo.html"),
			"Didn't get expected response.",
			"Success. Response matches with expected.",
			t)
	})
	*/

	t.Run("enable_rbac_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Enable Istio Authorization")
		util.Inspect(util.KubeApplyContents(meshNamespace, bookinfoRBAConDB, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)
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
	})

	t.Run("service_rbac_pass_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Enforcing Service-level access control")
		util.Inspect(util.KubeApplyContents(testNamespace, bookinfoMongodbPolicy, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)
		resp, _, err := util.GetHTTPResponse(productpageURL, nil)
		util.Inspect(err, "failed to get HTTP Response", "", t)
		defer util.CloseResponseBody(resp)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		util.Inspect(
			util.CompareHTTPResponse(body, "productpage-normal-user-mongo.html"),
			"Didn't get expected response.",
			"Success. Response matches with expected.",
			t)
	})

	t.Run("service_rbac_fail_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		util.KubeDelete(testNamespace, bookinfoRatingv2ServiceAccount, kubeconfigFile)
		util.Inspect(util.KubeApply(testNamespace, bookinfoRatingv2Yaml, kubeconfigFile), "failed to apply rule", "", t)
		log.Info("Waiting... Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)
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
	})

}
