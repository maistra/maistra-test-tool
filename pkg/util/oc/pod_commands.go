package oc

import (
	"fmt"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type NamespacedName struct {
	Namespace string
	Name      string
}

type PodLocatorFunc func(t test.TestHelper) NamespacedName

func Exec(t test.TestHelper, podLocator PodLocatorFunc, container string, cmd string, checks ...assert.CheckFunc) {
	t.T().Helper()
	pod := podLocator(t)
	shell.Execute(t,
		fmt.Sprintf("kubectl exec -n %s %s -c %s -- %s", pod.Namespace, pod.Name, container, cmd),
		checks...)
}

func Logs(t test.TestHelper, podLocator PodLocatorFunc, container string, checks ...assert.CheckFunc) {
	t.T().Helper()
	pod := podLocator(t)
	shell.Execute(t,
		fmt.Sprintf("kubectl logs -n %s %s -c %s", pod.Namespace, pod.Name, container),
		checks...)
}

func WaitPodRunning(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	retry.UntilSuccess(t, func(t test.TestHelper) {
		t.T().Helper()
		pod := podLocator(t)
		status := util.GetPodStatus(pod.Namespace, pod.Name)
		if status == "Running" {
			t.Logf("Pod %s in namespace %s is running!", pod.Name, pod.Namespace)
		} else {
			t.Fatalf("%s in namespace %s is not running: %s", pod.Name, pod.Namespace, status)
		}
	})
}

func WaitPodReady(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	var pod NamespacedName
	retry.UntilSuccess(t, func(t test.TestHelper) {
		pod = podLocator(t)
	})

	shell.Executef(t, "kubectl -n %s wait --for condition=Ready pod %s --timeout 30s", pod.Namespace, pod.Name)
}

func WaitDeploymentRolloutComplete(t test.TestHelper, ns string, deploymentNames ...string) {
	t.T().Helper()
	timeout := 1 * time.Minute // TODO: make this configurable?
	start := time.Now()
	for _, name := range deploymentNames {
		usedUpTime := time.Now().Sub(start)
		remainingTime := timeout - usedUpTime
		shell.Executef(t, "kubectl -n %s rollout status deploy/%s --timeout=%s", ns, name, remainingTime.Round(time.Second))
	}
}

func RestartAllPodsAndWaitReady(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	for _, ns := range namespaces {
		shell.Executef(t, "oc -n %s delete pod --all", ns)
	}
	for _, ns := range namespaces {
		WaitAllPodsReady(t, ns)
	}
}

func WaitAllPodsReady(t test.TestHelper, ns string) {
	t.T().Helper()
	shell.Executef(t, `oc -n %s wait --for condition=Ready --all pods --timeout 180s`, ns)
}
