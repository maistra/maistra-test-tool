// Copyright 2021 Red Hat, Inc.
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

package ossm

const (
	httpbinServiceMeshExtension = `
apiVersion: maistra.io/v1
kind: ServiceMeshExtension
metadata:
  name: header-append
spec:
  config:
    maistra: rocks
  image: quay.io/maistra-dev/header-append-filter:2.1
  phase: PostAuthZ
  priority: 1000
  workloadSelector:
    labels:
      app: httpbin
`

	testSSLDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testssl
spec:
  replicas: 1
  selector:
    matchLabels:
      app: testssl
  template:
    metadata:
      labels:
        app: testssl
    spec:
      containers:
      - name: testssl
        image: quay.io/maistra/testssl:latest
        imagePullPolicy: Always
`
)
