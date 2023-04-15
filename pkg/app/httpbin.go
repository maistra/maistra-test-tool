package app

import (
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type httpbin struct {
	ns             string
	injectSidecar  bool
	deploymentName string
	versionLabel   string
}

var _ App = &httpbin{}

func Httpbin(ns string) App {
	return &httpbin{
		ns:             ns,
		injectSidecar:  true,
		deploymentName: "httpbin",
		versionLabel:   "v1",
	}
}

func HttpbinNoSidecar(ns string) App {
	return &httpbin{
		ns:             ns,
		injectSidecar:  false,
		deploymentName: "httpbin",
		versionLabel:   "v1",
	}
}

func HttpbinV1(ns string) App {
	return &httpbin{
		ns:             ns,
		injectSidecar:  true,
		deploymentName: "httpbin-v1",
		versionLabel:   "v1",
	}
}

func HttpbinV2(ns string) App {
	return &httpbin{
		ns:             ns,
		injectSidecar:  true,
		deploymentName: "httpbin-v2",
		versionLabel:   "v2",
	}
}

func (a *httpbin) Name() string {
	return a.deploymentName
}

func (a *httpbin) Namespace() string {
	return a.ns
}

func (a *httpbin) Install(t test.TestHelper) {
	t.T().Helper()
	oc.ApplyTemplate(t, a.ns, httpbinTemplate, a.values())
}

func (a *httpbin) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFromTemplate(t, a.ns, httpbinTemplate, a.values())
}

func (a *httpbin) values() map[string]interface{} {
	return map[string]interface{}{
		"InjectSidecar": a.injectSidecar,
		"Name":          a.deploymentName,
		"Version":       a.versionLabel,
	}
}

func (a *httpbin) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, a.deploymentName)
}

const httpbinTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: httpbin
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
  labels:
    app: httpbin
    service: httpbin
spec:
  ports:
  - name: http
    port: 8000
    targetPort: 8000
  selector:
    app: httpbin
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpbin
      version: {{ .Version }}
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "{{ .InjectSidecar }}"
      labels:
        app: httpbin
        version: {{ .Version }}
    spec:
      serviceAccountName: httpbin
      containers:
      - name: httpbin
        image: {{ image "httpbin" }}
        command: ["gunicorn", "--access-logfile", "-", "-b", "[::]:8000", "httpbin:app"]
        ports:
        - containerPort: 8000
`
