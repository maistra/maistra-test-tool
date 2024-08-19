// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
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
	oc.ApplyTemplate(t, a.ns, fortioTemplate, nil)
}

func (a *fortio) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFromTemplate(t, a.ns, fortioTemplate, nil)
}

func (a *fortio) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "fortio-deploy")
}

const fortioTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: fortio
  labels:
    app: fortio
spec:
  ports:
  - port: 8080
    name: http
  selector:
    app: fortio
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fortio-deploy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: fortio
  template:
    metadata:
      annotations:
        # This annotation causes Envoy to serve cluster.outbound statistics via 15000/stats
        # in addition to the stats normally served by Istio.  The Circuit Breaking example task
        # gives an example of inspecting Envoy stats.
        sidecar.istio.io/inject: "true"
        sidecar.istio.io/statsInclusionPrefixes: cluster.outbound,cluster_manager,listener_manager,http_mixer_filter,tcp_mixer_filter,server,cluster.xds-grpc
      labels:
        app: fortio
    spec:
      containers:
      - name: fortio
        image: {{ image "fortio" }}
        ports:
        - containerPort: 8080
          name: http-fortio
        - containerPort: 8079
          name: grpc-ping
`
