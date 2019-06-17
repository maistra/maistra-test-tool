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
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup13(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, httpbinAllv1Yaml, kubeconfig)
	util.KubeDelete(namespace, httpbinMirrorv2Yaml, kubeconfig)
	util.KubeDelete(namespace, httpbinServiceYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinv1Yaml, kubeconfig)
	util.KubeDelete(namespace, httpbinv2Yaml, kubeconfig)
	util.KubeDelete(namespace, sleepv2Yaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func setup13(namespace, kubeconfig string) error {
	log.Info("# Deploy Httpbin v1, v2 and sleep v2")
	if err := util.KubeApply(namespace, httpbinv1Yaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply(namespace, httpbinv2Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(namespace, "app=httpbin,version=v1", kubeconfig); err != nil {
		return err
	}
	if err := util.CheckPodRunning(namespace, "app=httpbin,version=v2", kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply(namespace, httpbinServiceYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.KubeApply(namespace, sleepv2Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)

	err := util.CheckPodRunning(namespace, "app=sleep", kubeconfig)
	return err
}

func Test13(t *testing.T) {
	defer cleanup13(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Info("# TC_13 Mirroring")
	util.Inspect(setup13(testNamespace, kubeconfigFile), "failed to deploy samples", "", t)

	t.Run("no_mirror_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		util.Inspect(util.KubeApply(testNamespace, httpbinAllv1Yaml, kubeconfigFile), "failed to apply rule", "", t)
		log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)

		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfigFile)
		util.Inspect(err, "failed to get sleep pod name", "", t)
		_, err = util.PodExec(testNamespace, sleepPod, "sleep", "sh -c 'curl  http://httpbin:8080/headers' | python -m json.tool", true, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)

		// check httpbin v1 logs
		v1Pod, err := util.GetPodName(testNamespace, "app=httpbin,version=v1", kubeconfigFile)
		util.Inspect(err, "failed to get httpbin v1 pod name", "", t)
		v1msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", testNamespace, v1Pod, "httpbin")
		util.Inspect(err, "failed to get httpbin v1 log", "", t)
		// check httpbin v2 logs
		v2Pod, err := util.GetPodName(testNamespace, "app=httpbin,version=v2", kubeconfigFile)
		util.Inspect(err, "failed to get httpbin v2 pod name", "", t)
		v2msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", testNamespace, v2Pod, "httpbin")
		util.Inspect(err, "failed to get httpbin v2 log", "", t)
		if strings.Contains(v1msg, "\"GET /headers HTTP/1.1\" 200") && !strings.Contains(v2msg, "\"GET /headers HTTP/1.1\" 200") {
			log.Info("Success. v1 an v2 logs are expected")
		} else {
			t.Errorf("Error. v1 log: %s\n v2 log: %s", v1msg, v2msg)
		}
	})

	t.Run("mirror_v2_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		util.Inspect(util.KubeApply(testNamespace, httpbinMirrorv2Yaml, kubeconfigFile), "failed to apply rule", "", t)
		log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)

		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfigFile)
		util.Inspect(err, "failed to get sleep pod name", "", t)
		_, err = util.PodExec(testNamespace, sleepPod, "sleep", "sh -c 'curl  http://httpbin:8080/headers' | python -m json.tool", true, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)

		// check httpbin v1 logs
		v1Pod, err := util.GetPodName(testNamespace, "app=httpbin,version=v1", kubeconfigFile)
		util.Inspect(err, "failed to get httpbin v1 pod name", "", t)
		v1msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", testNamespace, v1Pod, "httpbin")
		util.Inspect(err, "failed to get httpbin v1 log", "", t)
		// check httpbin v2 logs
		v2Pod, err := util.GetPodName(testNamespace, "app=httpbin,version=v2", kubeconfigFile)
		util.Inspect(err, "failed to get httpbin v2 pod name", "", t)
		v2msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", testNamespace, v2Pod, "httpbin")
		util.Inspect(err, "failed to get httpbin v2 log", "", t)
		if strings.Contains(v1msg, "\"GET /headers HTTP/1.1\" 200") && strings.Contains(v2msg, "\"GET /headers HTTP/1.1\" 200") {
			log.Info("Success. v1 an v2 logs are expected")
		} else {
			t.Errorf("Error. v1 log: %s\n v2 log: %s", v1msg, v2msg)
		}
	})

}
