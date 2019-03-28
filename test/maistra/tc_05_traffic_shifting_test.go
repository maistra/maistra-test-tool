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
	"sync"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)

func cleanup05(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	cleanBookinfo(namespace, kubeconfig)
}

func setup05(namespace, kubeconfig string) error {
	if err := util.KubeApply(namespace, bookinfoAllv1Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func trafficShift50v3(namespace, kubeconfig string) error {
	log.Info("# Traffic shifting 50 percent v1 and 50 percent v3, tolerance 10 percent")
	if err := util.KubeApply(namespace, bookinfoReview50v3Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func trafficShiftAllv3(namespace, kubeconfig string) error {
	log.Infof("# Shifting all traffic to v3")
	if err := util.KubeApply(namespace, bookinfoReviewv3Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func isWithinPercentage(count int, total int, rate float64, tolerance float64) bool {
	minimum := int((rate - tolerance) * float64(total))
	maximum := int((rate + tolerance) * float64(total))
	return count >= minimum && count <= maximum
}

func Test05(t *testing.T) {
	defer cleanup05(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()
	
	log.Infof("# TC_05 Traffic Shifting")
	Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	ingress, err := GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)

	Inspect(setup05(testNamespace, kubeconfigFile), "failed to apply rules", "", t)

	t.Run("50%_v3_shift", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		Inspect(trafficShift50v3(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		tolerance := 0.10
		totalShot := 100
		once := sync.Once{}
		c1, cVersionToMigrate := 0, 0
		for i := 0; i < totalShot; i++ {
			resp, _, err := GetHTTPResponse(productpageURL, nil)
			Inspect(err, "failed to get response", "", t)
			if err := CheckHTTPResponse200(resp); err != nil {
				log.Errorf("unexpected response status %d", resp.StatusCode)
				continue
			}
			
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)

			var c1CompareError, cVersionToMigrateError error
			
			if c1CompareError = CompareHTTPResponse(body, "productpage-normal-user-v1.html"); c1CompareError == nil {
				c1++
			} else if cVersionToMigrateError = CompareHTTPResponse(body, "productpage-normal-user-v3.html"); cVersionToMigrateError == nil {
				cVersionToMigrate++
			} else {
				log.Errorf("received unexpected version")
				once.Do(func() {
					log.Infof("comparing to the original version: %v", c1CompareError)
					log.Infof("comparing to the version to migrate to: %v", cVersionToMigrateError)
				})
			}
			CloseResponseBody(resp)
		}
		
		if isWithinPercentage(c1, totalShot, 0.5, tolerance) && isWithinPercentage(cVersionToMigrate, totalShot, 0.5, tolerance) {
			log.Infof(
				"Success. Traffic shifting acts as expected for 50 percent. " +
				"old version hit %d, new version hit %d", c1, cVersionToMigrate)
		} else {
			t.Errorf(
				"Failed traffic shifting test for 50 percent. " +
				"old version hit %d, new version hit %d", c1, cVersionToMigrate)
		}
	})

	t.Run("100%_v3_shift", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()
		
		Inspect(trafficShiftAllv3(testNamespace, kubeconfigFile), "failed to apply rules", "", t)

		tolerance := 0.0
		totalShot := 100
		once := sync.Once{}
		cVersionToMigrate := 0

		for i := 0; i < totalShot; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			resp, _, err := GetHTTPResponse(productpageURL, nil)
			Inspect(err, "failed to get response", "", t)
			if err := CheckHTTPResponse200(resp); err != nil {
				log.Errorf("unexpected response status %d", resp.StatusCode)
				continue
			}
			
			body, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)

			var cVersionToMigrateError error
			
			if cVersionToMigrateError = CompareHTTPResponse(body, "productpage-normal-user-v3.html"); cVersionToMigrateError == nil {
				cVersionToMigrate++
			} else {
				log.Errorf("received unexpected version")
				once.Do(func() {
					log.Infof("comparing to the version to migrate to: %v", cVersionToMigrateError)
				})
			}
			CloseResponseBody(resp)
		}
		
		if isWithinPercentage(cVersionToMigrate, totalShot, 1, tolerance) {
			log.Infof(
				"Success. Traffic shifting acts as expected for 100 percent. " +
				"new version hit %d", cVersionToMigrate)
		} else {
			t.Errorf(
				"Failed traffic shifting test for 100 percent. " +
				"new version hit %d", cVersionToMigrate)
		}	
	})

}