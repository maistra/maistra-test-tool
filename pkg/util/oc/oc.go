package oc

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func ApplyString(t test.TestHelper, ns string, yaml string) {
	t.T().Helper()
	if err := util.KubeApplyContents(ns, yaml); err != nil {
		t.Fatalf("Failed to apply manifest: %v;\nYAML: %v", err, yaml)
	}
}

func ApplyTemplate(t test.TestHelper, ns string, yaml string, data interface{}) {
	t.T().Helper()
	template := template.Run(t, yaml, data)
	ApplyString(t, ns, template)
}

func DeleteFromTemplate(t test.TestHelper, ns string, yaml string, data interface{}) {
	t.T().Helper()
	template := template.Run(t, yaml, data)
	DeleteFromString(t, ns, template)
}

func ApplyFile(t test.TestHelper, ns string, file string) {
	t.T().Helper()
	if err := util.KubeApply(ns, file); err != nil {
		t.Fatalf("Failed to apply manifest file %s: %v", file, err)
	}
}

func DeleteFromString(t test.TestHelper, ns string, yaml string) {
	t.T().Helper()
	if err := util.KubeDeleteContents(ns, yaml); err != nil {
		t.Fatalf("Failed to delete objects in YAML: %v; YAML: %v", err, yaml)
	}
}

func DeleteFile(t test.TestHelper, ns string, file string) {
	t.T().Helper()
	if err := util.KubeDelete(ns, file); err != nil {
		t.Fatalf("Failed to delete objects from file %s: %v", file, err)
	}
}

func CreateTLSSecret(t test.TestHelper, ns, name string, keyFile, certFile string) {
	DeleteSecret(t, ns, name)
	if _, err := util.CreateTLSSecret(name, ns, keyFile, certFile); err != nil {
		t.Fatalf("Failed to create secret %s\n", name)
	}
}

func CreateTLSSecretWithCACert(t test.TestHelper, ns, name string, keyFile, certFile, caCertFile string) {
	t.T().Helper()
	DeleteSecret(t, ns, name)
	shell.Executef(t,
		`kubectl create -n %s secret generic %s --from-file=tls.key=%s --from-file=tls.crt=%s --from-file=ca.crt=%s`,
		ns, name, keyFile, certFile, caCertFile)
}

func DeleteSecret(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	shell.ExecuteIgnoreError(t, fmt.Sprintf(`kubectl -n %s delete secret %s`, ns, name))
}

func DeleteNamespace(t test.TestHelper, ns string) {
	t.T().Helper()
	t.Logf("Deleting namespace %q", ns)
	shell.Executef(t, "kubectl delete ns %s", ns)
}

func CreateNamespace(t test.TestHelper, ns string) {
	t.T().Helper()
	t.Logf("Creating namespace %q", ns)
	shell.Executef(t, "oc new-project %s", ns)
}

func RecreateNamespace(t test.TestHelper, ns string) {
	t.T().Helper()
	DeleteNamespace(t, ns)
	CreateNamespace(t, ns)
}

func WaitSMCPReady(t test.TestHelper, ns string, name string) {
	t.T().Helper()
	shell.Executef(t, `oc -n %s wait --for condition=Ready smcp/%s --timeout 300s`, ns, name)
}

func GetAllResources(t test.TestHelper, ns string, checks ...common.CheckFunc) {
	t.T().Helper()
	shell.Execute(t,
		fmt.Sprintf(`oc get all -n %s`, ns),
		checks...)
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
