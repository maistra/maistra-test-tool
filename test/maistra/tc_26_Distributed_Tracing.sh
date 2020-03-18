#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
MESH="istio-system"

INGRESS_HOST="$(${OC_COMMAND} get routes -n ${MESH} -l app=istio-ingressgateway -o jsonpath='{.items[0].spec.host}')"
JAEGER_ROUTE="$(${OC_COMMAND} get routes -n ${MESH} -l app=jaeger -o jsonpath='{.items[0].spec.host}')"

while getopts 'h:' OPTION; do
  case "$OPTION" in
    h) INGRESS_HOST="${OPTARG}" ;;
  esac
done
shift $((OPTIND-1))

INGRESS_PORT="$(${OC_COMMAND} -n ${MESH} get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].port}')"
SECURE_INGRESS_PORT="$(${OC_COMMAND} -n ${MESH} get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="https")].port}')"
GATEWAY_URL="${INGRESS_HOST}:${INGRESS_PORT}"

function cleanup() {
  set +e
  banner "Cleanup"
  echo "bookinfo" | ./bookinfo_uninstall.sh
}
trap cleanup EXIT


function banner() {
  message="$1"
  border="$(echo ${message} | sed -e 's+.+=+g')"
  echo "${border}"
  echo "${message}"
  echo "${border}"
}

function check_jaeger() {
    echo
    echo "https://${JAEGER_ROUTE}"
    echo "# Go to Jaeger dashboard and select Service 'productpage'"
    echo "# Click 'Find Traces'"
    echo "# Check Tracing spans from Jaeger dashboard..."
    read -p "Press enter to continue: "
}

function main() {
    banner "TC_26 Distributed Tracing"
    echo "bookinfo" | ./bookinfo_install.sh

    sleep 10
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    
    check_jaeger
    banner "TC_26 passed"
}

main
