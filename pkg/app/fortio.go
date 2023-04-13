package app

import (
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type fortio struct {
	ns string
}

var _ App = &fortio{}

func Fortio(ns string) App {
	return &fortio{ns: ns}
}

func (a *fortio) Name() string {
	return "fortio"
}

func (a *fortio) Namespace() string {
	return a.ns
}

func (a *fortio) Install(t test.TestHelper) {
	t.T().Helper()
	oc.ApplyFile(t, a.ns, examples.FortioYamlFile())
}

func (a *fortio) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFile(t, a.ns, examples.FortioYamlFile())
}

func (a *fortio) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "fortio-deploy")
}
