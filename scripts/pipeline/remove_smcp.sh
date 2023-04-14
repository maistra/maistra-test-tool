#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

# This script removes 'basic-smcp' from istio-system namespace and removes the namespace.
# oc client is expected to be logged in

smcp_namespace="istio-system"
smcp_name="basic-smcp"

oc delete smcp/${smcp_name} -n ${smcp_namespace} || true
oc delete project ${smcp_namespace} || true
# oc delete project does not wait for a namespace to be removed, we need to also call 'oc delete namespace'
oc delete namespace ${smcp_namespace} || true
