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

	util.ShellMuteOutput("kubectl delete policy default -n bookinfo")
	util.ShellMuteOutput("kubectl delete destinationrule -n %s default", namespace)
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}

func setup18(namespace, kubeconfig string) error {
	util.OcGrantPermission("bookinfo-ratings-v2", namespace, kubeconfig)
	if err := util.KubeApply(namespace, bookinfoRatingv2ServiceAccount, kubeconfig); err != nil {
		return err
	}

	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	err := util.CheckPodRunning(namespace, "app=ratings,version=v2", kubeconfig)
	return err
}

func Test18(t *testing.T) {
	defer cleanup18(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_18 Authorization for TCP Services")
	updateYaml(testNamespace)
	log.Info("Clean existing mesh policy")
	util.ShellSilent("kubectl delete meshpolicy default")
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)

	util.CreateNamespace(testNamespace, kubeconfigFile)
	util.OcGrantPermission("default", testNamespace, kubeconfigFile)

	log.Info("Enable mutual TLS")
	util.Inspect(util.KubeApplyContents(testNamespace, bookinfoPolicy, kubeconfigFile), "failed to apply MeshPolicy", "", t)
	mtlsRule := strings.Replace(mtlsRuleTemplate, "@token@", testNamespace, -1)
	util.Inspect(util.KubeApplyContents(testNamespace, mtlsRule, kubeconfigFile), "failed to apply rule", "", t)
	log.Info("Waiting... Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)

	log.Info("Deploy bookinfo")
	util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	ingress, err := util.GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)

	log.Info("Create Service Accounts")
	util.Inspect(setup18(testNamespace, kubeconfigFile), "failed to create service account", "", t)
	util.Inspect(util.KubeApply(testNamespace, bookinfoRuleAllTLSYaml, kubeconfigFile), "failed to apply rule", "", t)

	log.Info("Deploy MongoDB")
	util.Inspect(util.KubeApply(testNamespace, bookinfoRatingDBYaml, kubeconfigFile), "failed to apply rule", "", t)

	util.Inspect(deployMongoDB(testNamespace, kubeconfigFile), "failed to deploy mongoDB", "", t)

	log.Info("Redeploy bookinfo ratings v2")
	util.KubeDelete(testNamespace, bookinfoRatingv2ServiceAccount, kubeconfigFile)
	log.Info("Waiting... Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	util.Inspect(setup18(testNamespace, kubeconfigFile), "failed to create service account", "", t)

	log.Info("Waiting... Sleep 40 seconds...")
	time.Sleep(time.Duration(40) * time.Second)

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

	t.Run("enable_rbac_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Enable Istio Authorization")
		util.Inspect(util.KubeApplyContents(testNamespace, bookinfoRBAConDB, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 20 seconds...")
		time.Sleep(time.Duration(20) * time.Second)
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
		log.Info("Waiting... Sleep 20 seconds...")
		time.Sleep(time.Duration(20) * time.Second)
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
