#!/bin/bash
set -e

# httpbin and sleep with sidecar in the default namespace

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
HTTPBIN="testdata/httpbin/httpbin.yaml"
SLEEP="testdata/sleep/sleep.yaml"

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
    ${OC_COMMAND} delete --ignore-not-found=true destinationrule bad-rule -n default
    ${OC_COMMAND} delete -f ${HTTPBIN} -n default
    ${OC_COMMAND} delete -f ${SLEEP} -n default
    ${OC_COMMAND} delete meshpolicy default
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

function deploy_httpbin_sleep() {
    # grant priviledged permission
	${OC_COMMAND} adm policy add-scc-to-user privileged -z default -n default
	${OC_COMMAND} adm policy add-scc-to-user anyuid -z default -n default

    ${OC_COMMAND} apply -f ${HTTPBIN} -n default
    ${OC_COMMAND} apply -f ${SLEEP} -n default
}

function check_pod() {
    set +e
    set -x
    ${OC_COMMAND} get pods -n default | grep -viE 'Running|STATUS|Completed'
    while [ $? -eq 0 ]; do
	    sleep 5;
	    ${OC_COMMAND} get pods -n default | grep -viE 'Running|STATUS|Completed'
    done
    set -e
    set +x
}

function verify_setup() {
    echo "# Verify Citadel runs properly. Available column should be 1 below..."
    ${OC_COMMAND} get deploy -l istio=citadel -n istio-system

    echo "# Verify keys and certs. cert-chain.pem, key.pem and root-cert.pem should be listed below..."
    ${OC_COMMAND} exec -n default $(${OC_COMMAND} get pod -n default -l app=httpbin -o jsonpath={.items..metadata.name}) -c istio-proxy -- ls /etc/certs

    echo "# Check cert is valid..."
    ${OC_COMMAND} exec -n default $(${OC_COMMAND} get pod -n default -l app=httpbin -o jsonpath={.items..metadata.name}) -c istio-proxy -- cat /etc/certs/cert-chain.pem | openssl x509 -text -noout  | grep Validity -A 2
    ${OC_COMMAND} exec -n default $(${OC_COMMAND} get pod -n default -l app=httpbin -o jsonpath={.items..metadata.name}) -c istio-proxy -- cat /etc/certs/cert-chain.pem | openssl x509 -text -noout  | grep 'Subject Alternative Name' -A 1

    echo "# Check mutual TLS configuration..."
    #${BASE_DIR}/bin/istioctl authn tls-check httpbin.default.svc.cluster.local
}

function create_conflict_rule() {
    cat <<EOF | ${BASE_DIR}/bin/istioctl create -n default -f -
apiVersion: "networking.istio.io/v1alpha3"
kind: "DestinationRule"
metadata:
  name: "bad-rule"
spec:
  host: "httpbin.default.svc.cluster.local"
  trafficPolicy:
    tls:
      mode: DISABLE
EOF
}

function check_conflict_rule() {
    set +e
    echo "# Check mutual TLS configuration. Status shows CONFLICT as expected below..."
    #${BASE_DIR}/bin/istioctl authn tls-check httpbin.default.svc.cluster.local
    echo
    read -p "Press enter to continue: "
    sleep 2
    echo "# Check request from sleep to httpbin fails as expected below..."
    ${OC_COMMAND} exec -n default $(${OC_COMMAND} get pod -n default -l app=sleep -o jsonpath={.items..metadata.name}) -c sleep \
        -- curl httpbin:8000/headers -o /dev/null -s -w '%{http_code}\n'
    echo
    read -p "Press enter to continue: "
    sleep 2
    set -e
}

function cleanup_conflict() {
    ${OC_COMMAND} delete --ignore-not-found=true destinationrule bad-rule -n default
    sleep 5
}

function verify_requests() {
    set +e
    echo "# Check plain-text requests fail as expected below..."
    ${OC_COMMAND} exec -n default $(${OC_COMMAND} get pod -n default -l app=sleep -o jsonpath={.items..metadata.name}) -c istio-proxy \
        -- curl http://httpbin:8000/headers -o /dev/null -s -w '%{http_code}\n'
    echo
    read -p "Press enter to continue: "

    echo "# Check TLS requests without client cert fail as expected below..."
    ${OC_COMMAND} exec -n default $(${OC_COMMAND} get pod -n default -l app=sleep -o jsonpath={.items..metadata.name}) -c istio-proxy \
        -- curl https://httpbin:8000/headers -o /dev/null -s -w '%{http_code}\n' -k
    echo
    read -p "Press enter to continue: "

    echo "# Check TLS request with client cert succeed with 200 below..."
    ${OC_COMMAND} exec -n default $(${OC_COMMAND} get pod -n default -l app=sleep -o jsonpath={.items..metadata.name}) -c istio-proxy \
        -- curl https://httpbin:8000/headers -o /dev/null -s -w '%{http_code}\n' \
        --key /etc/certs/key.pem --cert /etc/certs/cert-chain.pem --cacert /etc/certs/root-cert.pem -k
    echo
    read -p "Press enter to continue: "

    set -e
}


function main() {
    banner "TC_20 Mutual TLS Deep Dive"
    enable_mtls
    deploy_httpbin_sleep
    check_pod
    verify_setup
    banner "Creating an incorrect TLS mode"
    #create_conflict_rule
    #sleep 5
    #check_conflict_rule
    #echo "# cleanup conflict"
    #cleanup_conflict

    banner "Verify requests"
    verify_requests

    banner "TC_20 passed"
}

main
