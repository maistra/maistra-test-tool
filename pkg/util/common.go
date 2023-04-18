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
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"

	"golang.org/x/net/publicsuffix"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

var (
	cjopts = cookiejar.Options{PublicSuffixList: publicsuffix.List}
)

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

// CloseResponseBody ...
func CloseResponseBody(r *http.Response) {
	if r == nil {
		return
	}
	if err := r.Body.Close(); err != nil {
		log.Log.Error(err)
	}
}

// CompareHTTPResponse compares a HTTP Response body with a model HTML file
// modelFile is the file name. Not the file path.
// resources directory is github.com/maistra/maistra-test-tool/testdata/resources
func CompareHTTPResponse(body []byte, modelFile string) error {
	modelPath := filepath.Join(env.GetRootDir()+"/testdata/resources/html", modelFile)
	if err := CompareToFile(body, modelPath); err != nil {
		os.WriteFile("/tmp/diffbody", body, 0644)
		return err
	}
	return nil
}
