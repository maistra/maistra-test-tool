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
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup06(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, echo20v2Yaml, kubeconfig)
	util.KubeDelete(namespace, echoAllv1Yaml, kubeconfig)
	util.KubeDelete(namespace, echoYaml, kubeconfig)
	cleanBookinfo(namespace, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func routeTrafficAllv1(namespace, kubeconfig string) error {
	log.Info("Route all TCP traffic to v1 echo")
	if err := util.KubeApply(namespace, echoAllv1Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func routeTraffic20v2(namespace, kubeconfig string) error {
	log.Info("Route 20% of the traffic to v2 echo")
	if err := util.KubeApply(namespace, echo20v2Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func checkEcho(ingressHost, ingressTCPPort string) (string, error) {
	msg, err := util.ShellSilent("docker run -e INGRESS_HOST=%s -e INGRESS_PORT=%s --rm busybox sh -c \"(date; sleep 1) | nc %s %s\"",
		ingressHost, ingressTCPPort, ingressHost, ingressTCPPort)
	if err != nil {
		return "", err
	}
	return msg, nil
}

func Test06(t *testing.T) {
	defer cleanup06(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_06 TCP Traffic Shifting")
	util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)

	ingress, err := util.GetOCP4Ingressgateway("istio-system", kubeconfigFile)
	util.Inspect(err, "cannot get ingress host ip", "", t)

	ingressTCPPort, err := util.GetTCPIngressPort("istio-system", "istio-ingressgateway", kubeconfigFile)
	util.Inspect(err, "cannot get ingress TCP port", "", t)

	util.Inspect(deployEcho(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
	time.Sleep(time.Duration(20) * time.Second)

	t.Run("100_percent_v1_shift_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("# Shifting all TCP traffic to v1")
		util.Inspect(routeTrafficAllv1(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		time.Sleep(time.Duration(5) * time.Second)

		tolerance := 0.0
		totalShot := 10
		versionCount := 0

		log.Infof("Waiting for checking echo dates. Sleep %d seconds...", totalShot*1)

		for i := 0; i < totalShot; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			msg, err := checkEcho(ingress, ingressTCPPort)
			if err != nil {
				ingress, err = util.GetOCPIngressgateway("istio=ingressgateway","istio-system", kubeconfigFile)
				msg, err = checkEcho(ingress, ingressTCPPort)
			}
			util.Inspect(err, "faild to get date", "", t)
			if strings.Contains(msg, "one") {
				versionCount++
			} else {
				log.Errorf("unexpected echo version: %s", msg)
			}
		}

		if isWithinPercentage(versionCount, totalShot, 1, tolerance) {
			log.Info("Success. TCP Traffic shifting acts as expected for 100 percent.")
		} else {
			t.Errorf(
				"Failed traffic shifting test for 100 percent. "+
					"Expected version hit %d", versionCount)
		}
	})

	t.Run("20_percent_v2_shift_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("# Shifting 20% TCP traffic to v2 tolerance 10% ")
		util.Inspect(routeTraffic20v2(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		time.Sleep(time.Duration(5) * time.Second)
		ingress, err := util.GetOCP4Ingressgateway("istio-system", kubeconfigFile)
		util.Inspect(err, "cannot get ingress host ip", "", t)

		tolerance := 0.15
		totalShot := 60
		c1, c2 := 0, 0

		log.Infof("Waiting for checking echo dates. Sleep %d seconds...", totalShot*1)

		for i := 0; i < totalShot; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			msg, err := checkEcho(ingress, ingressTCPPort)
			if err != nil {
				ingress, err = util.GetOCPIngressgateway("istio=ingressgateway","istio-system", kubeconfigFile)
				msg, err = checkEcho(ingress, ingressTCPPort)
			}
			util.Inspect(err, "failed to get date", "", t)
			if strings.Contains(msg, "one") {
				c1++
			} else if strings.Contains(msg, "two") {
				c2++
			} else {
				log.Errorf("unexpected echo version: %s", msg)
			}
		}

		if isWithinPercentage(c1, totalShot, 0.8, tolerance) && isWithinPercentage(c2, totalShot, 0.2, tolerance) {
			log.Infof("Success. Traffic shifting acts as expected. "+
				"v1 version hit %d, v2 version hit %d", c1, c2)
		} else {
			t.Errorf("Failed traffic shifting test for 20 percent. "+
				"v1 version hit %d, v2 version hit %d", c1, c2)
		}
	})

}
