package app

import (
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type httpbinV2 struct {
	ns string
}

var _ App = &httpbinV2{}

func HttpbinV2(ns string) App {
	return &httpbinV2{ns: ns}
}

func (a *httpbinV2) Name() string {
	return "httpbin-v2"
}

func (a *httpbinV2) Namespace() string {
	return a.ns
}

func (a *httpbinV2) Install(t test.TestHelper) {
	t.T().Helper()
	oc.ApplyFile(t, a.ns, examples.HttpbinV2YamlFile())
}

func (a *httpbinV2) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFile(t, a.ns, examples.HttpbinV2YamlFile())
}

func (a *httpbinV2) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "httpbin-v2")
}
