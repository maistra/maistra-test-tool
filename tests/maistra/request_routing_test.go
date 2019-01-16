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
	"net/http"
	"testing"
	"io/ioutil"
	"time"
	
	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)


const (
	modelDir					= "testdata/bookinfo/output/"
	bookinfoAllv1Yaml			= "testdata/bookinfo/networking/virtual-service-all-v1.yaml"
	testRule					= "testdata/bookinfo/networking/virtual-service-reviews-test-v2.yaml"
)


func cleanup(namespace string, kubeconfig string) {
	log.Infof("Cleanup. Following error can be ignored.")
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	util.KubeDelete(namespace, testRule, kubeconfig)
}


func routeTraffic(namespace string, kubeconfig string) error {
	log.Infof("Routing traffic to all v1")
	err := util.KubeApply(namespace, bookinfoAllv1Yaml, kubeconfig)
	return err
}

func routeTrafficUser(namespace string, kubeconfig string) error {
	log.Infof("Traffic routing based on user identity")
	err := util.KubeApply(namespace, testRule, kubeconfig)
	return err
}

func closeResponseBody(r *http.Response) {
	if err := r.Body.Close(); err != nil {
		log.Errora(err)
	}
}

func checkHTTPResponseCode(gateway string) (int, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/productpage", gateway))
	if err != nil {
		return -1, err
	}

	defer closeResponseBody(resp)
	log.Infof("Get from page: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		log.Errorf("Get response from product page failed!")
		return -1, fmt.Errorf("status code is %d", resp.StatusCode)
	}
	return 1, nil
}

func checkResponseBody(gateway string) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/productpage", gateway))
	if err != nil {
		return nil, err
	}

	defer closeResponseBody(resp)
	log.Infof("Get Response Body")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func saveHTTPResponse(gateway, dst string) (int, error) {
	log.Infof("Write response body to file: %s", dst)
	body, err := checkResponseBody(gateway)
	err = ioutil.WriteFile(dst, body, 0644)
	if err != nil {
		return -1, err
	}
	return 1, nil
}

func checkRoutingResponse(url, modelFile string) (int, error) {
	startT := time.Now()
	resp, err := http.Get(fmt.Sprintf(url))
	if err != nil {
		return -1, err
	}
	defer closeResponseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("status code is %d", resp.StatusCode)
	}
	duration := int(time.Since(startT) / (time.Second / time.Microsecond))
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	if err = util.CompareToFile(body, modelFile); err != nil {
		log.Errorf("Error: didn't get expected response")
		ioutil.WriteFile("/tmp/diffbody", body, 0644)
		return -1, err
	}
	return duration, err
}


func Test03(t *testing.T) {
	log.Infof("TC_03 Traffic Routing")
	fail := false
	host, _ := util.GetIngress("istio-ingressgateway","ingressgateway", "istio-system", "", "NodePort")
	productpageURL := fmt.Sprintf("http://%s/productpage", host)

	t.Run("A1", func(t *testing.T) {
		routeTraffic("bookinfo", "")
		for i := 0; i <= 4; i++ {
			duration, err := checkRoutingResponse(productpageURL, modelDir + "productpage-normal-user-v1.html")
			log.Infof("bookinfo productpage returned in %d ms", duration)
			if err != nil {
				fail = true
				break
			} 
		}
	})
	t.Run("A2", func(t *testing.T) {
		routeTrafficUser("bookinfo", "")
	//	checkHTTPResponse(host) 
	})
	
	if !fail {
		log.Infof("TC_03 passed")
	} else {
		log.Infof("TC_03 failed")
	}
	
	defer cleanup("bookinfo", "")
}
