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

package app

import (
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type sleep struct {
	ns            string
	injectSidecar bool
	tproxy        bool
}

var _ App = &sleep{}

func Sleep(ns string) App {
	return &sleep{ns: ns, injectSidecar: true}
}

func SleepNoSidecar(ns string) App {
	return &sleep{ns: ns, injectSidecar: false}
}

func SleepTroxy(ns string) App {
	return &sleep{ns: ns, injectSidecar: true, tproxy: true}
}

func (a *sleep) Name() string {
	return "sleep"
}

func (a *sleep) Namespace() string {
	return a.ns
}

func (a *sleep) Install(t test.TestHelper) {
	t.T().Helper()
	oc.ApplyTemplate(t, a.ns, sleepTemplate, a.values(t))
}

func (a *sleep) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFromTemplate(t, a.ns, sleepTemplate, a.values(t))
}

func (a *sleep) values(t test.TestHelper) map[string]interface{} {
	proxy := oc.GetProxy(t)
	return map[string]interface{}{
		"InjectSidecar": a.injectSidecar,
		"HttpProxy":     proxy.HTTPProxy,
		"HttpsProxy":    proxy.HTTPSProxy,
		"NoProxy":       proxy.NoProxy,
	}
}

func (a *sleep) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "sleep")
}

type CurlOpts struct {
	Method  string
	Headers []string
	Options []string
}

func ExecInSleepPod(t test.TestHelper, ns string, command string, checks ...common.CheckFunc) {
	t.T().Helper()
	retry.UntilSuccess(t, func(t test.TestHelper) {
		t.T().Helper()
		oc.Exec(t, pod.MatchingSelector("app=sleep", ns), "sleep", command, checks...)
	})
}

func AssertSleepPodRequestSuccess(t test.TestHelper, sleepNamespace string, url string, opts ...CurlOpts) {
	assertSleepPodRequestResponse(t, sleepNamespace, url, "200", opts...)
}

func AssertSleepPodRequestFailure(t test.TestHelper, sleepNamespace string, url string, opts ...CurlOpts) {
	assertSleepPodRequestResponse(t, sleepNamespace, url, curlFailedMessage, opts...)
}

func AssertSleepPodRequestForbidden(t test.TestHelper, sleepNamespace string, url string, opts ...CurlOpts) {
	assertSleepPodRequestResponse(t, sleepNamespace, url, "403", opts...)
}

func AssertSleepPodRequestUnauthorized(t test.TestHelper, sleepNamespace string, url string, opts ...CurlOpts) {
	assertSleepPodRequestResponse(t, sleepNamespace, url, "401", opts...)
}

func AssertSleepPodZeroesPlaceholder(t test.TestHelper, sleepNamespace string, url string, opts ...CurlOpts) {
	assertSleepPodRequestResponse(t, sleepNamespace, url, "000", opts...)
}

func assertSleepPodRequestResponse(t test.TestHelper, sleepNamespace, url, expected string, opts ...CurlOpts) {
	command := buildCurlCmd(url, opts...)
	ExecInSleepPod(t, sleepNamespace, command,
		assert.OutputContains(expected,
			fmt.Sprintf("Got expected \"%s\"", expected),
			fmt.Sprintf("Expect \"%s\", but got a different response", expected)))
}

func buildCurlCmd(url string, opts ...CurlOpts) string {
	var opt CurlOpts
	if len(opts) > 0 {
		opt = opts[0]
	} else {
		opt = CurlOpts{}
	}

	method, headers, options := "", "", ""
	if opt.Method == "" {
		method = "GET"
	} else {
		method = opt.Method
	}
	if opt.Options != nil {
		for _, option := range opt.Options {
			options += " " + option
		}
	}
	if opt.Headers != nil {
		for _, header := range opt.Headers {
			headers += fmt.Sprintf(` -H "%s"`, header)
		}
	}

	return fmt.Sprintf(`curl -sS %s%s -X %s -o /dev/null -w "%%{http_code}" %s 2>/dev/null || echo %s`, options, headers, method, url, curlFailedMessage)
}

const curlFailedMessage = "CURL_FAILED"

const sleepTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sleep
---
apiVersion: v1
kind: Service
metadata:
  name: sleep
  labels:
    app: sleep
spec:
  ports:
  - port: 80
    name: http
  selector:
    app: sleep
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sleep
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sleep
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "{{ .InjectSidecar }}"
      {{ if .Tproxy }}
        sidecar.istio.io/interceptionMode: TPROXY
      {{ end }}
      labels:
        app: sleep
    spec:
      terminationGracePeriodSeconds: 0
      serviceAccountName: sleep
      containers:
      - name: sleep
        image: {{ image "sleep" }} 
        command: ["/bin/sleep", "3650d"]
        env:
        - name: HTTPS_PROXY
          value: {{ .HttpsProxy }}
        - name: HTTP_PROXY
          value: {{ .HttpProxy }}
        - name: NO_PROXY
          value: {{ .NoProxy }}
        volumeMounts:
        - mountPath: /etc/sleep/tls
          name: secret-volume
      volumes:
      - name: secret-volume
        secret:
          secretName: sleep-secret
          optional: true
`
