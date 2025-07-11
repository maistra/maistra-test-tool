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
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type OC struct {
	kubeconfig string
}

func NewOC(kubeconfig string) *OC {
	return &OC{kubeconfig: kubeconfig}
}

func (o OC) ApplyTemplateString(t test.TestHelper, ns string, tmpl string, input interface{}) {
	t.T().Helper()
	o.retryFunction(t, func() {
		t.T().Helper()
		o.ApplyString(t, ns, template.Run(t, tmpl, input))
	})
}

func (o OC) GetOCPVersion(t test.TestHelper) string {
	t.T().Helper()
	output := ""
	o.withKubeconfig(t, func() {
		t.T().Helper()
		output = shell.Execute(t, "oc version")
		// The output have this format:
		// 	Client Version: 4.12.0-rc.5
		// Kustomize Version: v4.5.7
		// Server Version: 4.10.59
		// Kubernetes Version: v1.23.17+16bcd69
	})

	// We want to split only the line with "Server Version"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Server Version") {
			// Get the version number
			version := strings.Split(line, ":")[1]
			version = strings.TrimSpace(version)
			return version
		}
	}
	// We never reach this point so if this happens we will return an error
	t.Fatal("Unable to get OCP version")
	return ""
}

func (o OC) DeleteFromTemplate(t test.TestHelper, ns string, tmpl string, input interface{}) {
	t.T().Helper()
	o.DeleteFromString(t, ns, template.Run(t, tmpl, input))
}

func (o OC) ApplyTemplateFile(t test.TestHelper, ns string, tmplFile string, input interface{}) {
	t.T().Helper()
	o.retryFunction(t, func() {
		t.T().Helper()
		templateString, err := os.ReadFile(tmplFile)
		if err != nil {
			t.Fatalf("could not read template file %s: %v", tmplFile, err)
		}
		o.ApplyTemplateString(t, ns, string(templateString), input)
	})
}

// retryFunction retries the specified function if it fails.
func (o OC) retryFunction(t test.TestHelper, f func()) {
	t.T().Helper()
	maxAttempts := 5
	var attemptT *test.RetryTestHelper
	warning := false
	for i := 0; i < maxAttempts; i++ {
		attemptT = retry.Attempt(t, func(t test.TestHelper) {
			t.T().Helper()
			o.withKubeconfig(t, f)
		})
		if !attemptT.Failed() {
			if warning {
				t.Logf("WARNING: attempt %d of %d succeeded after previous failures.", i+1, maxAttempts)
			}
			attemptT.FlushLogBuffer()
			return
		}
		// Wait for 1 second before retrying
		warning = true
		time.Sleep(1 * time.Second)
	}

	// the last attempt has failed, so we print the buffered log statements
	attemptT.FlushLogBuffer()
	t.Fatalf("Command failed after %d attempts.", maxAttempts)
}

// ApplyString applies the specified YAMLs using oc apply and retries if the command fails.
func (o OC) ApplyString(t test.TestHelper, ns string, yamls ...string) {
	t.T().Helper()
	o.retryFunction(t, func() {
		t.T().Helper()
		shell.ExecuteWithInput(t, fmt.Sprintf("oc %s apply -f -", nsFlag(ns)), concatenateYamls(yamls...))
	})
}

// ApplyFile applies the specified file using oc apply and retries if the command fails.
func (o OC) ApplyFile(t test.TestHelper, ns string, file string) {
	t.T().Helper()
	o.retryFunction(t, func() {
		t.T().Helper()
		o.Invokef(t, "oc %s apply -f %s", nsFlag(ns), file)
	})
}

func (o OC) DeleteFromString(t test.TestHelper, ns string, yamls ...string) {
	t.T().Helper()
	t.Logf("Deleting resources from namespace %s", ns)
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.ExecuteWithInput(t, fmt.Sprintf("oc %s delete -f - --ignore-not-found", nsFlag(ns)), concatenateYamls(yamls...))
	})
}

func (o OC) DeleteFile(t test.TestHelper, ns string, file string) {
	t.T().Helper()
	t.Logf("Deleting file %s from namespace %s", file, ns)
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.Executef(t, "kubectl delete %s -f %s --ignore-not-found", nsFlag(ns), file)
	})
}

func concatenateYamls(yamls ...string) string {
	return strings.Join(yamls, "\n---\n")
}

func nsFlag(ns string) string {
	if ns == "" {
		return ""
	}
	return "-n " + ns
}

