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
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/net/publicsuffix"
	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)


var (
	cjopts 					= cookiejar.Options { PublicSuffixList: publicsuffix.List,}
	testRetryTimes			= 5

	host					= getOCPIngress("istio-ingressgateway","ingressgateway", "istio-system", "", "NodePort")
	productpageURL 			= fmt.Sprintf("http://%s/productpage", host)
	testUserJar				= getCookieJar(testUsername, "", "http://" + host)
)


func inspect(err error, fMsg, sMsg string, t *testing.T) {
	if err != nil {
		log.Errorf("%s. Error %s", fMsg, err)
		t.Error(err)
	} else if sMsg != "" {
		log.Info(sMsg)
	}
}

func getOCPIngress(serviceName, podLabel, namespace, kubeconfig string, serviceType string) string {
	host, err := util.GetIngress(serviceName, podLabel, namespace, kubeconfig, serviceType)
	if err != nil {
		log.Errorf("failed to get ingressgateway: %v", err)
		return ""
	}
	return host
}

func getCookieJar(username, pass, gateway string) *cookiejar.Jar {
	jar, err := setupCookieJar(username, pass, gateway)
	if err != nil {
		log.Errorf("failed to get login user cookiejar: %v", err)
		return nil
	}
	return jar
}

func setupCookieJar(username, pass, gateway string) (*cookiejar.Jar, error) {
	jar, err := cookiejar.New(&cjopts)
	if err != nil {
		return nil, fmt.Errorf("failed building cookiejar: %v", err)
	}
	client := http.Client{Jar: jar}
	resp, err := client.PostForm(fmt.Sprintf("%s/login", gateway), url.Values{
		"password": {pass},
		"username": {username},
	})
	if err != nil {
		return nil, fmt.Errorf("failed login for user '%s': %v", username, err)
	}
	resp.Body.Close()
	return jar, nil
}

func getWithCookieJar(url string, jar *cookiejar.Jar) (*http.Response, error) {
	// Declare http client
	client := &http.Client{Jar: jar}

	// Declare HTTP Method and Url
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func getWithHost(url string, host string) (*http.Response, error) {
	// Declare http client
	client := &http.Client{}

	// Declare HTTP Method and Url
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Host = host
	req.Header.Set("Host", req.Host)
	return client.Do(req)
}

func closeResponseBody(r *http.Response) {
	if err := r.Body.Close(); err != nil {
		log.Errora(err)
	}
}

// getHTTPResponse returns a HTTP Response object and response time in milliseconds.
// if cookiejar is nil, it sends a HTTP GET Request without user login.
func getHTTPResponse(url string, jar *cookiejar.Jar) (*http.Response, int, error) {
	var resp *http.Response
	var duration int
	var err error

	if jar != nil {
		startT := time.Now()
		resp, err = getWithCookieJar(url, jar)
		duration = int(time.Since(startT) / (time.Second / time.Microsecond))
	} else {
		startT := time.Now()
		resp, err = http.Get(url)
		duration = int(time.Since(startT) / (time.Second / time.Microsecond))
	}
	return resp, duration, err
}

// checkHTTPResposeCode returns an error if Response code is not 200
func checkHTTPResponse200(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		log.Errorf("Get response failed!")
		return fmt.Errorf("status code is %d", resp.StatusCode)
	}
	return nil
}

// saveHTTPResponse writes  a Response body to a file dst
func saveHTTPResponse(body []byte, dst string) error {
	log.Infof("Write response body to file: %s", dst)
	if err := ioutil.WriteFile(dst, body, 0644); err != nil {
		return err
	}
	return nil
}

// compareHTTPResponse compares a HTTP Response body with a model HTML file
// modelFile is the file name. Not the file path. 
func compareHTTPResponse(body []byte, modelFile string) error {
	modelPath := filepath.Join(modelDir, modelFile)
	if err := util.CompareToFile(body, modelPath); err != nil {
		ioutil.WriteFile("/tmp/diffbody", body, 0644)
		return err
	}
	return nil
}
