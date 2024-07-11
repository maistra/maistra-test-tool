package oc

import (
	"fmt"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var DefaultOC = NewOC("")

func WithKubeconfig(location string) *OC {
	return NewOC(location)
}

func ApplyString(t test.TestHelper, ns string, yamls ...string) {
	t.T().Helper()
	DefaultOC.ApplyString(t, ns, yamls...)
}

func GetOCPVersion(t test.TestHelper) string {
	t.T().Helper()
	return DefaultOC.GetOCPVersion(t)
}

func ReplaceOrApplyString(t test.TestHelper, ns string, yaml string) {
	t.T().Helper()
	DefaultOC.ReplaceOrApplyString(t, ns, yaml)
}

func ApplyTemplate(t test.TestHelper, ns string, template string, input interface{}) {
	t.T().Helper()
	DefaultOC.ApplyTemplateString(t, ns, template, input)
}

func ReplaceOrApplyTemplate(t test.TestHelper, ns string, template string, input interface{}) {
	t.T().Helper()
	DefaultOC.ReplaceOrApplyTemplate(t, ns, template, input)
}

func DeleteFromTemplate(t test.TestHelper, ns string, yaml string, data interface{}) {
	t.T().Helper()
	DefaultOC.DeleteFromTemplate(t, ns, yaml, data)
}

func ApplyFile(t test.TestHelper, ns string, file string) {
	t.T().Helper()
	DefaultOC.ApplyFile(t, ns, file)
}

func DeleteFromString(t test.TestHelper, ns string, yamls ...string) {
	t.T().Helper()
	DefaultOC.DeleteFromString(t, ns, yamls...)
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

func DeleteSecret(t test.TestHelper, ns string, name ...string) {
	t.T().Helper()
	DefaultOC.DeleteSecret(t, ns, name...)
}

func DeleteConfigMap(t test.TestHelper, ns string, name ...string) {
	t.T().Helper()
	DefaultOC.DeleteConfigMap(t, ns, name...)
}

func DeleteResource(t test.TestHelper, ns string, kind string, name ...string) {
	t.T().Helper()
	DefaultOC.DeleteResource(t, ns, kind, name...)
}

func GetResouceNameByLabel(t test.TestHelper, ns string, kind string, label string) string {
	t.T().Helper()
	return DefaultOC.GetResouceNameByLabel(t, ns, kind, label)
}

func ResourceByLabelExists(t test.TestHelper, ns string, kind string, label string) bool {
	t.T().Helper()
	return DefaultOC.ResourceByLabelExists(t, ns, kind, label)
}

func AnyResourceExist(t test.TestHelper, ns string, kind string) bool {
	t.T().Helper()
	return DefaultOC.AnyResourceExist(t, ns, kind)
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

func GetServiceClusterIP(t test.TestHelper, ns, serviceName string) string {
	t.T().Helper()
	return DefaultOC.GetServiceClusterIP(t, ns, serviceName)
}

func Logs(t test.TestHelper, podLocator PodLocatorFunc, container string, checks ...common.CheckFunc) {
	t.T().Helper()
	DefaultOC.Logs(t, podLocator, container, checks...)
}

func LogsSince(t test.TestHelper, start time.Time, podLocator PodLocatorFunc, container string, checks ...common.CheckFunc) {
	t.T().Helper()
	DefaultOC.LogsSince(t, start, podLocator, container, checks...)
}

func LogsFromPods(t test.TestHelper, ns, selector string, checks ...common.CheckFunc) {
	t.T().Helper()
	DefaultOC.LogsFromPods(t, ns, selector, checks...)
}

func WaitPodRunning(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	DefaultOC.WaitPodRunning(t, podLocator)
}

func WaitPodReady(t test.TestHelper, podLocator PodLocatorFunc) {
	t.T().Helper()
	DefaultOC.WaitPodReadyWithOptions(t, retry.Options(), podLocator)
}

func WaitPodReadyWithOptions(t test.TestHelper, retry retry.RetryOptions, podLocator PodLocatorFunc) {
	t.T().Helper()
	DefaultOC.WaitPodReadyWithOptions(t, retry, podLocator)
}

func UndoRollout(t test.TestHelper, ns string, kind, name string) {
	t.T().Helper()
	DefaultOC.UndoRollout(t, ns, kind, name)
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

func WaitPodsExist(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	DefaultOC.WaitPodsExist(t, namespaces...)
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
	DefaultOC.WaitFor(t, ns, kind, name, fmt.Sprintf("condition=%s", condition))
}

func WaitForPhase(t test.TestHelper, ns string, kind string, name string, phase string) {
	t.T().Helper()
	DefaultOC.WaitFor(t, ns, kind, name, fmt.Sprintf("jsonpath='{.status.phase}'=%s", phase))
}

func WaitSMMRReady(t test.TestHelper, ns string) {
	t.T().Helper()
	DefaultOC.WaitSMMRReady(t, ns)
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
	DefaultOC.ScaleDeploymentAndWait(t, ns, name, replicas)
}

// TouchSMCP causes the SMCP to be fully reconciled
func TouchSMCP(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	DefaultOC.TouchSMCP(t, ns, name)
}

func RemoveLabel(t test.TestHelper, ns string, kind string, name string, label string) {
	t.T().Helper()
	DefaultOC.RemoveLabel(t, ns, kind, name, label)
}

func Label(t test.TestHelper, ns string, kind string, name string, labels string) {
	t.T().Helper()
	DefaultOC.Label(t, ns, kind, name, labels)
}

func TaintNode(t test.TestHelper, name string, taints ...string) {
	t.T().Helper()
	DefaultOC.TaintNode(t, name, taints...)
}

func Get(t test.TestHelper, ns, kind, name string, checks ...common.CheckFunc) string {
	t.T().Helper()
	return DefaultOC.Get(t, ns, kind, name, checks...)
}

func GetYaml(t test.TestHelper, ns, kind, name string, checks ...common.CheckFunc) string {
	t.T().Helper()
	return DefaultOC.GetYaml(t, ns, kind, name, checks...)
}

// GetJson returns the JSON representation of the resource, you can set a jsonPath to extract a specific value or send "" to get the full JSON
func GetJson(t test.TestHelper, ns, kind, name string, jsonPath string, checks ...common.CheckFunc) string {
	t.T().Helper()
	return DefaultOC.GetJson(t, ns, kind, name, jsonPath, checks...)
}

func GetProxy(t test.TestHelper) *Proxy {
	t.T().Helper()
	return DefaultOC.GetProxy(t)
}

func WaitUntilResourceExist(t test.TestHelper, ns string, kind string, name string) {
	t.T().Helper()
	DefaultOC.WaitUntilResourceExist(t, ns, kind, name)
}
