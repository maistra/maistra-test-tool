// Copyright 2019 Red Hat, Inc.
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

package maistra

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup19(kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.ShellMuteOutput("kubectl delete -n foo servicerolebinding bind-httpbin-viewer")
	util.ShellMuteOutput("kubectl delete -n foo servicerole httpbin-viewer")
	util.ShellMuteOutput("kubectl delete -n foo clusterrbacconfig default")
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
	util.ShellMuteOutput("kubectl delete -n foo policy jwt-example")
	util.KubeDelete("foo", httpbinYaml, kubeconfig)
	util.KubeDelete("foo", sleepYaml, kubeconfig)
	util.ShellMuteOutput("kubectl delete policy -n %s default", "foo")
	util.ShellMuteOutput("kubectl delete destinationrule -n %s default", "foo")
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
	//util.DeleteNamespace("foo", kubeconfig)
}

func Test19(t *testing.T) {
	defer cleanup19(kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_19 Authorization for groups and list claims")
	util.Inspect(util.CreateNamespace("foo", kubeconfigFile), "failed to create namespace", "", t)
	util.OcGrantPermission("default", "foo", kubeconfigFile)

	log.Info("Enable mTLS")
	util.Inspect(util.KubeApplyContents("foo", mtlsPolicy, kubeconfigFile), "failed to apply policy", "", t)
	mtlsRule := strings.Replace(mtlsRuleTemplate, "@token@", "foo", -1)
	util.Inspect(util.KubeApplyContents("foo", mtlsRule, kubeconfigFile), "failed to apply rule", "", t)
	log.Info("Waiting... Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)

	util.Inspect(deployHttpbin("foo", kubeconfigFile), "failed to deploy httpbin", "", t)
	util.Inspect(deploySleep("foo", kubeconfigFile), "failed to deploy sleep", "", t)

	token, err := util.ShellSilent("curl %s -s", jwtURLGroup)
	token = strings.Trim(token, "\n")
	util.Inspect(err, "failed to get JWT token", "", t)
	sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfigFile)
	util.Inspect(err, "failed to get sleep pod name", "", t)

	t.Run("verify_setup_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		time.Sleep(time.Duration(5) * time.Second)
		cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"%%{http_code}\"", "foo")
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, false, kubeconfigFile)
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, false, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Verify setup -- Unexpected response code: %s", msg)
			log.Errorf("Verify setup -- Unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("config_jwt_mtls_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Configure JWT authentication with mutual TLS")
		util.Inspect(util.KubeApplyContents("foo", fooJWTPolicy2, kubeconfigFile), "failed to apply policy", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)

		cmd := fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"%%{http_code}\" --header \"Authorization: Bearer %s\"", token)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		cmd = fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"%%{http_code}\"", "foo")
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, false, kubeconfigFile)
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, false, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "401") {
			t.Errorf("Expected: 401 -- Unexpected response code: %s", msg)
			log.Errorf("Expected: 401 -- Unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("group_rbac_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Configure groups-based authorization")
		util.Inspect(util.KubeApplyContents("foo", fooRBAC, kubeconfigFile), "failed to apply clusterrbacconfig", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)

		cmd := fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"%%{http_code}\" --header \"Authorization: Bearer %s\"", token)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "403") {
			t.Errorf("Expected: 403; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 403; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		util.Inspect(util.KubeApplyContents("foo", fooRBACRole, kubeconfigFile), "failed to apply servicerole", "", t)
		util.Inspect(util.KubeApplyContentSilent("foo", fooRBACRoleBinding, kubeconfigFile), "failed to apply servicerolebinding", "", t)
		log.Info("Waiting... Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)

		cmd = fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"%%{http_code}\" --header \"Authorization: Bearer %s\"", token)
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		util.ShellMuteOutput("kubectl delete -n foo servicerolebinding bind-httpbin-viewer")
		util.Inspect(util.KubeApplyContentSilent("foo", fooRBACRoleBinding2, kubeconfigFile), "failed to apply servicerolebinding", "", t)
		log.Info("Waiting... Sleep 30 seconds...")
		time.Sleep(time.Duration(30) * time.Second)

		cmd = fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"%%{http_code}\" --header \"Authorization: Bearer %s\"", token)
		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	t.Run("list_claims_rbac_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Configure the authorization of list-typed claims")
		util.Inspect(util.KubeApplyContents("foo", fooRBACRoleBinding3, kubeconfigFile), "failed to apply servicerolebinding", "", t)
		log.Info("Waiting... Sleep 30 seconds...")
		time.Sleep(time.Duration(30) * time.Second)

		cmd := fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"%%{http_code}\" --header \"Authorization: Bearer %s\"", token)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

}
