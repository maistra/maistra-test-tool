#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
LOG="testdata/logs/new_log.yaml"

INGRESS_HOST="$(${OC_COMMAND} get routes -n istio-system -l app=istio-ingressgateway -o jsonpath='{.items[0].spec.host}')"
PROMETHEUS_ROUTE="$(${OC_COMMAND} get routes -n istio-system -l app=prometheus -o jsonpath='{.items[0].spec.host}')"

while getopts 'h:' OPTION; do
  case "$OPTION" in
    h) INGRESS_HOST="${OPTARG}" ;;
  esac
done
shift $((OPTIND-1))

INGRESS_PORT="$(${OC_COMMAND} -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].port}')"
SECURE_INGRESS_PORT="$(${OC_COMMAND} -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="https")].port}')"
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
    ${OC_COMMAND} delete -f ${LOG}
    #killall ${OC_COMMAND}
    echo "bookinfo" | ./bookinfo_uninstall.sh
}
trap cleanup EXIT


function check_logs() {
    ${OC_COMMAND} apply -f ${LOG}
    sleep 5

    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage

    echo
    echo "http://${PROMETHEUS_ROUTE}"
    echo "# Go to Prometheus Dashboard and query Execute 'istio_double_request_count'..."
    read -p "Press enter to continue: "
    sleep 2

    echo "# Verify logs stream has been created."
    ${OC_COMMAND} -n istio-system logs \
        $(${OC_COMMAND} -n istio-system get pods -l istio-mixer-type=telemetry -o jsonpath='{.items[0].metadata.name}') \
        -c mixer | grep \"instance\":\"newlog.logentry.istio-system\"
    echo
    echo "# Check logs stream..."
    read -p "Press enter to continue: "
}

function main() {
    banner "TC_32 Collecting Logs"
    echo "bookinfo" | ./bookinfo_install.sh

    sleep 10

    check_logs

    banner "TC_32 passed"
}

main
