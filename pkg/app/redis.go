package app

import (
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type redis struct {
	ns string
}

var _ App = &redis{}

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
	oc.ApplyTemplate(t, a.ns, redisTemplate, nil)
}

func (a *redis) Uninstall(t test.TestHelper) {
	t.T().Helper()
	t.Logf("Uninstall Redis from namespace %q", a.ns)
	oc.DeleteFromTemplate(t, a.ns, redisTemplate, nil)
}

func (a *redis) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "redis")
}

const redisTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: redis
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  labels:
    app: redis
spec:
  ports:
  - port: 6379
    name: redis-port
  selector:
    app: redis
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
      labels:
        app: redis
    spec:
      terminationGracePeriodSeconds: 0
      serviceAccountName: redis
      containers:
      - name: redis
        image: docker.io/redis:6.2	# multi-arch image (supports x86, p, z, arm)
        ports:
        - containerPort: 6379
`
