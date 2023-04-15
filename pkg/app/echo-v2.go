// Copyright 2023 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

type echoV2 struct {
	ns string
}

var _ App = &echoV2{}

func EchoV2(ns string) App {
	return &echoV2{ns: ns}
}

func (a *echoV2) Name() string {
	return "echo-v2"
}

func (a *echoV2) Namespace() string {
	return a.ns
}

func (a *echoV2) Install(t test.TestHelper) {
	t.T().Helper()
	oc.ApplyTemplate(t, a.ns, tcpEchoV2Template, nil)
}

func (a *echoV2) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFromTemplate(t, a.ns, tcpEchoV2Template, nil)
}

func (a *echoV2) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "tcp-echo-v2")
}

const tcpEchoV2Template = `
apiVersion: v1
kind: Service
metadata:
  name: tcp-echo
  labels:
    app: tcp-echo
spec:
  ports:
  - name: tcp
    port: 9000
  #- name: tcp-other
  #  port: 9001
  ## Port 9002 is omitted intentionally for testing the pass through filter chain.
  selector:
    app: tcp-echo
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tcp-echo-v2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tcp-echo
      version: v2
  template:
    metadata:
      labels:
        app: tcp-echo
        version: v2
    spec:
      containers:
      - name: tcp-echo
        image: {{ perArch "docker.io/istio/tcp-echo-server:1.2" "quay.io/maistra/tcp-echo-server:0.0-ibm-p" "quay.io/maistra/tcp-echo-server:2.0-ibm-z" "docker.io/istio/tcp-echo-server:1.2" }}
        imagePullPolicy: IfNotPresent
        #args: [ "9000,9001,9002", "two" ]
        args: [ "9000", "two" ]
        ports:
        - containerPort: 9000
        #- containerPort: 9001
`
