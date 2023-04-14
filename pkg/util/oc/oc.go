package oc

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var DefaultOC = NewOC("")

func WithKubeconfig(location string) *OC {
	return NewOC(location)
}

func ApplyString(t test.TestHelper, ns string, yaml string) {
	t.T().Helper()
	DefaultOC.ApplyString(t, ns, yaml)
}

func ApplyTemplate(t test.TestHelper, ns string, template string, input interface{}) {
	t.T().Helper()
	DefaultOC.ApplyTemplateString(t, ns, template, input)
}

func DeleteFromTemplate(t test.TestHelper, ns string, yaml string, data interface{}) {
	t.T().Helper()
	DefaultOC.DeleteFromTemplate(t, ns, yaml, data)
}

func ApplyFile(t test.TestHelper, ns string, file string) {
	t.T().Helper()
	DefaultOC.ApplyFile(t, ns, file)
}

func DeleteFromString(t test.TestHelper, ns string, yaml string) {
	t.T().Helper()
	DefaultOC.DeleteFromString(t, ns, yaml)
}

func DeleteFile(t test.TestHelper, ns string, file string) {
	t.T().Helper()
	DefaultOC.DeleteFile(t, ns, file)
}

func CreateTLSSecret(t test.TestHelper, ns, name string, keyFile, certFile string) {
	t.T().Helper()
	DefaultOC.CreateTLSSecret(t, ns, name, keyFile, certFile)
}

func CreateGenericSecretFromFiles(t test.TestHelper, ns, name string, files ...string) {
	t.T().Helper()
	DefaultOC.CreateGenericSecretFromFiles(t, ns, name, files...)
}

func DeleteSecret(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	DefaultOC.DeleteSecret(t, ns, name)
}

func DeleteConfigMap(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	DefaultOC.DeleteConfigMap(t, ns, name)
}

func DeleteResource(t test.TestHelper, ns string, kind, name string) {
	t.T().Helper()
	DefaultOC.DeleteResource(t, ns, kind, name)
}

func DeleteNamespace(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	DefaultOC.DeleteNamespace(t, namespaces...)
}

func CreateNamespace(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	DefaultOC.CreateNamespace(t, namespaces...)
}

func RecreateNamespace(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	DefaultOC.RecreateNamespace(t, namespaces...)
}

func WaitSMCPReady(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	DefaultOC.WaitSMCPReady(t, ns, name)
}

func Patch(t test.TestHelper, ns, kind, name string, mergeType string, patch string) {
	t.T().Helper()
	DefaultOC.Patch(t, ns, kind, name, mergeType, patch)
}

func GetConfigMapData(t test.TestHelper, ns, name string) map[string]string {
	t.T().Helper()
	return DefaultOC.GetConfigMapData(t, ns, name)
}

func CreateConfigMapFromFiles(t test.TestHelper, ns, name string, files ...string) {
	t.T().Helper()
	DefaultOC.CreateConfigMapFromFiles(t, ns, name, files...)
}

func Exec(t test.TestHelper, podLocator PodLocatorFunc, container string, cmd string, checks ...common.CheckFunc) string {
	t.T().Helper()
	return DefaultOC.Exec(t, podLocator, container, cmd, checks...)
}

func GetPodIP(t test.TestHelper, podLocator PodLocatorFunc) string {
	t.T().Helper()
	return DefaultOC.GetPodIP(t, podLocator)
}

func Logs(t test.TestHelper, podLocator PodLocatorFunc, container string, checks ...common.CheckFunc) {
	t.T().Helper()
	DefaultOC.Logs(t, podLocator, container, checks...)
}

func WaitPodRunning(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	DefaultOC.WaitPodRunning(t, podLocator)
}

func WaitPodReady(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	DefaultOC.WaitPodReady(t, podLocator)
}

func WaitDeploymentRolloutComplete(t test.TestHelper, ns string, deploymentNames ...string) {
	t.T().Helper()
	DefaultOC.WaitDeploymentRolloutComplete(t, ns, deploymentNames...)
}

func RestartAllPodsAndWaitReady(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	DefaultOC.RestartAllPodsAndWaitReady(t, namespaces...)
}

func RestartAllPods(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	DefaultOC.RestartAllPods(t, namespaces...)
}

func WaitAllPodsReady(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	DefaultOC.WaitAllPodsReady(t, namespaces...)
}

func DeletePodNoWait(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	DefaultOC.DeletePodNoWait(t, podLocator)
}

func WaitCondition(t test.TestHelper, ns string, kind string, name string, condition string) {
	t.T().Helper()
	DefaultOC.WaitCondition(t, ns, kind, name, condition)
}

func WaitSMMRReady(t test.TestHelper, ns string) {
	t.T().Helper()
	shell.Executef(t, `oc -n %s wait --for condition=Ready smmr/default --timeout 300s`, ns)
}

func GetAllResources(t test.TestHelper, ns string, checks ...common.CheckFunc) {
	t.T().Helper()
	DefaultOC.GetAllResources(t, ns, checks...)
}

func DeletePod(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	DefaultOC.DeletePod(t, podLocator)
}

func ScaleDeploymentAndWait(t test.TestHelper, ns string, name string, replicas int) {
	t.T().Helper()
	shell.Executef(t, `oc -n %s scale deployment %s --replicas %d`, ns, name, replicas)
	WaitDeploymentRolloutComplete(t, ns, name)
}

// TouchSMCP causes the SMCP to be fully reconciled
func TouchSMCP(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	Patch(t, ns, "smcp", name, "merge", fmt.Sprintf(`{"spec":{"techPreview":{"foo":"foo%d"}}}`, rand.Int()))
}

func Label(t test.TestHelper, ns string, kind string, name string, labels string) {
	t.T().Helper()
	nsFlag := ""
	if ns != "" {
		nsFlag = "-n " + ns
	}
	shell.Executef(t, "oc %slabel %s %s %s", nsFlag, kind, name, labels)
}

func Get(t test.TestHelper, ns, kind, name string, checks ...common.CheckFunc) {
	t.T().Helper()
	shell.Execute(t, fmt.Sprintf("oc -n %s get %s/%s", ns, kind, name), checks...)
}

func GetYaml(t test.TestHelper, ns, kind, name string, checks ...common.CheckFunc) {
	t.T().Helper()
	shell.Execute(t, fmt.Sprintf("oc -n %s get %s/%s -oyaml", ns, kind, name), checks...)
}
