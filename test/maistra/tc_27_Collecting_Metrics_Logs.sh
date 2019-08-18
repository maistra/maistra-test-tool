#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
MESH="service-mesh-1"
TELEMETRY="testdata/telemetry/new_telemetry.yaml"

INGRESS_HOST="$(${OC_COMMAND} get routes -n ${MESH} -l app=istio-ingressgateway -o jsonpath='{.items[0].spec.host}')"
PROMETHEUS_ROUTE="$(${OC_COMMAND} get routes -n ${MESH} -l app=prometheus -o jsonpath='{.items[0].spec.host}')"

while getopts 'h:' OPTION; do
  case "$OPTION" in
    h) INGRESS_HOST="${OPTARG}" ;;
  esac
done
shift $((OPTIND-1))

INGRESS_PORT="$(${OC_COMMAND} -n ${MESH} get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].port}')"
SECURE_INGRESS_PORT="$(${OC_COMMAND} -n ${MESH} get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="https")].port}')"
GATEWAY_URL="${INGRESS_HOST}:${INGRESS_PORT}"


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
    ${OC_COMMAND} delete -f ${TELEMETRY} -n ${MESH}
    #killall ${OC_COMMAND}
    echo "bookinfo" | ./bookinfo_uninstall.sh
}
trap cleanup EXIT


function check_metrics() {

    #cat ${TELEMETRY} | sed 's@\[mesh\]@${MESH}@g' | ${OC_COMMAND} apply -n ${MESH} -f -
    ${OC_COMMAND} apply -f ${TELEMETRY} -n ${MESH}
    sleep 5
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage

    echo
    echo "https://${PROMETHEUS_ROUTE}"
    echo "# Go to Prometheus Dashboard and query Execute 'istio_double_request_count'..."
    read -p "Press enter to continue: "
    sleep 2

    echo "# Verify logs stream has been created."
    ${OC_COMMAND} -n ${MESH} logs \
        $(${OC_COMMAND} -n ${MESH} get pods -l istio-mixer-type=telemetry -o jsonpath='{.items[0].metadata.name}') \
        -c mixer | grep \"instance\":\"newlog.logentry.${MESH}\"
    echo
    echo "# Check logs stream..."
    read -p "Press enter to continue: "
}

function main() {
    banner "TC_27 Collecting Metrics Logs"
    echo "bookinfo" | ./bookinfo_install.sh

    sleep 10

    check_metrics

    banner "TC_27 passed"
}

main
