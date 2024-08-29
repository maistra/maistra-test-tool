// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package curl

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type HTTPResponseCheckFunc func(t test.TestHelper, response *http.Response, responseBody []byte, responseErr error, duration time.Duration)

func Request(t test.TestHelper, url string, requestOption RequestOption, checks ...HTTPResponseCheckFunc) []byte {
	t.T().Helper()
	if requestOption == nil {
		requestOption = NilRequestOption{}
	}

	// t.Logf("HTTP request: %s", url)

	startT := time.Now()
	client := &http.Client{}
	if err := requestOption.ApplyToClient(client); err != nil {
		t.Fatalf("failed to modify client: %v", err)
	}

	// Declare HTTP Method and Url
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create HTTP Request: %v", err)
	}
	if err := requestOption.ApplyToRequest(req); err != nil {
		t.Fatalf("failed to modify request: %v", err)
	}

	resp, resp_err := client.Do(req)
	if resp_err != nil {
		t.Logf("failed to get HTTP Response: %v", resp_err)
	}

	var responseBody []byte
	if resp != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("failed to close response body: %v", err)
			}
		}()
		responseBody, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}
	}

	duration := time.Since(startT)
	for _, check := range checks {
		check(t, resp, responseBody, resp_err, duration)
	}
	return responseBody
}

type RequestOption interface {
	ApplyToRequest(req *http.Request) error
	ApplyToClient(client *http.Client) error
}

type cookieJarClientModifier struct {
	CookieJar http.CookieJar
}

var _ RequestOption = cookieJarClientModifier{}

func WithCookieJar(jar *cookiejar.Jar) RequestOption {
	return cookieJarClientModifier{CookieJar: jar}
}

func (w cookieJarClientModifier) ApplyToRequest(req *http.Request) error {
	return nil
}

func (w cookieJarClientModifier) ApplyToClient(client *http.Client) error {
	client.Jar = w.CookieJar
	return nil
}

type NilRequestOption struct{}

var _ RequestOption = NilRequestOption{}

func (n NilRequestOption) ApplyToClient(client *http.Client) error {
	return nil
}

func (n NilRequestOption) ApplyToRequest(req *http.Request) error {
	return nil
}
