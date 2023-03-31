#!/bin/bash

oc apply -n istio-system -f - <<EOF
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
  namespace: istio-system
spec:
  members:
    - bookinfo
    - foo
    - bar
    - legacy
EOF

numberOfNamespaces=$1
projects=$(eval echo test{1..$numberOfNamespaces})
oc delete namespace $projects
