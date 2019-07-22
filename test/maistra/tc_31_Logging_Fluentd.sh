#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
LOGGING_STACK="testdata/telemetry/logging-stack.yaml"
FLUENTD_ISTIO="testdata/telemetry/fluentd-istio.yaml"

INGRESS_HOST="$(${OC_COMMAND} get routes -n istio-system -l app=istio-ingressgateway -o jsonpath='{.items[0].spec.host}')"

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
    ${OC_COMMAND} delete -f ${FLUENTD_ISTIO}
    ${OC_COMMAND} delete -f ${LOGGING_STACK}
    echo "bookinfo" | ./bookinfo_uninstall.sh
    killall ${OC_COMMAND}
}
trap cleanup EXIT


function apply_stack() {

    set +e
	${OC_COMMAND} new-project logging
	set -e
    # grant priviledged permission
	# ${OC_COMMAND} adm policy add-scc-to-user privileged -z default -n logging
	${OC_COMMAND} adm policy add-scc-to-user anyuid -z default -n logging
    ${OC_COMMAND} apply -f ${LOGGING_STACK}
    sleep 10
}

function check_pod() {
    set +e
    set -x
    ${OC_COMMAND} get pods | grep -viE 'Running|STATUS'
    while [ $? -eq 0 ]; do
	    sleep 5;
	    ${OC_COMMAND} get pods | grep -viE 'Running|STATUS'
    done
    set -e
    set +x
}

function configure_istio() {
    ${OC_COMMAND} apply -f ${FLUENTD_ISTIO}
}

function check_logs() {
    
    ${OC_COMMAND} -n logging port-forward $(${OC_COMMAND} -n logging get pod -l app=kibana -o jsonpath='{.items[0].metadata.name}') 5601:5601 &

    curl http://${GATEWAY_URL}//productpage
    curl http://${GATEWAY_URL}//productpage
    curl http://${GATEWAY_URL}//productpage
    curl http://${GATEWAY_URL}//productpage
    curl http://${GATEWAY_URL}//productpage
    curl http://${GATEWAY_URL}//productpage

    echo
    echo "# Go to Kibana UI (localhost:5601)"
    echo "# Click the 'Set up index patterns' in the top right"
    echo "# Use * as the index pattern, and click 'Next step'"
    echo "# Select @timestamp as the Time Filter field name, and click 'Create index pattern'"
    echo "# Click 'Discover' on the left menu, and start exploring the logs generated"
    read -p "Press enter to continue: "
}

function main() {
    banner "TC_30 Logging with Fluentd"
    echo "bookinfo" | ./bookinfo_install.sh
    sleep 10

    apply_stack
    check_pod
    configure_istio
    sleep 20

    check_logs
    banner "TC_30 passed"
}
main