func (o OC) CreateGenericSecretFromFiles(t test.TestHelper, ns, name string, files ...string) {
	t.T().Helper()
	o.createSecretOrConfigMapFromFiles(t, ns, "secret generic", name, files...)
}

func (o OC) CreateConfigMapFromFiles(t test.TestHelper, ns, name string, files ...string) {
	t.T().Helper()
	o.createSecretOrConfigMapFromFiles(t, ns, "configmap", name, files...)
}

func (o OC) createSecretOrConfigMapFromFiles(t test.TestHelper, ns string, kind string, name string, files ...string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		k := kind
		if kind == "secret generic" {
			k = "secret"
		}
		retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(5).LogAttempts(false), func(t test.TestHelper) {
			t.T().Helper()
			o.DeleteResource(t, ns, k, name)
		})
		cmd := fmt.Sprintf(`oc create %s %s -n %s `, kind, name, ns)
		for _, file := range files {
			cmd += fmt.Sprintf(" --from-file=%s", file)
		}
		shell.Execute(t, cmd)
	})
}

func (o OC) CreateTLSSecret(t test.TestHelper, ns, name string, keyFile, certFile string) {
	t.T().Helper()
	o.DeleteSecret(t, ns, name)
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.Executef(t, "oc %s create secret tls %s --key %s --cert %s", nsFlag(ns), name, keyFile, certFile)
	})
}

func (o OC) DeleteSecret(t test.TestHelper, ns string, name ...string) {
	t.T().Helper()
	o.DeleteResource(t, ns, "secret", name...)
}

func (o OC) DeleteConfigMap(t test.TestHelper, ns string, name ...string) {
	t.T().Helper()
	o.DeleteResource(t, ns, "configmap", name...)
}

func (o OC) DeleteResource(t test.TestHelper, ns string, kind string, names ...string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.Executef(t, "kubectl %s delete %s %s --ignore-not-found", nsFlag(ns), kind, strings.Join(names, " "))
	})
}

func (o OC) DeleteResourcesByLabel(t test.TestHelper, ns string, kind string, label string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		var nsMsg string
		if ns != "" {
			nsMsg = " in namespace " + ns
		}
		t.Logf("Deleting %s resources matching selector %s%s", kind, label, nsMsg)
		shell.Executef(t, "kubectl %s delete %s -l %s", nsFlag(ns), kind, label)
	})
}

// DeleteNamespace deletes the specified namespaces and waits until they are fully deleted
func (o OC) DeleteNamespace(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		t.Logf("Deleting namespaces: %v", namespaces)
		o.Invokef(t, "kubectl delete ns --ignore-not-found %s", strings.Join(namespaces, " "))
	})
}

// DeleteTestBoundNamespaces deletes namespaces with the maistra test label that should be
// deleted/recreated between each test.
func (o OC) DeleteTestBoundNamespaces(t test.TestHelper) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		t.Logf("Deleting namespaces matching selector: %v", testBoundNamespacesSelector)
		o.Invokef(t, "kubectl delete ns -l %s", testBoundNamespacesSelector)
	})
}

