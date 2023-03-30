package app

import (
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type httpbinV1 struct {
	ns string
}

var _ App = &httpbinV1{}

func HttpbinV1(ns string) App {
	return &httpbinV1{ns: ns}
}

func (a *httpbinV1) Name() string {
	return "httpbin-v1"
}

func (a *httpbinV1) Namespace() string {
	return a.ns
}

func (a *httpbinV1) Install(t test.TestHelper) {
	t.T().Helper()
	oc.ApplyFile(t, a.ns, examples.HttpbinV1YamlFile())
}

func (a *httpbinV1) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFile(t, a.ns, examples.HttpbinV1YamlFile())
}

func (a *httpbinV1) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "httpbin-v1")
}
