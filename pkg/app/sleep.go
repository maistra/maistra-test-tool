package app

import (
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
	oc.ApplyTemplate(t, a.ns, sleepTemplate, a.values())
}

func (a *sleep) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFromTemplate(t, a.ns, sleepTemplate, a.values())
}

func (a *sleep) values() map[string]interface{} {
	proxy, _ := util.GetProxy()
	return map[string]interface{}{
		"InjectSidecar": a.injectSidecar,
		"HttpProxy":     proxy.HTTPProxy,
		"HttpsProxy":    proxy.HTTPSProxy,
		"NoProxy":       proxy.NoProxy,
	}
}

func (a *sleep) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "sleep")
}

const sleepTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sleep
---
apiVersion: v1
kind: Service
metadata:
  name: sleep
  labels:
    app: sleep
spec:
  ports:
  - port: 80
    name: http
  selector:
    app: sleep
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sleep
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sleep
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "{{ .InjectSidecar }}"
      labels:
        app: sleep
    spec:
      terminationGracePeriodSeconds: 0
      serviceAccountName: sleep
      containers:
      - name: sleep
        image: {{ image "sleep" }} 
        command: ["/bin/sleep", "3650d"]
        env:
        - name: HTTPS_PROXY
          value: {{ .HttpsProxy }}
        - name: HTTP_PROXY
          value: {{ .HttpProxy }}
        - name: NO_PROXY
          value: {{ .NoProxy }}
        volumeMounts:
        - mountPath: /etc/sleep/tls
          name: secret-volume
      volumes:
      - name: secret-volume
        secret:
          secretName: sleep-secret
          optional: true
`
