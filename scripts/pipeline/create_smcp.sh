#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

# This script creates a basic SMCP cr with given version in istio-system namespace.
# SMCP_VERSION env variable is required
# oc client is expected to be logged in

smcp_namespace="istio-system"
smcp_name="basic-smcp"

if ! oc get namespace ${smcp_namespace}
then
  oc new-project ${smcp_namespace}

  # wait for servicemeshoperator to be sucesfully installed in the newly created namespace
  i=1
  until oc get csv -n ${smcp_namespace} 2>&1 | grep servicemeshoperator | grep Succeeded
  do
    if [ $i -gt 10 ]
    then
      echo "Timeout waiting for servicemeshoperator installation"
      exit 1
    fi

    echo "Waiting for servicemeshoperator installation"
    sleep 10
    ((i=i+1))
  done

  # workaround for https://issues.redhat.com/browse/OSSM-521
  sleep 120
fi

oc apply -f - <<EOF
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: ${smcp_name}
  namespace: ${smcp_namespace}
spec:
  version: ${SMCP_VERSION}
  security:
    dataPlane:
      mtls: true
      automtls: true
    controlPlane:
      mtls: true
  tracing:
    type: Jaeger
  addons:
    jaeger:
      install:
        storage:
          type: Memory
    grafana:
      enabled: true
    kiali:
      enabled: true
    prometheus:
      enabled: true
EOF

oc wait --for condition=Ready smcp/${smcp_name} -n ${smcp_namespace} --timeout=180s