func (o OC) CreateNamespace(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		t.Logf("Creating namespaces: %v", namespaces)

		yaml := ""
		for _, ns := range namespaces {
			yaml += fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  labels:
    %s: %s
  name: %s
---`, MaistraTestLabel, testBoundNSLabelValue, ns)
		}
		o.ApplyString(t, "", yaml)
	})
}

func (o OC) RecreateNamespace(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		o.DeleteNamespace(t, namespaces...)
		o.CreateNamespace(t, namespaces...)
	})
}

func (o OC) WaitSMCPReady(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		o.WaitFor(t, ns, "smcp", name, "condition=Ready")
	})
}

func (o OC) WaitKialiReady(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		o.WaitFor(t, ns, "Kiali", name, "condition=Successful")
	})
}

func (o OC) Patch(t test.TestHelper, ns, kind, name string, mergeType string, patch string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		// quote the patch using single quotes, while escaping existing single quotes in the string
		// for example: "foo'bar" becomes "'foo'\''bar'"
		quotedPatch := fmt.Sprintf("'%s'", strings.ReplaceAll(patch, `'`, `'\\''`))
		o.Invokef(t, `oc -n %s patch %s/%s --type %s -p %s`, ns, kind, name, mergeType, quotedPatch)
	})
}

func (o OC) GetConfigMapData(t test.TestHelper, ns, name string) map[string]string {
	data := make(map[string]string)
	o.withKubeconfig(t, func() {
		t.T().Helper()
		manifest := o.Invokef(t, "oc get configmap -n %s %s -o json", ns, name)
		m := map[string]interface{}{}
		err := json.Unmarshal([]byte(manifest), &m)
		if err != nil {
			t.Fatalf("could not unmarshal ConfigMap JSON: %v", err)
		}

		if dataMap, ok := m["data"].(map[string]interface{}); ok {
			for k, v := range dataMap {
				if str, ok := v.(string); ok {
					data[k] = str
				} else {
					t.Fatalf("shouldn't happen")
				}
			}
		} else {
			t.Fatalf("could not get .data from ConfigMap JSON: %v", err)
		}
	})
	return data
}

func (o OC) ScaleDeploymentAndWait(t test.TestHelper, ns string, deployment string, replicas int) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		o.Invokef(t, "oc -n %s scale deploy/%s --replicas %d", ns, deployment, replicas)
		WaitDeploymentRolloutComplete(t, ns, deployment)
	})
}

// TouchSMCP causes the SMCP to be fully reconciled
func (o OC) TouchSMCP(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	o.Patch(t, ns, "smcp", name, "merge", fmt.Sprintf(`{"spec":{"techPreview":{"foo":"foo%d"}}}`, rand.Int()))
}

func (o OC) ExposeSvc(t test.TestHelper, ns string, svcName string, servicePort string, routeName string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		t.Logf("Exposing svc: %v", svcName)
		o.Invokef(t, "oc -n %s expose svc %s --port=%s --name=%s", ns, svcName, servicePort, routeName)
	})
}

func (o OC) GetRouteURL(t test.TestHelper, ns string, name string) string {
	t.T().Helper()
	url := ""
	o.withKubeconfig(t, func() {
		t.T().Helper()
		url = o.Invokef(t, "oc -n %s get route %s -o jsonpath='{.spec.host}'", ns, name)
	})
	return url
}

func (o OC) Invokef(t test.TestHelper, format string, a ...any) string {
	t.T().Helper()
	var output string
	o.withKubeconfig(t, func() {
		t.T().Helper()
		output = o.Invoke(t, fmt.Sprintf(format, a...))
	})
	return output
}

func (o OC) Invoke(t test.TestHelper, command string, checks ...common.CheckFunc) string {
	t.T().Helper()
	var output string
	o.withKubeconfig(t, func() {
		t.T().Helper()
		output = shell.Execute(t, command, checks...)
	})
	return output
}

// This is a hack and will prevent us from running tests in parallel.
//
// A different approach would be to set the environment variable only for the
// command we're executing (see shell.ExecuteWithEnv), but this is error-prone
// because it's not easy to ensure that all helper functions will use it. Or,
// perhaps, TestHelper should have a SetEnv() function and then the shell.Execute
// function should check the env vars added to TestHelper and execute the
// command with those variables. But to use this approach, we must first
// refactor all the tests and ensure that only one function in the entire codebase
// is used to execute commands (TODO)
func (o OC) withKubeconfig(t test.TestHelper, f func()) {
	t.T().Helper()
	if o.kubeconfig == "" {
		f()
	} else {
		oldValue := env.GetKubeconfig()
		setEnv(t, "KUBECONFIG", o.kubeconfig)
		f()
		setEnv(t, "KUBECONFIG", oldValue)
	}
}

func (o OC) UndoRollout(t test.TestHelper, ns string, kind, name string) {
	shell.Executef(t, `kubectl -n %s rollout undo %s %s`, ns, kind, name)
}

func (o OC) TaintNode(t test.TestHelper, name string, taints ...string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.Executef(t, `oc adm taint nodes %s %s`, name, strings.Join(taints, " "))
	})
}

func (o OC) RemoveLabel(t test.TestHelper, ns string, kind string, name string, label string) {
	o.Label(t, ns, kind, name, label+"-")
}

func (o OC) Label(t test.TestHelper, ns string, kind string, name string, labels string) {
	t.T().Helper()
	nsFlag := ""
	if ns != "" {
		nsFlag = fmt.Sprintf("-n %s ", ns)
	}
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.Executef(t, "oc %slabel %s %s %s", nsFlag, kind, name, labels)
	})
}

func (o OC) Get(t test.TestHelper, ns, kind, name string, checks ...common.CheckFunc) string {
	t.T().Helper()
	var val string
	element := fmt.Sprintf("%s/%s", kind, name)
	if name == "" {
		element = kind
	}
	o.withKubeconfig(t, func() {
		t.T().Helper()
		val = shell.Execute(t, fmt.Sprintf("oc %s get %s", nsFlag(ns), element), checks...)
	})
	return val
}

func (o OC) GetYaml(t test.TestHelper, ns, kind, name string, checks ...common.CheckFunc) string {
	t.T().Helper()
	var val string
	o.withKubeconfig(t, func() {
		t.T().Helper()
		val = shell.Execute(t, fmt.Sprintf("oc %s get %s/%s -oyaml", nsFlag(ns), kind, name), checks...)
	})
	return val
}

func (o OC) GetJson(t test.TestHelper, ns, kind, name, jsonPath string, checks ...common.CheckFunc) string {
	t.T().Helper()
	var jsonString string
	o.withKubeconfig(t, func() {
		t.T().Helper()
		if jsonPath == "" {
			jsonString = shell.Execute(t, fmt.Sprintf(`oc %s get %s %s -o json`, nsFlag(ns), kind, name), checks...)
		} else {
			jsonPath = strings.ReplaceAll(jsonPath, "'", `'"'"'`)
			jsonString = shell.Execute(t, fmt.Sprintf(`oc %s get %s %s -o jsonpath='%s'`, nsFlag(ns), kind, name, jsonPath), checks...)
		}
	})
	return jsonString
}

