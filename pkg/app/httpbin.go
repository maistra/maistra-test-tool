package app

import (
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type httpbin struct {
	ns            string
	injectSidecar bool
}

var _ App = &httpbin{}

func Httpbin(ns string) App {
	return &httpbin{ns: ns, injectSidecar: true}
}

func HttpbinNoSidecar(ns string) App {
	return &httpbin{ns: ns, injectSidecar: false}
}

func (a *httpbin) Name() string {
	return "httpbin"
}

func (a *httpbin) Namespace() string {
	return a.ns
}

func (a *httpbin) Install(t test.TestHelper) {
	t.T().Helper()
	if a.injectSidecar {
		oc.ApplyFile(t, a.ns, examples.HttpbinYamlFile())
	} else {
		oc.ApplyFile(t, a.ns, examples.HttpbinLegacyYamlFile())
	}
}

func (a *httpbin) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFile(t, a.ns, examples.HttpbinYamlFile())
}

func (a *httpbin) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "httpbin")
}
