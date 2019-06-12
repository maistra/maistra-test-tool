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

func cleanup15(kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete("foo", httpbinYaml, kubeconfig)
	util.KubeDelete("foo", sleepYaml, kubeconfig)
	util.KubeDelete("bar", httpbinYaml, kubeconfig)
	util.KubeDelete("bar", sleepYaml, kubeconfig)
	util.KubeDelete("legacy", sleepLegacyYaml, kubeconfig)

	util.ShellMuteOutput("kubectl delete policy example-httpbin-permissive -n foo")
	util.ShellMuteOutput("kubectl delete destinationrule example-httpbin-istio-client-mtls -n foo")

	util.DeleteNamespace("foo bar legacy", kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}

func setup15(kubeconfig string) error {
	if err := util.KubeApply("foo", httpbinYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("foo", sleepYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("bar", httpbinYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("bar", sleepYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("legacy", sleepLegacyYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning("foo", "app=httpbin", kubeconfigFile); err != nil {
		return err
	}
	if err := util.CheckPodRunning("foo", "app=sleep", kubeconfigFile); err != nil {
		return err
	}
	if err := util.CheckPodRunning("bar", "app=httpbin", kubeconfigFile); err != nil {
		return err
	}
	if err := util.CheckPodRunning("bar", "app=sleep", kubeconfigFile); err != nil {
		return err
	}
	if err := util.CheckPodRunning("legacy", "app=sleep", kubeconfigFile); err != nil {
		return err
	}
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(15) * time.Second)
	return nil
}

func Test15(t *testing.T) {
	defer cleanup15(kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_15 Mutual TLS Migration")
	namespaces := []string{"foo", "bar", "legacy"}

	// Create namespaces
	for _, ns := range namespaces {
		util.Inspect(util.CreateNamespace(ns, kubeconfigFile), "failed to create namespace", "", t)
		util.OcGrantPermission("default", ns, kubeconfigFile)
	}
	time.Sleep(time.Duration(5) * time.Second)
	util.Inspect(setup15(kubeconfigFile), "failed to apply deployments", "", t)

	t.Run("verify_setup", func(t *testing.T) {
		log.Info("Verify setup")

		for _, from := range namespaces {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
			util.Inspect(err, "failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.foo: %%{http_code}\"", from)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
			util.Inspect(err, "failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				t.Errorf("Verify setup expected 200; Got unexpected response code: %s", msg)
				log.Errorf("Verify setup expected 200; Got unexpected response code: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	})

	t.Run("mTLS_and_plain_text", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Configure the server to accept both mutual TLS and plain text traffic")
		util.Inspect(util.KubeApplyContents("foo", tlsPermissivePolicy, kubeconfigFile), "failed to apply foo permissive policy", "", t)
		time.Sleep(time.Duration(5) * time.Second)

		for _, from := range namespaces {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
			util.Inspect(err, "failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.foo: %%{http_code}\"", from)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
			util.Inspect(err, "failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				t.Errorf("mTLS and plain text expected 200; Got unexpected response code: %s", msg)
				log.Errorf("mTLS and plain text expected 200; Got unexpected response code: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	})

	t.Run("mTLS", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Configure clients to send mutual TLS traffic")
		util.Inspect(util.KubeApplyContents("foo", tlsRule, kubeconfigFile), "failed to apply foo tls rule", "", t)
		time.Sleep(time.Duration(5) * time.Second)

		for _, from := range namespaces {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
			util.Inspect(err, "failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.foo: %%{http_code}\"", from)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
			util.Inspect(err, "failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				t.Errorf("mTLS traffic expected 200; Got unexpected response code: %s", msg)
				log.Errorf("mTLS traffic expected 200; Got unexpected response code: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	})

	t.Run("lock_down_mTLS", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Lock down to mutual TLS")
		util.Inspect(util.KubeApplyContents("foo", tlsStrictPolicy, kubeconfigFile), "failed to apply foo tls strict policy", "", t)
		time.Sleep(time.Duration(10) * time.Second)

		for _, from := range namespaces {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
			util.Inspect(err, "failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.foo: %%{http_code}\"", from)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)

			if from == "legacy" {
				if err != nil {
					log.Infof("Expected sleep.legacy fails: %v", err)
				} else {
					t.Errorf("Expected sleep.legacy fails; Got unexpected response: %s", msg)
					log.Errorf("Expected sleep.legacy fails; Got unexpected response: %s", msg)
				}
				continue
			}

			util.Inspect(err, "failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				t.Errorf("mTLS traffic expected 200; Got unexpected response code: %s", msg)
				log.Errorf("mTLS traffic expected 200; Got unexpected response code: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	})

}
