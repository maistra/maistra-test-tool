#!/bin/bash


cat << EOF > /tmp/ServiceMeshMemberRoll.yaml
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
for i in $(seq 1 $numberOfNamespaces)
do
  oc create namespace test$i
  echo "    - test"$i >> /tmp/ServiceMeshMemberRoll.yaml
done

oc apply -f /tmp/ServiceMeshMemberRoll.yaml -n istio-system
