#!/bin/bash

set -e

DIR=$(cd $(dirname $0); pwd -P)
OC_COMMAND="oc"

INGRESS_HOST="$(${OC_COMMAND} -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"

while getopts 'h:' OPTION; do
  case "$OPTION" in
    h) INGRESS_HOST="${OPTARG}" ;;
  esac
done
shift $((OPTIND-1))

INGRESS_PORT="$(${OC_COMMAND} -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].port}')"
SECURE_INGRESS_PORT="$(${OC_COMMAND} -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="https")].port}')"
GATEWAY_URL="${INGRESS_HOST}:${INGRESS_PORT}"


code=$(curl -o /dev/null -s -w "%{http_code}\n" ${GATEWAY_URL}/productpage)
if [ $code -ne 200 ]; then
	echo "http code: ${code}  bookinfo is not healthy..."
	exit 1;
else
	echo "bookinfo is healthy."
fi




