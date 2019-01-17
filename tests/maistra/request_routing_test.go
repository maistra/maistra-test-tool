// Copyright 2019 Istio Authors
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

// Package dashboard provides testing of the grafana dashboards used in Istio
// to provide mesh monitoring capabilities.

package maistra

import (
	"fmt"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)


const (
	modelDir					= "testdata/bookinfo/output/"
	bookinfoAllv1Yaml			= "testdata/bookinfo/networking/virtual-service-all-v1.yaml"
	testRule					= "testdata/bookinfo/networking/virtual-service-reviews-test-v2.yaml"
	testUsername				= "jason"
)


func cleanup(namespace string, kubeconfig string) {
	log.Infof("Cleanup. Following error can be ignored.")
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	util.KubeDelete(namespace, testRule, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func routeTraffic(namespace string, kubeconfig string) error {
	log.Infof("Routing traffic to all v1")
	if err := util.KubeApply(namespace, bookinfoAllv1Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func routeTrafficUser(namespace string, kubeconfig string) error {
	log.Infof("Traffic routing based on user identity")
	if err := util.KubeApply(namespace, testRule, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func Test03(t *testing.T) {
	log.Infof("TC_03 Traffic Routing")
	fail := false
	host, _ := util.GetIngress("istio-ingressgateway","ingressgateway", "istio-system", "", "NodePort")
	productpageURL := fmt.Sprintf("http://%s/productpage", host)
	testUserJar, _ := setupCookieJar(testUsername, "", "http://" + host)

	t.Run("A1", func(t *testing.T) {
		routeTraffic("bookinfo", "")
		for i := 0; i <= 4; i++ {
			duration, err := checkRoutingResponse( nil, productpageURL, modelDir + "productpage-normal-user-v1.html")
			log.Infof("bookinfo productpage returned in %d ms", duration)
			if err != nil {
				fail = true
				break
			} 
		}
	})
	t.Run("A2", func(t *testing.T) {
		routeTrafficUser("bookinfo", "")
		for i := 0; i <= 4; i++ {
			duration, err := checkRoutingResponse( testUserJar, productpageURL, modelDir + "productpage-test-user-v2.html")
			log.Infof("bookinfo productpage returned in %d ms", duration)
			if err != nil {
				fail = true
				break
			} 
		}
	})
	
	if !fail {
		log.Infof("TC_03 passed")
	} else {
		log.Infof("TC_03 failed")
	}
	
	defer cleanup("bookinfo", "")
}
