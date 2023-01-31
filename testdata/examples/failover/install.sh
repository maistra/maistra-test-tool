#!/bin/bash

# Copyright Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# shellcheck disable=SC1091
source common.sh

log "Retrieving root certificates"
WEST_MESH_CERT=$(oc1 get configmap -n west-mesh-system istio-ca-root-cert -o jsonpath='{.data.root-cert\.pem}' | sed ':a;N;$!ba;s/\n/\\\n    /g')
EAST_MESH_CERT=$(oc2 get configmap -n east-mesh-system istio-ca-root-cert -o jsonpath='{.data.root-cert\.pem}' | sed ':a;N;$!ba;s/\n/\\\n    /g')

n=0
until [ "$n" -ge 4 ]
do
  if [ -z "$MESH1_CERT" ] || [ -z "$MESH2_CERT" ]; then
    log "Retrieving root certificates (retry)"
    sleep 30
    MESH1_CERT=$(oc1 get configmap -n mesh1-system istio-ca-root-cert -o jsonpath='{.data.root-cert\.pem}' | sed ':a;N;$!ba;s/\n/\\\n    /g')
    MESH2_CERT=$(oc2 get configmap -n mesh2-system istio-ca-root-cert -o jsonpath='{.data.root-cert\.pem}' | sed ':a;N;$!ba;s/\n/\\\n    /g')
    n=$((n+1))
  else
    log "Both root certificates retrieved"
    break
  fi
done

WEST_MESH_DISCOVERY_PORT="${MESH1_DISCOVERY_PORT:-8188}"
WEST_MESH_SERVICE_PORT="${MESH1_SERVICE_PORT:-15443}"
EAST_MESH_DISCOVERY_PORT="${MESH2_DISCOVERY_PORT:-8188}"
EAST_MESH_SERVICE_PORT="${MESH2_SERVICE_PORT:-15443}"

log "Retrieving ingress addresses"
if [ "${MESH1_KUBECONFIG}" == "${MESH2_KUBECONFIG}" ]; then
  echo "Single cluster detected; using cluster-local service for ingress"
  WEST_MESH_ADDRESS=east-mesh-ingress.west-mesh-system.svc.cluster.local
  EAST_MESH_ADDRESS=west-mesh-ingress.east-mesh-system.svc.cluster.local
else
  echo "Two clusters detected; using load-balancer service for ingress"

  while [ -z "$WEST_MESH_ADDRESS" ]
  do
    WEST_MESH_ADDRESS=$(oc1 -n west-mesh-system get svc east-mesh-ingress -o jsonpath="{.status.loadBalancer.ingress[].ip}")
    if [ -z "$WEST_MESH_ADDRESS" ]; then
      WEST_MESH_ADDRESS=$(oc1 -n west-mesh-system get svc east-mesh-ingress -o jsonpath="{.status.loadBalancer.ingress[].hostname}")
      if [ -z "$WEST_MESH_ADDRESS" ]; then
        echo "Waiting for load balancer to be provisioned for Service west-mesh-system/east-mesh-ingress..."
        sleep 30
      fi
    fi
  done

  while [ -z "$EAST_MESH_ADDRESS" ]
  do
    EAST_MESH_ADDRESS=$(oc2 -n east-mesh-system get svc west-mesh-ingress -o jsonpath="{.status.loadBalancer.ingress[].ip}")
    if [ -z "$EAST_MESH_ADDRESS" ]; then
      EAST_MESH_ADDRESS=$(oc2 -n east-mesh-system get svc west-mesh-ingress -o jsonpath="{.status.loadBalancer.ingress[].hostname}")
      if [ -z "$EAST_MESH_ADDRESS" ]; then
        echo "Waiting for load balancer to be provisioned for Service east-mesh-system/west-mesh-ingress..."
        sleep 30
      fi
    fi
  done
fi

echo
echo WEST_MESH_ADDRESS="${WEST_MESH_ADDRESS}"
echo WEST_MESH_DISCOVERY_PORT="${WEST_MESH_DISCOVERY_PORT}"
echo WEST_MESH_SERVICE_PORT="${WEST_MESH_SERVICE_PORT}"
echo
echo EAST_MESH_ADDRESS="${EAST_MESH_ADDRESS}"
echo EAST_MESH_DISCOVERY_PORT="${EAST_MESH_DISCOVERY_PORT}"
echo EAST_MESH_SERVICE_PORT="${EAST_MESH_SERVICE_PORT}"



log "Enabling federation for west-mesh"
sed "s:{{EAST_MESH_CERT}}:$EAST_MESH_CERT:g" export/configmap.yaml | oc1 apply -f -
sleep 10
sed -e "s:{{EAST_MESH_ADDRESS}}:$EAST_MESH_ADDRESS:g" -e "s:{{EAST_MESH_DISCOVERY_PORT}}:$EAST_MESH_DISCOVERY_PORT:g" -e "s:{{EAST_MESH_SERVICE_PORT}}:$EAST_MESH_SERVICE_PORT:g" export/servicemeshpeer.yaml | oc1 apply -f -
#oc1 apply -f export/exportedserviceset.yaml

log "Enabling federation for east-mesh"
sed "s:{{WEST_MESH_CERT}}:$WEST_MESH_CERT:g" import/configmap.yaml | oc2 apply -f -
sleep 10
sed -e "s:{{WEST_MESH_ADDRESS}}:$WEST_MESH_ADDRESS:g" -e "s:{{WEST_MESH_DISCOVERY_PORT}}:$WEST_MESH_DISCOVERY_PORT:g" -e "s:{{WEST_MESH_SERVICE_PORT}}:$WEST_MESH_SERVICE_PORT:g" import/servicemeshpeer.yaml | oc2 apply -f -
#oc2 apply -f import/importedserviceset.yaml

log "Installing bookinfo in west-mesh"
oc1 -n bookinfo-ha apply -f bookinfo/bookinfo.yaml
oc1 -n bookinfo-ha apply -f bookinfo/destination-rule-all.yaml

log "Installing bookinfo in east-mesh"
oc2 -n bookinfo-ha apply -f bookinfo/bookinfo.yaml
oc2 -n bookinfo-ha apply -f bookinfo/bookinfo-gateway.yaml
oc2 -n bookinfo-ha apply -f bookinfo/destination-rule-all.yaml
oc2 -n bookinfo-ha apply -f bookinfo/virtual-service-reviews-v3.yaml

#log "Enable failover for ratings service"
#oc2 apply -f examples/destinationrule-failover.yaml

sleep 20
log "INSTALLATION COMPLETE"

oc1 -n west-mesh-system get servicemeshpeer east-mesh -o json
oc2 -n east-mesh-system get servicemeshpeer west-mesh -o json
#oc2 -n east-mesh-system get importedservicesets west-mesh -o json

echo
echo  "If servicemeshpeer connection is false. Then" 
echo  "Wait 10 minutes and then run install.sh again."
echo

echo "OCP node failure-domain region value for cluster01"
oc1 describe node  | grep failure-domain.beta.kubernetes.io/region
echo "OCP node failure-domain region value for cluster02"
oc2 describe node  | grep failure-domain.beta.kubernetes.io/region


