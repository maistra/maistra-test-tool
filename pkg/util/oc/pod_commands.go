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

package oc

import (
	"fmt"
	"math"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type NamespacedName struct {
	Namespace string
	Name      string
}

func NewNamespacedName(ns, name string) NamespacedName {
	return NamespacedName{
		Namespace: ns,
		Name:      name,
	}
}

type PodLocatorFunc func(t test.TestHelper, oc *OC) NamespacedName

func (o OC) Exec(t test.TestHelper, podLocator PodLocatorFunc, container string, cmd string, checks ...common.CheckFunc) string {
	t.T().Helper()
	pod := podLocator(t, &o)
	if pod.Name == "" || pod.Namespace == "" {
		t.Fatal("could not find pod using podLocatorFunc")
	}
	containerFlag := ""
	if container != "" {
		containerFlag = "-c " + container
	}
	return o.Invoke(t,
		fmt.Sprintf("kubectl exec -n %s %s %s -- %s", pod.Namespace, pod.Name, containerFlag, cmd),
		checks...)
}

func (o OC) GetPodIP(t test.TestHelper, podLocator PodLocatorFunc) string {
	t.T().Helper()
	pod := podLocator(t, &o)
	return o.Invoke(t, fmt.Sprintf("kubectl get pod -n %s %s -o jsonpath='{.status.podIP}'", pod.Namespace, pod.Name))
}

func (o OC) Logs(t test.TestHelper, podLocator PodLocatorFunc, container string, checks ...common.CheckFunc) {
	t.T().Helper()
	pod := podLocator(t, &o)
	o.Invoke(t,
		fmt.Sprintf("kubectl logs -n %s %s -c %s", pod.Namespace, pod.Name, container),
		checks...)
}

func (o OC) LogsSince(t test.TestHelper, start time.Time, podLocator PodLocatorFunc, container string, checks ...common.CheckFunc) {
	t.T().Helper()
	pod := podLocator(t, &o)
	o.Invoke(t,
		fmt.Sprintf("kubectl logs -n %s %s -c %s --since=%ds", pod.Namespace, pod.Name, container, int(math.Ceil(time.Since(start).Seconds()))),
		checks...)
}

func (o OC) LogsFromPods(t test.TestHelper, ns, selector string, checks ...common.CheckFunc) {
	t.T().Helper()
	o.Invoke(t,
		fmt.Sprintf("kubectl -n %s logs -l %s --all-containers --tail=-1", ns, selector),
		checks...)
}

func (o OC) WaitPodRunning(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	retry.UntilSuccessWithOptions(t, retry.Options().LogAttempts(false), func(t test.TestHelper) {
		t.T().Helper()

		o.withKubeconfig(t, func() {
			t.T().Helper()
			pod := podLocator(t, &o)
			phase := shell.Executef(t, `kubectl -n %s get pods %s -o jsonpath="{.status.phase}"`, pod.Namespace, pod.Name)
			if phase == "Running" {
				t.Logf("Pod %s/%s is running!", pod.Namespace, pod.Name)
			} else {
				t.Fatalf("Pod %s/%s is not running: %s", pod.Namespace, pod.Name, phase)
			}
		})
	})
}

func (o OC) WaitPodReadyWithOptions(t test.TestHelper, options retry.RetryOptions, podLocator PodLocatorFunc) {
	t.T().Helper()
	var pod NamespacedName
	retry.UntilSuccessWithOptions(t, options, func(t test.TestHelper) {
		t.T().Helper()
		pod = podLocator(t, &o)
		condition := o.Invokef(t, "kubectl -n %s wait --for condition=Ready pod %s --timeout 1s || true", pod.Namespace, pod.Name) // TODO: Change shell execute to do not fail on error
		if strings.Contains(condition, "condition met") {
			t.Logf("Pod %s in namespace %s is ready!", pod.Name, pod.Namespace)
		} else {
			t.Fatalf("error: %s in namespace %s is not ready: %s", pod.Name, pod.Namespace, condition)
		}
	})
}

func (o OC) WaitDeploymentRolloutComplete(t test.TestHelper, ns string, deploymentNames ...string) {
	t.T().Helper()
	timeout := 4 * time.Minute // TODO: make this configurable?
	start := time.Now()
	for _, name := range deploymentNames {
		usedUpTime := time.Now().Sub(start)
		remainingTime := timeout - usedUpTime
		o.Invokef(t, "kubectl -n %s rollout status deploy/%s --timeout=%s", ns, name, remainingTime.Round(time.Second))
	}
}

func (o OC) RestartAllPodsAndWaitReady(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	o.RestartAllPods(t, namespaces...)
	o.WaitAllPodsReady(t, namespaces...)
}

func (o OC) RestartAllPods(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	for _, ns := range namespaces {
		o.Invokef(t, "oc -n %s delete pod --all", ns)
	}
}

func (o OC) WaitPodsExist(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	for _, ns := range namespaces {
		retry.UntilSuccess(t, func(t test.TestHelper) {
			shell.Execute(t, fmt.Sprintf("oc get pods -n %s", ns), assert.OutputDoesNotContain(
				fmt.Sprintf("No resources found in %s namespace.", ns),
				fmt.Sprintf("Found pods in %s", ns),
				fmt.Sprintf("Did not find any pod in %s", ns),
			))
		})
	}
}

func (o OC) WaitAllPodsReady(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	for _, ns := range namespaces {
		o.Invokef(t, `oc -n %s wait --for condition=Ready --all pods --timeout 180s`, ns)
	}
}

func (o OC) DeletePodNoWait(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	pod := podLocator(t, &o)
	shell.Executef(t, `oc -n %s delete pod %s --wait=false`, pod.Namespace, pod.Name)
}

// WaitFor runs `oc wait` 30 times every 10 seconds. If the resource doesn't
// reach the specified condition in the last attempt, the function logs the failure
// ForCondition is the condition to wait for, e.g. "condition=Ready"
func (o OC) WaitFor(t test.TestHelper, ns string, kind string, name string, forCondition string) {
	t.T().Helper()
	maxAttempts := 30
	var attemptT *test.RetryTestHelper
	for i := 0; i < maxAttempts; i++ {
		t.Logf("Wait for condition %s on %s %s/%s...", forCondition, kind, ns, name)
		attemptT = retry.Attempt(t, func(t test.TestHelper) {
			t.T().Helper()
			shell.Execute(t,
				fmt.Sprintf(`oc wait %s %s/%s --for %s --timeout %s`, nsFlag(ns), kind, name, forCondition, "10s"),
				require.OutputContains("condition met",
					fmt.Sprintf("Condition %s met by %s %s/%s", forCondition, kind, ns, name),
					fmt.Sprintf("Condition %s not met by %s %s/%s", forCondition, kind, ns, name)))
		})
		if !attemptT.Failed() {
			attemptT.FlushLogBuffer()
			return
		}
	}

	// the last attempt has failed, so we print the buffered log statements and the output of `oc describe` to facilitate debugging
	attemptT.FlushLogBuffer()
	t.Logf("Running oc describe -n %s %s/%s\n%s", ns, kind, name, shell.Executef(t, `oc describe -n %s %s/%s`, ns, kind, name))
	t.FailNow()
}

func (o OC) WaitSMMRReady(t test.TestHelper, ns string) {
	t.T().Helper()
	t.Logf("Wait for smmr/default to be ready in namespace %s", ns)
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.Executef(t, `oc -n %s wait --for condition=Ready smmr/default --timeout 300s`, ns)
	})
}

