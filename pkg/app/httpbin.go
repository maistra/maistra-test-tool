package app

import (
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type httpbin struct {
	ns string
}

var _ App = &httpbin{}

func Httpbin(ns string) App {
	return &httpbin{ns: ns}
}

func (a *httpbin) Name() string {
	return "httpbin"
}

func (a *httpbin) Namespace() string {
	return a.ns
}

func (a *httpbin) Install(t test.TestHelper) {
	t.T().Helper()
	oc.ApplyFile(t, a.ns, examples.HttpbinYamlFile())
}

func (a *httpbin) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFile(t, a.ns, examples.HttpbinYamlFile())
}

func (a *httpbin) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "httpbin")
}
