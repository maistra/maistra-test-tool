// Copyright 2024 Red Hat, Inc.
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

type grpcurl struct {
	ns string
}

var _ App = &grpcurl{}

func GrpCurl(ns string) App {

	return &grpcurl{
		ns: ns,
	}
}

func (a *grpcurl) Name() string {
	return "grpcurl"
}

func (a *grpcurl) Namespace() string {
	return a.ns
}

func (a *grpcurl) Install(t test.TestHelper) {
	t.T().Helper()
	oc.ApplyTemplate(t, a.ns, grpcCurlTemplate, nil)
}

func (a *grpcurl) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFromTemplate(t, a.ns, grpcCurlTemplate, nil)
}

func (a *grpcurl) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitCondition(t, a.ns, "Jobs", "grpcurl", "complete")
}

// TODO: if you want to use different `grpcurl` command as
// grpcurl -insecure -authority grpc.example.com istio-ingressgateway.istio-system:443 list
// refactor Job to Deployments and run command via `oc exec`
const grpcCurlTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: grpcurl
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: grpcurl
        version: v1
    spec:
      containers:
      - name: grpcurl
        image: {{ image "grpcurl" }}
        imagePullPolicy: IfNotPresent
        command: ["sh", "-c"]
        args: 
        - |
          echo "Empty command for grpc service to be ready"
          grpcurl -insecure -authority grpc.example.com istio-ingressgateway.istio-system:443 list
        ports:
        - containerPort: 443
      restartPolicy: Never
`
