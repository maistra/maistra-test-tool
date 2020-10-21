// Copyright 2020 Red Hat, Inc.
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

package tests

import (
	"fmt"
	"maistra/util"
	"strings"
	"testing"
	"time"

	"istio.io/pkg/log"
)

func cleanupAuthorizationDeny() {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents("foo", allowPathPolicy, kubeconfig)
	util.KubeDeleteContents("foo", GetPolicyDeny, kubeconfig)
	time.Sleep(time.Duration(waitTime*6) * time.Second)
	cleanHttpbin("foo")
	cleanSleep("foo")
}

func TestAuthorizationDeny(t *testing.T) {
	defer cleanupAuthorizationDeny()
	defer recoverPanic(t)

	log.Info("Authorization policies with a deny action")

	deployHttpbin("foo")
	deploySleep("foo")

	log.Info("Verify setup")
	sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfig)
	util.Inspect(err, "Failed to get sleep pod name", "", t)
	cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "%%{http_code}\n"`, "foo")
	msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
	util.Inspect(err, "Failed to get response", "", t)
	if !strings.Contains(msg, "200") {
		log.Errorf("Verify setup Unexpected response: %s", msg)
		t.Errorf("Verify setup Unexpected response: %s", msg)
	} else {
		log.Infof("Success. Get expected response: %s", msg)
	}

	t.Run("Security_authorization_explicitly_deny_request", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents("foo", GetPolicyDeny, kubeconfig)
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		log.Info("Verify GET requests are denied")
		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/get" -X GET -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Errorf("Verify deny GET requests Unexpected response: %s", msg)
			t.Errorf("Verify  deny GET requests Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		log.Info("Verify POST requests are allowed")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/post" -X POST -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Errorf("Verify POST requests Unexpected response: %s", msg)
			t.Errorf("Verify POST requests Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("Security_authorization_deny_request_header_check", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents("foo", headerValuePolicyDeny, kubeconfig)
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		log.Info("Verify GET requests with HTTP header x-token: admin are allowed")
		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/get" -X GET -H "x-token: admin" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Errorf("Verify GET requests with HTTP header x-token: admin Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: admin Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		log.Info("Verify GET requests with HTTP header x-token: guest are denied")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/get" -X GET -H "x-token: guest" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Errorf("Verify GET requests with HTTP header x-token: guest Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: guest Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("Security_authorization_allow_path_policy", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents("foo", allowPathPolicy, kubeconfig)
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		log.Info("Verify GET requests with the HTTP header x-token: guest at path /ip are denied")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/ip" -X GET -H "x-token: guest" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Errorf("Verify GET requests with HTTP header x-token: guest Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: guest Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		log.Info("Verify GET requests with the HTTP header x-token: admin at path /ip are allowed")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/ip" -X GET -H "x-token: admin" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Errorf("Verify GET requests with HTTP header x-token: admin at path /ip Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: admin at path /ip Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		log.Info("Verify GET requests with the HTTP header x-token: admin at path /get are denied")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/get" -X GET -H "x-token: admin" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Errorf("Verify GET requests with HTTP header x-token: admin at path /get Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: admin at path /get Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})
}
