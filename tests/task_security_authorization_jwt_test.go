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

func cleanupAuthorizationJWT() {
	log.Info("# Cleanup ...")

	util.KubeDeleteContents("foo", jwtRequestPrincipal, kubeconfig)
	util.KubeDeleteContents("foo", jwtExampleRules, kubeconfig)
	time.Sleep(time.Duration(waitTime*6) * time.Second)
	cleanHttpbin("foo")
	cleanSleep("foo")
}

func TestAuthorizationJWT(t *testing.T) {
	defer cleanupAuthorizationJWT()
	defer recoverPanic(t)

	log.Info("Authorization with JWT")

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

	t.Run("Security_authorization_allow_valid_JWT_list-typed_claims", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents("foo", jwtExampleRules, kubeconfig)
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		log.Info("Verify a request with an invalid JWT is denied")

		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -s -o /dev/null -H "Authorization: Bearer invalidToken" -w "%%{http_code}\n"`, "foo")
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "401") {
			log.Errorf("Verify denied request Unexpected response: %s", msg)
			t.Errorf("Verify denied request Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		log.Info("Verify a request without a JWT is allowed")

		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Errorf("Verify request without JWT Unexpected response: %s", msg)
			t.Errorf("Verify request without JWT Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("Security_authorization_allow_JWT_requestPrincipal", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents("foo", jwtRequestPrincipal, kubeconfig)
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		log.Info("Download JWT token")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.6/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellMuteOutput(`curl %s -s`, jwtURL)
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		log.Info("Verify request with a valid JWT")

		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -s -o /dev/null -H "Authorization: Bearer %s" -w "%%{http_code}\n"`, "foo", token)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Errorf("Verify request with valid JWT Unexpected response: %s", msg)
			t.Errorf("Verify request with valid JWT Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		log.Info("Verify request without a JWT is denied")

		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -s -o /dev/null -w "%%{http_code}\n"`, "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Errorf("Verify request without valid JWT Unexpected response: %s", msg)
			t.Errorf("Verify request without valid JWT Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("Security_authorization_allow_JWT_claims_group", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents("foo", jwtClaimsGroup, kubeconfig)
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		log.Info("Download JWT token and sets the groups claims")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.6/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellMuteOutput(`curl %s -s`, jwtURL)
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		groupURL := "https://raw.githubusercontent.com/istio/istio/release-1.6/security/tools/jwt/samples/groups-scope.jwt"
		tokenGroup, err := util.ShellMuteOutput(`curl %s -s`, groupURL)
		tokenGroup = strings.Trim(tokenGroup, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		log.Info("Verify request with a JWT includes group1 claim")

		cmd := fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -s -o /dev/null -H "Authorization: Bearer %s" -w "%%{http_code}\n"`, "foo", tokenGroup)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Errorf("Verify request with JWT group1 claim Unexpected response: %s", msg)
			t.Errorf("Verify request with JWT group1 claim Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		log.Info("Verify request without groups claim JWT is denied")

		cmd = fmt.Sprintf(`curl "http://httpbin.%s:8000/headers" -s -o /dev/null -H "Authorization: Bearer %s" -w "%%{http_code}\n"`, "foo", token)
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			log.Errorf("Verify request without groups claim JWT Unexpected response: %s", msg)
			t.Errorf("Verify request without groups claim JWT Unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})
}