func (o OC) GetAllResources(t test.TestHelper, ns string, checks ...common.CheckFunc) {
	t.T().Helper()
	shell.Execute(t,
		fmt.Sprintf(`oc get all -n %s`, ns),
		checks...)
}

func (o OC) DeletePod(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	var pod NamespacedName
	retry.UntilSuccess(t, func(t test.TestHelper) {
		pod = podLocator(t, &o)
	})
	retry.UntilSuccess(t, func(t test.TestHelper) {
		t.T().Helper()
		shell.Execute(t,
			fmt.Sprintf(`oc delete pod %s -n %s`, pod.Name, pod.Namespace),
			assert.OutputContains("deleted",
				fmt.Sprintf("Pod %s is being deleted", pod.Name),
				fmt.Sprintf("Pod %s deletion return an error", pod.Name)))
	})
}

func (o OC) WaitUntilResourceExist(t test.TestHelper, ns string, kind string, name string) {
	t.T().Helper()
	t.Logf("Wait until %s/%s in namespace %s exist", kind, name, ns)
	o.withKubeconfig(t, func() {
		t.T().Helper()
		retry.UntilSuccessWithOptions(t, retry.Options().DelayBetweenAttempts(5*time.Second), func(t test.TestHelper) {
			t.T().Helper()
			shell.Execute(t,
				fmt.Sprintf(`oc -n %s get %s/%s --ignore-not-found`, ns, kind, name),
				assert.OutputDoesNotContain(name,
					fmt.Sprintf("%s/%s was deleted", kind, name),
					fmt.Sprintf("%s/%s still exist", kind, name),
				))
		})
	})
}

func (o OC) GetPodAnnotations(t test.TestHelper, podLocator PodLocatorFunc) map[string]string {
	var data struct {
		Metadata struct {
			Annotations map[string]string `yaml:"annotations"`
		} `yaml:"metadata"`
	}

	po := podLocator(t, &o)
	yamlString := o.GetYaml(t, po.Namespace, "pod", po.Name)
	err := yaml.Unmarshal([]byte(yamlString), &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %s", err)
	}

	annotations := data.Metadata.Annotations
	return annotations
}
