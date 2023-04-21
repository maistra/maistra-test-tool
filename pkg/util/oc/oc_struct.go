package oc

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
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
	o.withKubeconfig(t, func() {
		t.T().Helper()
		o.ApplyString(t, ns, template.Run(t, tmpl, input))
	})
}

func (o OC) ReplaceOrApplyString(t test.TestHelper, ns string, yaml string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		output := shell.ExecuteWithInput(t, fmt.Sprintf("oc %s replace -f - || echo 'error captured'", nsFlag(ns)), yaml)
		if strings.Contains(output, "NotFound") {
			shell.ExecuteWithInput(t, fmt.Sprintf("oc %s apply -f -", nsFlag(ns)), yaml)
		}
	})
}

func (o OC) ReplaceOrApplyTemplate(t test.TestHelper, ns string, tmpl string, input interface{}) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		o.ReplaceOrApplyString(t, ns, template.Run(t, tmpl, input))
	})
}

func (o OC) DeleteFromTemplate(t test.TestHelper, ns string, tmpl string, input interface{}) {
	t.T().Helper()
	o.DeleteFromString(t, ns, template.Run(t, tmpl, input))
}

func (o OC) ApplyTemplateFile(t test.TestHelper, ns string, tmplFile string, input interface{}) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		templateString, err := os.ReadFile(tmplFile)
		if err != nil {
			t.Fatalf("could not read template file %s: %v", tmplFile, err)
		}
		o.ApplyTemplateString(t, ns, string(templateString), input)
	})
}

func (o OC) ApplyString(t test.TestHelper, ns string, yamls ...string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.ExecuteWithInput(t, fmt.Sprintf("oc %s apply -f -", nsFlag(ns)), concatenateYamls(yamls...))
	})
}

func (o OC) ApplyFile(t test.TestHelper, ns string, file string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		o.Invokef(t, "oc %s apply -f %s", nsFlag(ns), file)
	})
}

func (o OC) DeleteFromString(t test.TestHelper, ns string, yamls ...string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.ExecuteWithInput(t, fmt.Sprintf("oc %s delete -f - --ignore-not-found", nsFlag(ns)), concatenateYamls(yamls...))
	})
}

func (o OC) DeleteFile(t test.TestHelper, ns string, file string) {
	t.T().Helper()
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
		o.DeleteResource(t, ns, k, name)
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

// DeleteNamespace deletes the specified namespaces and waits until they are fully deleted
func (o OC) DeleteNamespace(t test.TestHelper, namespaces ...string) {
	t.T().Helper()
	o.withKubeconfig(t, func() {
		t.T().Helper()
		t.Logf("Deleting namespaces: %v", namespaces)
		o.Invokef(t, "kubectl delete ns %s", strings.Join(namespaces, " "))
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
  name: %s
---`, ns)
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
		t.Logf("Wait for SMCP %s/%s to be ready", ns, name)
		o.WaitCondition(t, ns, "smcp", name, "Ready")
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

func (o OC) Label(t test.TestHelper, ns string, kind string, name string, labels string) {
	t.T().Helper()
	nsFlag := ""
	if ns != "" {
		nsFlag = "-n " + ns
	}
	o.withKubeconfig(t, func() {
		t.T().Helper()
		shell.Executef(t, "oc %slabel %s %s %s", nsFlag, kind, name, labels)
	})
}

func (o OC) Get(t test.TestHelper, ns, kind, name string, checks ...common.CheckFunc) string {
	t.T().Helper()
	var val string
	o.withKubeconfig(t, func() {
		t.T().Helper()
		val = shell.Execute(t, fmt.Sprintf("oc %s get %s/%s", nsFlag(ns), kind, name), checks...)
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

// GetProxy returns the Proxy object from the cluster
func (o OC) GetProxy(t test.TestHelper) Proxy {
	type ProxyResource struct {
		Status Proxy `json:"status"`
	}

	proxyResource := ProxyResource{}

	proxyYaml := o.GetYaml(t, "", "proxy", "cluster")
	err := yaml.Unmarshal([]byte(proxyYaml), &proxyResource)
	if err != nil {
		t.Fatalf("Could not parse Proxy resource: %v", err)
	}

	proxy := proxyResource.Status
	if proxy.HTTPProxy != "" {
		t.Logf("HTTP_PROXY: %q", proxy.HTTPProxy)
	}
	if proxy.HTTPSProxy != "" {
		t.Logf("HTTPS_PROXY: %q", proxy.HTTPSProxy)
	}
	if proxy.NoProxy != "" {
		t.Logf("NO_PROXY: %q", proxy.NoProxy)
	}

	return proxy
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
