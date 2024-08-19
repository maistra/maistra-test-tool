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

type minio struct {
	ns string
}

var _ App = &minio{}

func Minio(ns string) App {
	return &minio{ns: ns}
}

func (a *minio) Name() string {
	return "minio"
}

func (a *minio) Namespace() string {
	return a.ns
}

func (a *minio) Install(t test.TestHelper) {
	t.T().Helper()
	if !oc.AnyResourceExist(t, "", "storageclass") {
		t.Fatal("Your cluster doesn't contain any storageclass. Minio cannot be installed due to the required dynamic provisioning storage unavailable!")
	}
	oc.ApplyTemplate(t, a.ns, minioTemplate, nil)
	oc.WaitDeploymentRolloutComplete(t, a.ns, "minio")
	minioRoute := oc.DefaultOC.GetRouteURL(t, a.ns, "minio-route")
	oc.ApplyTemplate(t, a.ns, minioSecretTemplate, map[string]string{"minioRoute": minioRoute})
}

func (a *minio) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFromTemplate(t, a.ns, minioTemplate, nil)
}

func (a *minio) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "minio")
}

const minioTemplate = `
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
 # This name uniquely identifies the PVC. Will be used in deployment below.
 name: minio-pv-claim
 labels:
   app: minio-storage-claim
spec:
 # Read more about access modes here: http://kubernetes.io/docs/user-guide/persistent-volumes/#access-modes
 accessModes:
   - ReadWriteOnce
 resources:
   # This is the request for storage. Should be available in the cluster.
   requests:
     storage: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
 name: minio
spec:
 selector:
   matchLabels:
     app: minio
 strategy:
   type: Recreate
 template:
   metadata:
     labels:
       # Label is used as selector in the service.
       app: minio
   spec:
     # Refer to the PVC created earlier
     volumes:
       - name: storage
         persistentVolumeClaim:
           # Name of the PVC created earlier
           claimName: minio-pv-claim
     initContainers:
       - name: create-buckets
         image: {{ image "busybox" }}
         command:
           - "sh"
           - "-c"
           - "mkdir -p /storage/tempo-data"
         volumeMounts:
           - name: storage # must match the volume name, above
             mountPath: "/storage"
     containers:
       - name: minio
         # Pulls the default Minio image from Docker Hub
         image: {{ image "minio" }}
         args:
           - server
           - /storage
           - --console-address
           - ":9001"
         env:
           # Minio access key and secret key
           - name: MINIO_ROOT_USER
             value: "minio"
           - name: MINIO_ROOT_PASSWORD
             value: "minio123"
         ports:
           - containerPort: 9000
           - containerPort: 9001
         volumeMounts:
           - name: storage # must match the volume name, above
             mountPath: "/storage"
---
apiVersion: v1
kind: Service
metadata:
 name: minio
spec:
 type: ClusterIP
 ports:
   - port: 9000
     targetPort: 9000
     protocol: TCP
     name: api
   - port: 9001
     targetPort: 9001
     protocol: TCP
     name: console
 selector:
   app: minio
---
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: minio-route
spec:
  to:
    kind: Service
    name: minio
  port:
    targetPort: api
`

const minioSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: my-storage-secret
type: Opaque
stringData:
  endpoint: http://{{ .minioRoute }}
  bucket: tempo-data
  access_key_id: minio
  access_key_secret: minio123
`
