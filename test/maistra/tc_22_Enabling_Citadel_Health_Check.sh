#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"

function banner() {
  message="$1"
  border="$(echo ${message} | sed -e 's+.+=+g')"
  echo "${border}"
  echo "${message}"
  echo "${border}"
}

function cleanup() {
  set +e
  banner "Cleanup"

  ${OC_COMMAND} rollout undo deployment -n istio-system istio-citadel
  ${OC_COMMAND} delete meshpolicy default
  rm -f /tmp/istio-citadel-new.yaml
  rm -f /tmp/istio-citadel-bak.yaml
}
trap cleanup EXIT

function enable_mtls() {
    cat <<EOF | ${OC_COMMAND} apply -f -
apiVersion: "authentication.istio.io/v1alpha1"
kind: "MeshPolicy"
metadata:
  name: "default"
spec:
  peers:
  - mtls: {}
EOF
}

function update_citadel_yaml() {
  ${OC_COMMAND} get deployment -n istio-system istio-citadel -o yaml > /tmp/istio-citadel-bak.yaml
  cp -f /tmp/istio-citadel-bak.yaml /tmp/istio-citadel-new.yaml
  TAB="        "
  sed -i $"/self-signed-ca=true/a\\${TAB}- --liveness-probe-path=/tmp/ca.liveness\n${TAB}- --liveness-probe-interval=10s\n${TAB}- --probe-check-interval=10s" \
    /tmp/istio-citadel-new.yaml
  sed -i $"/imagePullPolicy:/a\\${TAB}livenessProbe:\n${TAB}  exec:\n${TAB}    command:\n${TAB}    - /usr/local/bin/istio_ca\n${TAB}    - probe\n${TAB}    - --probe-path=/tmp/ca.liveness\n${TAB}    - --interval=125s\n${TAB}  initialDelaySeconds: 10\n${TAB}  periodSeconds: 10" \
    /tmp/istio-citadel-new.yaml
}

function deploy_citadel() {
  
  update_citadel_yaml
  ${OC_COMMAND} apply -n istio-system -f /tmp/istio-citadel-new.yaml

}

function verify_health_check() {
  echo "# sleep 30"
  sleep 30
  ${OC_COMMAND} logs `${OC_COMMAND} get po -n istio-system | grep istio-citadel | awk '{print $1}'` -n istio-system | grep CSR
  if [ $? != 0 ]; then
    echo "# Error: health check failed"
    exit 1
  fi
}


function main() {
  banner "Enabling Citadel health checking"
  enable_mtls
  deploy_citadel
  verify_health_check
  
  banner "TC_22 passed"
}

main
