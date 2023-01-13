#!/bin/bash

mkdir -p namespaces

cat << EOF > ServiceMeshMemberRoll.yaml
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

for i in {1..50}
do

  cat << EOF > namespaces/namespace-test$i.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: test$i
EOF

  oc apply -f namespaces/namespace-test$i.yaml
  echo "    - test"$i >> ServiceMeshMemberRoll.yaml

done

oc apply -f ServiceMeshMemberRoll.yaml -n istio-system
