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
	"time"

	"golang.org/x/net/publicsuffix"
	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)

var (
	cjopts = cookiejar.Options {
		PublicSuffixList: publicsuffix.List,
	}
)


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

func checkRoutingResponse(jar *cookiejar.Jar, url, modelFile string) (int, error) {
	var resp *http.Response
	var err error
	startT := time.Now()
	if jar != nil {
		resp, err = getWithCookieJar(fmt.Sprintf(url), jar)
	} else {
		resp, err = http.Get(fmt.Sprintf(url))
	}
	
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
