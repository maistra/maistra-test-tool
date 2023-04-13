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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

// RunTemplate renders a yaml template string in the yaml_configs.go file
func RunTemplate(tmpl string, input interface{}) string {
	if input == nil {
		log.Log.Fatal("input is nil")
	}

	tt, err := template.New("").
		Funcs(templateFuncMap).
		Parse(tmpl)
	if err != nil {
		log.Log.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tt.Execute(&buf, input); err != nil {
		log.Log.Fatal(err)
	}
	return buf.String()
}

func RunTemplateWithTestHelper(t test.TestHelper, tmpl string, input interface{}) string {
	if input == nil {
		t.Fatal("input is nil")
	}

	tt, err := template.New("").
		Funcs(templateFuncMap).
		Parse(tmpl)
	if err != nil {
		t.Fatalf("could not execute template: %v:\n%s", err, tmpl)
	}
	var buf bytes.Buffer
	if err := tt.Execute(&buf, input); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

var templateFuncMap = template.FuncMap{
	"toYaml": toYaml,
	"indent": indent,
	"until":  until,
}

func indent(spaces int, source string) string {
	res := strings.Split(source, "\n")
	for i, line := range res {
		if i > 0 {
			res[i] = fmt.Sprintf(fmt.Sprintf("%% %ds%%s", spaces), "", line)
		}
	}
	return strings.Join(res, "\n")
}

func toYaml(value interface{}) string {
	y, err := yaml.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("Unable to marshal %v", value))
	}

	return string(y)
}

// recover from panic if one occurred. This allows cleanup to be executed after panic.
func RecoverPanic(t *testing.T) {
	t.Helper()
	if err := recover(); err != nil {
		t.Errorf("Test panic: %v", err)
	}
}

func IsWithinPercentage(count int, total int, rate float64, tolerance float64) bool {
	minimum := int((rate - tolerance) * float64(total))
	maximum := int((rate + tolerance) * float64(total))
	return count >= minimum && count <= maximum
}

// curl command with CA
func CurlWithCA(url, ingressHost, secureIngressPort, host, cacertFile string) (*http.Response, error) {
	// Load CA cert
	caCert, err := ioutil.ReadFile(cacertFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS transport
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	// Custom DialContext
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == host+":"+secureIngressPort {
			addr = ingressHost + ":" + secureIngressPort
		}
		return dialer.DialContext(ctx, network, addr)
	}

	// Setup HTTPS client
	client := &http.Client{Transport: transport}

	// GET something
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Set host
	req.Host = host
	req.Header.Set("Host", req.Host)
	// Get response
	return client.Do(req)
}

// check user key from header
func CheckUserGroup(url, ingress, ingressPort, user string) (*http.Response, error) {
	// Declare http client
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Set header key user
	req.Header.Set("user", user)
	// Get response
	return client.Do(req)
}

// Define an until function for template
func until(n int) []int {
	nums := make([]int, n)
	for i := 0; i < n; i++ {
		nums[i] = i
	}
	return nums
}

func GenerateStrings(prefix string, count int) []string {
	arr := make([]string, count)
	for i := 0; i < count; i++ {
		arr[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return arr
}