// GetProxy returns the Proxy object from the cluster
func (o OC) GetProxy(t test.TestHelper) *Proxy {
	proxyJson := o.GetJson(t, "", "proxy", "cluster", "{.status}")

	proxy := &Proxy{}
	if proxyJson == "" {
		return proxy
	}
	var data map[string]interface{}
	err := json.Unmarshal([]byte(proxyJson), &data)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %s\n", err)
	}
	if data["httpProxy"] != nil {
		proxy.HTTPProxy = data["httpProxy"].(string)
	}
	if data["httpsProxy"] != nil {
		proxy.HTTPSProxy = data["httpsProxy"].(string)
	}
	if data["noProxy"] != nil {
		proxy.NoProxy = data["noProxy"].(string)
	}
	return proxy
}

func (o OC) ResourceExists(t test.TestHelper, ns, kind, name string) bool {
	t.T().Helper()
	var exists bool
	o.withKubeconfig(t, func() {
		t.T().Helper()
		output := shell.Execute(t, fmt.Sprintf("oc %s get %s/%s || true", nsFlag(ns), kind, name))
		exists = !(strings.Contains(output, "Error from server (NotFound)") || strings.Contains(output, "No resources found"))
	})
	return exists
}

// Function returns names of all resources (kind input) in the namespace (ns input) that match a particular label (label input).
// Label input can be an empty string, and then all resources in the namespace are returned
// When you are looking for a global scoped resource (e.g. nodes), ns can be empty
func (o OC) GetAllResourcesNames(t test.TestHelper, ns, kind, label string) []string {
	t.T().Helper()
	var values []string
	o.withKubeconfig(t, func() {
		t.T().Helper()
		output := shell.Execute(t, fmt.Sprintf("oc %s get %s -l '%s' -o jsonpath='{.items[*].metadata.name}' || true", nsFlag(ns), kind, label))
		if output == "" {
			t.Fatalf("Could not find resource %s with label %s in namespace %s", kind, label, ns)
		}
		values = strings.Split(output, " ")
	})
	return values
}

func (o OC) ResourceByLabelExists(t test.TestHelper, ns, kind, label string) bool {
	t.T().Helper()
	var exists bool
	o.withKubeconfig(t, func() {
		t.T().Helper()
		output := shell.Execute(t, fmt.Sprintf("oc %s get %s -l %s || true", nsFlag(ns), kind, label))
		exists = !(strings.Contains(output, "Error from server (NotFound)") || strings.Contains(output, "No resources found"))
	})
	return exists
}

func (o OC) AnyResourceExist(t test.TestHelper, ns string, kind string) bool {
	t.T().Helper()
	var exists bool
	o.withKubeconfig(t, func() {
		t.T().Helper()
		output := shell.Execute(t, fmt.Sprintf("oc %s get %s || true", nsFlag(ns), kind))
		exists = !(strings.Contains(output, "Error from server (NotFound)") || strings.Contains(output, "No resources found"))
	})
	return exists
}

func setEnv(t test.TestHelper, key string, value string) {
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("could not set %s: %v", key, err)
	}
}

type Proxy struct {
	HTTPProxy  string `json:"httpProxy"`
	HTTPSProxy string `json:"httpsProxy"`
	NoProxy    string `json:"noProxy"`
}
