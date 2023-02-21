#!/bin/bash

for i in {1..2}; do
oc apply -f - <<EOF
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  annotations:
  name: gw-http-$i
  namespace: test$i
spec:
  selector:
    istio: ingressgateway
  servers:
  - hosts:
    - 'http$i.test.ocp'
    port:
      name: http
      number: 80
      protocol: HTTP
EOF
done 