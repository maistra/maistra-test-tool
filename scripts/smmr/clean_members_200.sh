#!/bin/bash

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

oc apply -f ServiceMeshMemberRoll.yaml -n istio-system

for i in {1..200}
do
  oc delete project test$i
done
