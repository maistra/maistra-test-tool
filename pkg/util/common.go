// Copyright 2021 Red Hat, Inc.
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

package util

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

	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

var (
	cjopts = cookiejar.Options{PublicSuffixList: publicsuffix.List}
)

// Inspect error handling function
func Inspect(err error, failureMsg, successMsg string, t *testing.T) {
	if err != nil {
		t.Fatalf("%s. Error %s", failureMsg, err)
	} else if successMsg != "" {
		log.Log.Info(successMsg)
	}
}

// GetCookieJar ...
func GetCookieJar(username, pass, gateway string) *cookiejar.Jar {
	jar, err := SetupCookieJar(username, pass, gateway)
	if err != nil {
		log.Log.Errorf("failed to get login user cookiejar: %v", err)
		return nil
	}
	return jar
}

// SetupCookieJar sends http post request with user login
func SetupCookieJar(username, pass, gateway string) (*cookiejar.Jar, error) {
	jar, err := cookiejar.New(&cjopts)
	if err != nil {
		return nil, fmt.Errorf("failed building cookiejar: %v", err)
	}
	client := http.Client{Jar: jar}
	loginURL, err := url.Parse(fmt.Sprintf("%s/login", gateway))
	if err != nil {
		return nil, err
	}
	resp, err := client.PostForm(loginURL.String(), url.Values{
		"password": {pass},
		"username": {username},
	})
	if err != nil {
		return nil, fmt.Errorf("failed login for user '%s': %v", username, err)
	}
	if !containsSessionCookie(jar.Cookies(loginURL)) {
		return nil, fmt.Errorf("no session cookie returned by login URL %s", loginURL)
	}
	resp.Body.Close()
	return jar, nil
}

func containsSessionCookie(cookies []*http.Cookie) bool {
	for _, c := range cookies {
		if c.Name == "session" {
			return true
		}
	}
	return false
}

// GetWithCookieJar constructs reqeusts with login user cookie and returns a http response
func GetWithCookieJar(url string, jar *cookiejar.Jar) (*http.Response, error) {
	// Declare http client
	client := &http.Client{Jar: jar}

	// Declare HTTP Method and Url
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// CloseResponseBody ...
func CloseResponseBody(r *http.Response) {
	if r == nil {
		return
	}
	if err := r.Body.Close(); err != nil {
		log.Log.Error(err)
	}
}

// GetHTTPResponse returns a HTTP Response object and response time in milliseconds.
// if cookiejar is nil, it sends a HTTP GET Request without user login.
func GetHTTPResponse(url string, jar *cookiejar.Jar) (*http.Response, int, error) {
	var resp *http.Response
	var duration int
	var err error

	if jar != nil {
		startT := time.Now()
		resp, err = GetWithCookieJar(url, jar)
		duration = int(time.Since(startT) / (time.Second / time.Microsecond))
	} else {
		startT := time.Now()
		resp, err = http.Get(url)
		duration = int(time.Since(startT) / (time.Second / time.Microsecond))
	}
	return resp, duration, err
}

// CheckHTTPResponse200 returns an error if Response code is not 200
func CheckHTTPResponse200(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		log.Log.Errorf("Get response failed!")
		return fmt.Errorf("status code is %d", resp.StatusCode)
	}
	return nil
}

// SaveHTTPResponse writes  a Response body to a file dst
func SaveHTTPResponse(body []byte, dst string) error {
	log.Log.Infof("Write response body to file: %s", dst)
	if err := ioutil.WriteFile(dst, body, 0644); err != nil {
		return err
	}
	return nil
}

// CompareHTTPResponse compares a HTTP Response body with a model HTML file
// modelFile is the file name. Not the file path.
// resources directory is github.com/maistra/maistra-test-tool/testdata/resources
func CompareHTTPResponse(body []byte, modelFile string) error {
	modelPath := filepath.Join("../testdata/resources/html", modelFile)
	if err := CompareToFile(body, modelPath); err != nil {
		ioutil.WriteFile("/tmp/diffbody", body, 0644)
		return err
	}
	return nil
}

func Failf(t *testing.T, format string, a ...any) {
	Fail(t, fmt.Sprintf(format, a...))
}

func Fail(t *testing.T, str string) {
	t.Error(str)
	log.Log.Error(str)

}
