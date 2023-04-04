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

package authorization

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func cleanupAuthorDeny() {
	log.Log.Info("Cleanup")
	util.KubeDeleteContents("foo", AllowPathIPPolicy)
	util.KubeDeleteContents("foo", DenyHeaderNotAdminPolicy)
	util.KubeDeleteContents("foo", DenyGETPolicy)
	time.Sleep(time.Duration(40) * time.Second)
	sleep := examples.Sleep{"foo"}
	httpbin := examples.Httpbin{"foo"}
	sleep.Uninstall()
	httpbin.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestAuthorDeny(t *testing.T) {
	test.NewTest(t).Id("T23").Groups(test.Full, test.InterOp).NotRefactoredYet()

	defer cleanupAuthorDeny()
	defer util.RecoverPanic(t)

	log.Log.Info("Authorization policies with a deny action")
	httpbin := examples.Httpbin{"foo"}
	httpbin.Install()
	sleep := examples.Sleep{"foo"}
	sleep.Install()
	sleepPod, err := util.GetPodName("foo", "app=sleep")
	util.Inspect(err, "Failed to get sleep pod name", "", t)
	cmd := fmt.Sprintf(`curl http://httpbin.foo:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`)
	msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
	util.Inspect(err, "Failed to get response", "", t)
	if !strings.Contains(msg, "200") {
		log.Log.Errorf("Verify setup -- Unexpected response code: %s", msg)
	} else {
		log.Log.Infof("Success. Get expected response: %s", msg)
	}

	t.Run("Security_authorization_explicitly_deny_request", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Explicitly deny a request")
		util.KubeApplyContents("foo", DenyGETPolicy)
		time.Sleep(time.Duration(50) * time.Second)

		log.Log.Info("Verify GET requests are denied")
		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/get" -X GET -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Log.Errorf("Verify deny GET requests Unexpected response: %s", msg)
			t.Errorf("Verify  deny GET requests Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}

		log.Log.Info("Verify POST requests are allowed")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/post" -X POST -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Log.Errorf("Verify POST requests Unexpected response: %s", msg)
			t.Errorf("Verify POST requests Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("Security_authorization_deny_request_header_check", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Apply a deny policy when header x-token value is not admin")
		util.KubeApplyContents("foo", DenyHeaderNotAdminPolicy)
		time.Sleep(time.Duration(50) * time.Second)

		log.Log.Info("Verify GET requests with HTTP header x-token: admin are allowed")
		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/get" -X GET -H "x-token: admin" -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Log.Errorf("Verify GET requests with HTTP header x-token: admin Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: admin Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}

		log.Log.Info("Verify GET requests with HTTP header x-token: guest are denied")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/get" -X GET -H "x-token: guest" -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Log.Errorf("Verify GET requests with HTTP header x-token: guest Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: guest Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("Security_authorization_allow_path_policy", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Apply a policy that allows requests at the ip path")
		util.KubeApplyContents("foo", AllowPathIPPolicy)
		time.Sleep(time.Duration(50) * time.Second)

		log.Log.Info("Verify GET requests with the HTTP header x-token: guest at path /ip are denied")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/ip" -X GET -H "x-token: guest" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Log.Errorf("Verify GET requests with HTTP header x-token: guest Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: guest Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}

		log.Log.Info("Verify GET requests with the HTTP header x-token: admin at path /ip are allowed")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/ip" -X GET -H "x-token: admin" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Log.Errorf("Verify GET requests with HTTP header x-token: admin at path /ip Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: admin at path /ip Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}

		log.Log.Info("Verify GET requests with the HTTP header x-token: admin at path /get are denied")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/get" -X GET -H "x-token: admin" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Log.Errorf("Verify GET requests with HTTP header x-token: admin at path /get Unexpected response: %s", msg)
			t.Errorf("Verify GET requests with HTTP header x-token: admin at path /get Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}
	})
}
