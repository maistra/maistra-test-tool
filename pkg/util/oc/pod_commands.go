package oc

import (
	"fmt"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
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

func (o OC) WaitPodRunning(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	retry.UntilSuccessWithOptions(t, retry.Options().LogAttempts(false), func(t test.TestHelper) {
		t.T().Helper()
		pod := podLocator(t, &o)
		status := util.GetPodStatus(pod.Namespace, pod.Name)
		if status == "Running" {
			t.Logf("Pod %s/%s is running!", pod.Namespace, pod.Name)
		} else {
			t.Fatalf("Pod %s/%s is not running: %s", pod.Namespace, pod.Name, status)
		}
	})
}

func (o OC) WaitPodReady(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	var pod NamespacedName
	retry.UntilSuccess(t, func(t test.TestHelper) {
		pod = podLocator(t, &o)
	})
	condition := o.Invokef(t, "kubectl -n %s wait --for condition=Ready pod %s --timeout 30s || true", pod.Namespace, pod.Name) // TODO: Change shell execute to do not fail on error
	if strings.Contains(condition, "condition met") {
		t.Logf("Pod %s in namespace %s is ready!", pod.Name, pod.Namespace)
	} else {
		t.Fatalf("Error: %s in namespace %s is not ready: %s", pod.Name, pod.Namespace, condition)
	}
}

func (o OC) WaitDeploymentRolloutComplete(t test.TestHelper, ns string, deploymentNames ...string) {
	t.T().Helper()
	timeout := 3 * time.Minute // TODO: make this configurable?
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

func (o OC) WaitCondition(t test.TestHelper, ns string, kind string, name string, condition string) {
	t.T().Helper()
	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(30), func(t test.TestHelper) {
		shell.Execute(t,
			fmt.Sprintf(`oc wait -n %s %s/%s --for condition=%s  --timeout %s`, ns, kind, name, condition, "20s"),
			assert.OutputContains(condition,
				fmt.Sprintf("Condition %s met by %s %s/%s", condition, ns, kind, name),
				fmt.Sprintf("Condition %s not met %s %s/%s, retrying", condition, ns, kind, name)))
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
		shell.Execute(t,
			fmt.Sprintf(`oc delete pod %s -n %s`, pod.Name, pod.Namespace),
			assert.OutputContains("deleted",
				fmt.Sprintf("Pod %s is being deleted", pod.Name),
				fmt.Sprintf("Pod %s deletion return an error", pod.Name)))
	})
}
