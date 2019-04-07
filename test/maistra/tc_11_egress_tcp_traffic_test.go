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

func cleanup11(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, bookinfoRatingMySQLYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoRatingMySQLv2Yaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoRatingMySQLServiceEntryYaml, kubeconfig)
	cleanBookinfo(namespace, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func configTCPRatings(namespace, kubeconfig string) error {
	if err := util.KubeApply(namespace, bookinfoRatingMySQLv2Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to be cleaned up. Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)
	if err := util.KubeApply(namespace, bookinfoRatingMySQLYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func configEgressTCP(namespace, kubeconfig string) error {
	if err := util.KubeApply(namespace, bookinfoRatingMySQLServiceEntryYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to be cleaned up. Sleep 30 seconds...")
	time.Sleep(time.Duration(30) * time.Second)
	return nil
}


func Test11(t *testing.T) {
	defer cleanup11(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()
	panic("TBD update the external TCP DB location")

	log.Info("# TC_11 Control Egress TCP Traffic")
	util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	
	ingress, err := util.GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)

	util.Inspect(configTCPRatings(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
	
	resp, _, err := util.GetHTTPResponse(productpageURL, nil)
	util.Inspect(err, "failed to get productpage", "", t)
	util.CloseResponseBody(resp)

	log.Info("# Define a TCP mesh-external service entry")
	util.Inspect(configEgressTCP(testNamespace, kubeconfigFile), "failed to apply service entry", "", t)

	resp, _, err = util.GetHTTPResponse(productpageURL, nil)
	util.Inspect(err, "failed to get productpage", "", t)
	body, err := ioutil.ReadAll(resp.Body)
	util.Inspect(err, "failed to read response body", "", t)
	util.Inspect(
		util.CompareHTTPResponse(body, "productpage-normal-user-rating-one-star.html"),
		"Didn't get expected response",
		"Success. Response matches expected one star Ratings",
		t)
	util.CloseResponseBody(resp)

}