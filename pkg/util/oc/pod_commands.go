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

func MergePatch(t test.TestHelper, ns string, rs string, ptype string, patch string, checks ...assert.CheckFunc) {
	t.T().Helper()
	shell.Execute(t,
		fmt.Sprintf(`oc patch -n %s %s --type %s -p '%s'`, ns, rs, ptype, patch),
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

func WaitCondition(t test.TestHelper, ns string, kind string, name string, condition string) {
	t.T().Helper()
	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(30), func(t test.TestHelper) {
		shell.Executef(t,
			fmt.Sprintf(`oc wait -n %s %s/%s --for condition=%s  --timeout %s`, ns, kind, name, condition, "10s"),
			assert.OutputContains(condition,
				fmt.Sprintf("Condition %s met by %s %s/%s", condition, kind, ns, name),
				fmt.Sprintf("Condition %s not met %s %s/%s, retrying", condition, kind, ns, name)))
	})
}

func DeletePod(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	var pod NamespacedName
	retry.UntilSuccess(t, func(t test.TestHelper) {
		pod = podLocator(t)
	})
	shell.Executef(t,
		fmt.Sprintf(`oc delete pod %s -n %s `, pod.Name, pod.Namespace),
		assert.OutputContains("deleted",
			fmt.Sprintf("Pod %s is being deleted", pod.Name),
			fmt.Sprintf("Pod %s deletion return an error", pod.Name)))
	retry.UntilSuccess(t, func(t test.TestHelper) {
		shell.Executef(t,
			fmt.Sprintf(`oc get pod %s -n %s || true`, pod.Name, pod.Namespace), // TODO: modify the shell package to support this without failing when the pod does not exist
			assert.OutputContains("NotFound",
				fmt.Sprintf("Pod %s is deleted", pod.Name),
				fmt.Sprintf("Pod %s is not deleted yet", pod.Name)))
	})
}
