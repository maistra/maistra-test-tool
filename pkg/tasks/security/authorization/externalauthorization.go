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

package authorizaton

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

// referance doc : https://istio.io/latest/docs/tasks/security/authorization/authz-custom/

func cleanupExtAuth() {
	util.Log.Info("Cleanup Ext Auth")
	sleep := examples.Sleep{"foo"}
	httpbin := examples.Httpbin{"foo"}
	sleep.Uninstall()
	httpbin.Uninstall()
	util.KubeDeleteContents("foo", ExternalAuthzService)
	util.KubeDeleteContents("foo", ExternalRoute)
	time.Sleep(time.Duration(20) * time.Second)
	util.DeleteSMCP(smcpName, meshNamespace)
}

func TestExtAuth(t *testing.T) {
	defer cleanupExtAuth()
	defer util.RecoverPanic(t)

	util.Log.Info("Authorization with External Authorization")
	sleep := examples.Sleep{"foo"}
	sleep.Install()
	httpbin := examples.Httpbin{"foo"}
	httpbin.Install()

	sleepPod, err := util.GetPodName("foo", "app=sleep")
	if err != nil {
		util.Inspect(err, "Failed to get sleep pod name", "", t)
	}
	cmd := fmt.Sprintf(`curl http://httpbin.foo:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`)
	msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
	util.Inspect(err, "Failed to get response", "", t)
	util.Log.Info("Verify that sleep can access httpbin")
	if !strings.Contains(msg, "200") {
		util.Log.Errorf("Verify setup -- Unexpected response code: %s", msg)
	} else {
		util.Log.Infof("Success. Get expected response: %s", msg)
	}

	t.Run("Deploy the External Authorizer", func(t *testing.T) {

		util.Log.Info("Deploy the sample External Authorizer")
		util.KubeApplyContents("foo", ExternalAuthzService)
		time.Sleep(time.Duration(50) * time.Second)

		util.Log.Info("Verfiy the sample external authorizer is up and running")
		extAuthPod, err := util.GetPodName("foo", "app=ext-authz")
		util.Inspect(err, "Failed to get ext-authz pod name", "", t)
		extAuthPodverf, err := util.Shell("oc logs %s -n foo -c ext-authz", extAuthPod)

		if strings.Contains(extAuthPodverf, "Starting HTTP server at [::]:8000") && strings.Contains(extAuthPodverf, "Starting gRPC server at [::]:9000") {
			util.Log.Infof("Success the sample external authorizer is up and running")
		} else {
			util.Log.Errorf("Failed the sample external authorizer is not up and not running")
		}

	})
	t.Run("Define the external authorizer", func(t *testing.T) {

		util.Log.Info("Edit configmap for two external providers")
		util.Shell(`kubectl patch -n %s configmap/istio-%s --type merge -p '{"data": {"mesh": "# Add the following content to define the external authorizers.\nextensionProviders:\n- name: \"sample-ext-authz-grpc\"\n  envoyExtAuthzGrpc:\n    service: \"ext-authz.foo.svc.cluster.local\"\n    port: \"9000\"\n- name: \"sample-ext-authz-http\"\n  envoyExtAuthzHttp:\n    service: \"ext-authz.foo.svc.cluster.local\"\n    port: \"8000\"\n    includeRequestHeadersInCheck: [\"x-ext-authz\"]"}}'`, meshNamespace, smcpName)
	})

	t.Run("Enable with external authorization", func(t *testing.T) {

		util.Log.Info("Enable with external authorization")
		util.KubeApplyContents("foo", ExternalRoute)

		util.Log.Info("Verfiy the header with deny server")
		deny := fmt.Sprintf(`curl "http://httpbin.foo:8000/headers" -H "x-ext-authz: deny" -s`)
		extAuthDeny, err := util.Shell("kubectl exec %s -c sleep -n foo -- %s", sleepPod, deny)
		util.Inspect(err, "Failed to run the command", "", t)
		if strings.Contains(extAuthDeny, "denied by ext_authz for not found header `x-ext-authz: allow` in the request") {
			util.Log.Infof("Success, verfication the header with deny server")
		} else {
			util.Log.Errorf("verfication failed with deny server")
		}

		util.Log.Info("Verfiy the header with allow server")

		type Headers struct {
			Accept     string
			Host       string
			UserAgent  string `json:"User-Agent"`
			XB3Sampled string `json:"X-B3-Sampled"`
			XB3Spanid  string `json:"X-B3-Spanid"`
			XB3Traceid string `json:"X-B3-Traceid"`
			XExtAuthz  string `json:"X-Ext-Authz"`
		}

		type Response struct {
			Headers Headers
		}

		var response Response
		allow := fmt.Sprintf(`curl "http://httpbin.foo:8000/headers" -H "x-ext-authz: allow" -s`)
		response1, err := util.Shell("kubectl exec %s -c sleep -n foo -- %s", sleepPod, allow)

		json.Unmarshal([]byte(response1), &response)
		if response.Headers.XExtAuthz == "allow" && response.Headers.Host == "httpbin.foo:8000" && response.Headers.Accept == "*/*" {
			util.Log.Infof("Success, verfication the header with allow server")
		} else {
			util.Log.Errorf("Failed, verfication the header with allow server")
		}

		util.Log.Info("Verfiy request to path /ip is allowed and does not trigger the external authorization")
		cmds := fmt.Sprintf(`curl http://httpbin.foo:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`)
		msges, err := util.PodExec("foo", sleepPod, "sleep", cmds, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msges, "200") {
			util.Log.Errorf("Verify setup -- Unexpected response code: %s", msges)
		} else {
			util.Log.Infof("Success. Get expected response: %s", msges)
		}

		extAuthPod, err := util.GetPodName("foo", "app=ext-authz")
		extAuthPodverfs, err := util.Shell("oc logs %s -n foo -c ext-authz", extAuthPod)
		if strings.Contains(extAuthPodverfs, "Starting HTTP server at [::]:8000") && strings.Contains(extAuthPodverfs, "Starting gRPC server at [::]:9000") && response.Headers.XExtAuthz == "allow" && response.Headers.Host == "httpbin.foo:8000" && response.Headers.Accept == "*/*" {
			util.Log.Infof("Success, log of the sample ext_authz server to confirm it was called twice (for the two requests)")
		} else {
			util.Log.Errorf("Failed to get the 2 request from ext_authz")
		}
	})
}
