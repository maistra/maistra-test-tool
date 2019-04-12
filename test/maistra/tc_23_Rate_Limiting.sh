#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
RATE_LIMIT="testdata/policy/mixer-rule-productpage-ratelimit.yaml"
RATE_LIMIT_CONDITIONAL="testdata/policy/mixer-rule-conditional.yaml"
VS_ALL_V1="testdata/bookinfo/networking/virtual-service-all-v1.yaml"

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
    ${OC_COMMAND} delete -f ${RATE_LIMIT}
    ${OC_COMMAND} delete -f ${VS_ALL_V1}
    rm -f /tmp/mesh.yaml
    echo "bookinfo" | ./bookinfo_uninstall.sh
}
trap cleanup EXIT

function enable_policy_check() {
  echo "# Verify disablePolicyChecks should be false"
  ${OC_COMMAND} -n istio-system get cm istio -o jsonpath="{@.data.mesh}" | grep disablePolicyChecks

  echo "# Enabling Policy Enforcement"
  ${OC_COMMAND} -n istio-system get cm istio -o jsonpath="{@.data.mesh}" | \
    sed -e "s/disablePolicyChecks: true/disablePolicyChecks: false/" > /tmp/mesh.yaml
  
  ${OC_COMMAND} -n istio-system create cm istio -o yaml --dry-run --from-file=mesh=/tmp/mesh.yaml | kubectl replace -f -
  echo "# Verify disablePolicyChecks should be false"
  ${OC_COMMAND} -n istio-system get cm istio -o jsonpath="{@.data.mesh}" | grep disablePolicyChecks
}


function apply_rate_limit() {
    ${OC_COMMAND} apply -f ${VS_ALL_V1}
    ${OC_COMMAND} apply -f ${RATE_LIMIT}
    echo "# Verify memquota handler was created"
    ${OC_COMMAND} -n istio-system get memquota handler -o yaml
    echo "# Verify quota instance was created"
    ${OC_COMMAND} -n istio-system get quotas requestcount -o yaml
    echo "# Verify quota rule was created"
    ${OC_COMMAND} -n istio-system get rules quota -o yaml
    echo "# Verify quotaspec was created"
    ${OC_COMMAND} -n istio-system get QuotaSpec request-count -o yaml
    echo "# Verify quotaspecbinding was created"
    ${OC_COMMAND} -n istio-system get QuotaSpecBinding request-count -o yaml
}

function check_quota() {
    echo "# sleep 50 seconds..."
    sleep 50
    echo
    echo "http://${GATEWAY_URL}/productpage"
    read -p "Press enter to continue: "
}

function apply_rate_limit_2() {
    ${OC_COMMAND} apply -f ${RATE_LIMIT_CONDITIONAL}
}


function main() {
    banner "TC_23 Rate Limiting"
    echo "bookinfo" | ./bookinfo_install.sh
    
    enable_policy_check
    apply_rate_limit
    echo "# Check productpage. It permits 2 requests every 5 seconds. Refresh productpage and see 'Quota is exhausted' as expected..."  
    check_quota
    banner "Conditional rate limits"
    apply_rate_limit_2
    echo "# Check productpage. Without login, it permits 2 requests every 5 seconds. Refresh productpage and see 'Quota is exhausted' as expected. Login user jason should not see quota limit message..."
    check_quota
    
    banner "TC_23 passed"
}

main
