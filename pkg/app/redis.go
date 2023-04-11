package app

import (
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type redis struct {
	ns string
}

var _ App = &bookinfo{}

func Redis(ns string) App {
	return &redis{ns: ns}
}

func (a *redis) Name() string {
	return "redis"
}

func (a *redis) Namespace() string {
	return a.ns
}

func (a *redis) Install(t test.TestHelper) {
	t.T().Helper()
	oc.CreateNamespace(t, a.ns)
	t.Log("Deploy Redis in namespace %q", a.ns)
	oc.ApplyFile(t, a.ns, examples.RedisYamlFile())
}

func (a *redis) Uninstall(t test.TestHelper) {
	t.T().Helper()
	t.Logf("Uninstalling Redis from namespace %q", a.ns)
	oc.DeleteFile(t, a.ns, examples.RedisYamlFile())
}

func (a *redis) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "redis")
}
