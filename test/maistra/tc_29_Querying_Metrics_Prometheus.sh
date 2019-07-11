#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"


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
    echo "bookinfo" | ./bookinfo_uninstall.sh
    #killall oc
}
trap cleanup EXIT

function query_metrics() {
    echo "# Verify prometheus service is running"
    ${OC_COMMAND} -n istio-system get svc prometheus
    
    #oc -n istio-system port-forward $(oc -n istio-system get pod -l app=prometheus -o jsonpath='{.items[0].metadata.name}') 9090:9090 &
    echo
    echo "https://${PROMETHEUS_ROUTE}" 
    echo "# Go to Prometheus dashboard"
    echo "# Query: istio_requests_total"
    echo '# Query: istio_requests_total{destination_service="productpage.bookinfo.svc.cluster.local"}'
    echo '# Query: istio_requests_total{destination_service="reviews.bookinfo.svc.cluster.local", destination_version="v3"}'
    echo '# Query: rate(istio_requests_total{destination_service=~"productpage.*", response_code="200"}[5m])'
    read -p "Press enter to continue: "
}

function main() {
    banner "TC_28 Querying Metrics Prometheus"
    echo "bookinfo" | ./bookinfo_install.sh
    sleep 10

    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage
    curl -o /dev/null -s -w "%{http_code}\n" http://$GATEWAY_URL/productpage

    query_metrics

    banner "TC_28 passed"
}

main
