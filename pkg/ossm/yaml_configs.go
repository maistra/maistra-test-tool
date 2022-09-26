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
  smcpV23_template = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: v2.3
  tracing:
    type: Jaeger
    sampling: 10000
  policy:
    type: Istiod
  addons:
    grafana:
      enabled: true
    jaeger:
      install:
        storage:
          type: Memory
    kiali:
      enabled: true
    prometheus:
      enabled: true
  
  telemetry:
    type: Istiod
`
  smcpV22_template = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: v2.2
  tracing:
    type: Jaeger
    sampling: 10000
  policy:
    type: Istiod
  addons:
    grafana:
      enabled: true
    jaeger:
      install:
        storage:
          type: Memory
    kiali:
      enabled: true
    prometheus:
      enabled: true
  
  telemetry:
    type: Istiod
`
	smcpV21_template = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: v2.1
  tracing:
    type: Jaeger
    sampling: 10000
  policy:
    type: Istiod
  addons:
    grafana:
      enabled: true
    jaeger:
      install:
        storage:
          type: Memory
    kiali:
      enabled: true
    prometheus:
      enabled: true
  
  telemetry:
    type: Istiod
`

	smcpV20_template = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: v2.0
  tracing:
    type: Jaeger
    sampling: 10000
  policy:
    type: Istiod
  addons:
    grafana:
      enabled: true
    jaeger:
      install:
        storage:
          type: Memory
    kiali:
      enabled: true
    prometheus:
      enabled: true
  
  telemetry:
    type: Istiod
`

	smcpV11_template = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: v1.1
  tracing:
    type: Jaeger
    sampling: 10000
  addons:
    grafana:
      enabled: true
    jaeger:
      install:
        storage:
          type: Memory
    kiali:
      enabled: true
    prometheus:
      enabled: true
`

	smmr = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  # a list of namespaces that should be joined into the service mesh
  # for example, to add the bookinfo namespace
  - bookinfo
  - foo
  - bar
  - legacy
`

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

	testSSLDeploymentZ = `
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
        image: quay.io/maistra/testssl:0.0-ibm-z
        imagePullPolicy: Always
`

	testSSLDeploymentP = `
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
        image: quay.io/maistra/testssl:0.0-ibm-p
        imagePullPolicy: Always
`

	rateLimitSMCPPatch = `
spec:
  techPreview:
    rateLimiting:
      rls:
        enabled: true
        storageBackend: redis
        storageAddress: redis.redis:6379
      rawRules:
        domain: productpage-ratelimit
        descriptors:
          - key: PATH
            value: "/productpage"
            rate_limit:
              unit: minute
              requests_per_unit: 1
          - key: PATH
            rate_limit:
              unit: minute
              requests_per_unit: 100

`

	testAnnotationProxyEnv = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testenv
spec:
  replicas: 1
  selector:
    matchLabels:
      app: env
  template:
    metadata:
      annotations:
        sidecar.maistra.io/proxyEnv: '{ "maistra_test_env": "env_value", "maistra_test_env_2": "env_value_2" }'
      labels:
        app: env
    spec:
      containers:
      - name: testenv
        image: quay.io/maistra/testssl:latest
        imagePullPolicy: Always
`

	testAnnotationProxyEnvZ = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testenv
spec:
  replicas: 1
  selector:
    matchLabels:
      app: env
  template:
    metadata:
      annotations:
        sidecar.maistra.io/proxyEnv: '{ "maistra_test_env": "env_value", "maistra_test_env_2": "env_value_2" }'
      labels:
        app: env
    spec:
      containers:
      - name: testenv
        image: quay.io/maistra/testssl:0.0-ibm-z
        imagePullPolicy: Always
`

	testAnnotationProxyEnvP = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testenv
spec:
  replicas: 1
  selector:
    matchLabels:
      app: env
  template:
    metadata:
      annotations:
        sidecar.maistra.io/proxyEnv: '{ "maistra_test_env": "env_value", "maistra_test_env_2": "env_value_2" }'
      labels:
        app: env
    spec:
      containers:
      - name: testenv
        image: quay.io/maistra/testssl:0.0-ibm-p
        imagePullPolicy: Always
`

	testSpecProxyEnv = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testenv
spec:
  replicas: 1
  selector:
    matchLabels:
      app: env
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "true"
      labels:
        app: env
    spec:
      containers:
      - name: testenv
        image: docker.io/nginx
        imagePullPolicy: Always
`

	ProxyEnvSMCPPath = `
spec:
  proxy:
    runtime:
      container:
        env:
          maistra_test_foo: maistra_test_bar
`

	testInitContainerYAML = `
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: sleep-init
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: sleep-init
    template:
      metadata:
        annotations:
          sidecar.istio.io/inject: "true"
        labels:
          app: sleep-init
      spec:
        terminationGracePeriodSeconds: 0

        initContainers:
        - name: init
          image: curlimages/curl
          command: ["/bin/echo", "init worked"]
          imagePullPolicy: IfNotPresent

        containers:
        - name: sleep
          image: curlimages/curl
          command: ["/bin/sleep", "3650d"]
          imagePullPolicy: IfNotPresent
 `
)
