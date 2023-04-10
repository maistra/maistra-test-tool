package app

import (
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type sleep struct {
	ns            string
	injectSidecar bool
}

var _ App = &sleep{}

func Sleep(ns string) App {
	return &sleep{ns: ns, injectSidecar: true}
}

func SleepNoSidecar(ns string) App {
	return &sleep{ns: ns, injectSidecar: false}
}

func (a *sleep) Name() string {
	return "sleep"
}

func (a *sleep) Namespace() string {
	return a.ns
}

func (a *sleep) Install(t test.TestHelper) {
	t.T().Helper()
	proxy, _ := util.GetProxy()
	configMapYAML := util.RunTemplate(examples.SleepConfigMap(), proxy)
	oc.ApplyString(t, a.ns, configMapYAML)
	if a.injectSidecar {
		oc.ApplyFile(t, a.ns, examples.SleepYamlFile())
	} else {
		oc.ApplyFile(t, a.ns, examples.SleepLegacyYamlFile())
	}
}

func (a *sleep) Uninstall(t test.TestHelper) {
	t.T().Helper()
	proxy, _ := util.GetProxy()
	configMapYAML := util.RunTemplate(examples.SleepConfigMap(), proxy)
	oc.DeleteFromString(t, a.ns, configMapYAML)
	oc.DeleteFile(t, a.ns, examples.SleepYamlFile())
}

func (a *sleep) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "sleep")
}
