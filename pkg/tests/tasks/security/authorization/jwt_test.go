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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func cleanupAuthorJWT() {
	log.Log.Info("Cleanup")
	util.KubeDeleteContents("foo", JWTGroupClaimRule)
	util.KubeDeleteContents("foo", JWTRequireRule)
	util.KubeDeleteContents("foo", JWTExampleRule)
	time.Sleep(time.Duration(40) * time.Second)
	sleep := examples.Sleep{"foo"}
	httpbin := examples.Httpbin{"foo"}
	sleep.Uninstall()
	httpbin.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestAuthorJWT(t *testing.T) {
	test.NewTest(t).Id("T22").Groups(test.Full, test.InterOp).NotRefactoredYet()

	defer cleanupAuthorJWT()
	defer util.RecoverPanic(t)

	log.Log.Info("Authorization with JWT Token")
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

	t.Run("Security_authorization_allow_valid_JWT_list-typed_claims", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Allow requests with valid JWT and list-typed claims")
		util.KubeApplyContents("foo", JWTExampleRule)
		time.Sleep(time.Duration(50) * time.Second)

		log.Log.Info("Verify a request with an invalid JWT is denied")
		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -sS -o /dev/null -H "Authorization: Bearer invalidToken" -w "%%{http_code}\n"`, "foo")
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "401") {
			log.Log.Errorf("Verify denied request Unexpected response: %s", msg)
			t.Errorf("Verify denied request Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}

		log.Log.Info("Verify a request without a JWT is allowed")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Log.Errorf("Verify request without JWT Unexpected response: %s", msg)
			t.Errorf("Verify request without JWT Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("Security_authorization_allow_JWT_requestPrincipal", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Apply a policy requires all requests to have a valid JWT")
		util.KubeApplyContents("foo", JWTRequireRule)
		time.Sleep(time.Duration(50) * time.Second)

		log.Log.Info("Download JWT token")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellMuteOutput(`curl %s -s`, jwtURL)
		token = strings.Trim(token, "\n")
		// token, err = util.Shell(`echo %s | cut -d '.' -f2 - | base64 --decode -`, token)
		util.Inspect(err, "Failed to get JWT token", "", t)

		log.Log.Info("Verify request with a valid JWT")
		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -sS -o /dev/null -H "Authorization: Bearer %s" -w "%%{http_code}\n"`, "foo", token)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Log.Errorf("Verify request with valid JWT Unexpected response: %s", msg)
			t.Errorf("Verify request with valid JWT Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}

		log.Log.Info("Verify request without a JWT is denied")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Log.Errorf("Verify request without valid JWT Unexpected response: %s", msg)
			t.Errorf("Verify request without valid JWT Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("Security_authorization_allow_JWT_claims_group", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Apply a require jwt policy with a group claim")
		util.KubeApplyContents("foo", JWTGroupClaimRule)
		time.Sleep(time.Duration(50) * time.Second)

		log.Log.Info("Download JWT token and sets the groups claims")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellMuteOutput(`curl %s -s`, jwtURL)
		token = strings.Trim(token, "\n")
		// token, err = util.Shell(`echo %s | cut -d '.' -f2 - | base64 --decode -`, token)
		util.Inspect(err, "Failed to get JWT token", "", t)

		groupURL := "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/groups-scope.jwt"
		tokenGroup, err := util.ShellMuteOutput(`curl %s -s`, groupURL)
		tokenGroup = strings.Trim(tokenGroup, "\n")
		// tokenGroup, err = util.Shell(`echo %s | cut -d '.' -f2 - | base64 --decode -`, tokenGroup)
		util.Inspect(err, "Failed to get JWT token", "", t)

		log.Log.Info("Verify request with a JWT includes group1 claim")
		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -s -o /dev/null -H "Authorization: Bearer %s" -w "%%{http_code}\n"`, "foo", tokenGroup)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Log.Errorf("Verify request with JWT group1 claim Unexpected response: %s", msg)
			t.Errorf("Verify request with JWT group1 claim Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}

		log.Log.Info("Verify request without groups claim JWT is denied")
		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -s -o /dev/null -H "Authorization: Bearer %s" -w "%%{http_code}\n"`, "foo", token)
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Log.Errorf("Verify request without groups claim JWT Unexpected response: %s", msg)
			t.Errorf("Verify request without groups claim JWT Unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}
	})
}
