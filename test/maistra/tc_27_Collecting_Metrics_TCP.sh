#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
TCP_TELEMETRY="testdata/telemetry/tcp_telemetry.yaml"
BOOKINFO_RATE_V2="testdata/bookinfo/networking/bookinfo-ratings-v2.yaml"
BOOKINFO_DB="testdata/bookinfo/platform/kube/bookinfo-db.yaml"
BOOKINFO_RULE="testdata/bookinfo/networking/destination-rule-all.yaml"
VS_RATE_DB="testdata/bookinfo/networking/virtual-service-ratings-db.yaml"

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
    ${OC_COMMAND} delete -f ${TCP_TELEMETRY} -n bookinfo
    ${OC_COMMAND} delete -f ${BOOKINFO_RATE_V2} -n bookinfo
    ${OC_COMMAND} delete -f ${BOOKINFO_DB} -n bookinfo
    ${OC_COMMAND} delete -f ${BOOKINFO_RULE} -n bookinfo
    ${OC_COMMAND} delete -f ${VS_RATE_DB} -n bookinfo
    echo "bookinfo" | ./bookinfo_uninstall.sh
    #killall oc
}
trap cleanup EXIT

function check_tcp_metrics() {
    ${OC_COMMAND} apply -f ${TCP_TELEMETRY} -n bookinfo
    sleep 2
    ${OC_COMMAND} apply -f ${BOOKINFO_RATE_V2} -n bookinfo
    ${OC_COMMAND} apply -f ${BOOKINFO_DB} -n bookinfo
    ${OC_COMMAND} apply -f ${BOOKINFO_RULE} -n bookinfo
    ${OC_COMMAND} apply -f ${VS_RATE_DB} -n bookinfo
    sleep 5

    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage

    #${OC_COMMAND} -n istio-system port-forward $(${OC_COMMAND} -n istio-system get pod -l app=prometheus -o jsonpath='{.items[0].metadata.name}') 9090:9090 &

    echo
    echo "http://${PROMETHEUS_ROUTE}"
    echo "# Go to Prometheus Dashboard and query Execute 'istio_mongo_received_bytes'..."
    read -p "Press enter to continue: "
}

function main() {
    banner "TC_27 Collecting Metrics TCP"
    echo "bookinfo" | ./bookinfo_install.sh
    sleep 10

    check_tcp_metrics

    banner "TC_27 passed"
}

main
