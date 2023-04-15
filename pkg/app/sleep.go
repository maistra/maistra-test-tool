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
		oc.ApplyTemplate(t, a.ns, sleepNoSidecarTemplate, nil)
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

const sleepNoSidecarTemplate = `
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
      labels:
        app: sleep
    spec:
      terminationGracePeriodSeconds: 0
      serviceAccountName: sleep
      containers:
      - name: sleep
        image: quay.io/{{ perArch "openshifttest/sleep:multiarch" "maistra/governmentpaas-curl-ssl:0.0-ibm-p" "maistra/governmentpaas-curl-ssl:0.0-ibm-z" "openshifttest/sleep:multiarch" }}
        command: ["/bin/sleep", "3650d"]
        env:
        - name: HTTPS_PROXY
          valueFrom:
            configMapKeyRef:
              name: sleep-configmap
              key: https-proxy
        - name: HTTP_PROXY
          valueFrom:
            configMapKeyRef:
              name: sleep-configmap
              key: http-proxy
        - name: NO_PROXY
          valueFrom:
            configMapKeyRef:
              name: sleep-configmap
              key: no-proxy
        volumeMounts:
        - mountPath: /etc/sleep/tls
          name: secret-volume
      volumes:
      - name: secret-volume
        secret:
          secretName: sleep-secret
          optional: true
`
