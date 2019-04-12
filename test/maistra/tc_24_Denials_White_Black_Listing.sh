#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
DENY_POLICY="testdata/policy/mixer-rule-deny-label.yaml"
CHECK_VERSION_RULE="testdata/policy/checkversion-rule.yaml"
CHECKIP_RULE="testdata/policy/checkip-rule.yaml"
APPVERSION_INSTANCE="testdata/policy/appversion-instance.yaml"
SOURCEIP_INSTANCE="testdata/policy/sourceip-instance.yaml"
WHITELIST="testdata/policy/whitelist-handler.yaml"
WHITELISTIP="testdata/policy/whitelistip-handler.yaml"
VS_ALL_V1="testdata/bookinfo/networking/virtual-service-all-v1.yaml"
VS_REVIEW_V2_V3="testdata/bookinfo/networking/virtual-service-reviews-jason-v2-v3.yaml"

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
	${OC_COMMAND} delete -f ${WHITELISTIP}
	${OC_COMMAND} delete -f ${SOURCEIP_INSTANCE}
	${OC_COMMAND} delete -f ${CHECKIP_RULE}
	${OC_COMMAND} delete -f ${DENY_POLICY}
	${OC_COMMAND} delete -f ${CHECK_VERSION_RULE}
	${OC_COMMAND} delete -f ${APPVERSION_INSTANCE}
	${OC_COMMAND} delete -f ${WHITELIST}
	${OC_COMMAND} delete -f ${VS_ALL_V1}
	${OC_COMMAND} delete --ignore-not-found=true -f ${VS_REVIEW_V2_V3}
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


function apply_deny_policy() {
	${OC_COMMAND} apply -f ${VS_ALL_V1}
	${OC_COMMAND} apply -f ${VS_REVIEW_V2_V3}
	echo "# Explicitly deny access to v3 reviews"
	${OC_COMMAND} apply -f ${DENY_POLICY}
}

function check_deny_1() {
	set +e
	set -x
	${OC_COMMAND} get rule | grep denyreviewsv3
	while [ $? -ne 0 ]; do
		sleep 5;
		${OC_COMMAND} get rule | grep denyreviewsv3
	done
	set -e
	set +x
}

function check_point() {
	echo "# sleep 20 seconds..."
	sleep 20
	echo
	echo "http://${GATEWAY_URL}/productpage"
	read -p "Press enter to continue: "
}

function check_deny_2() {
	set +e
	set -x
	${OC_COMMAND} get rule | grep denyreviewsv3
	while [ $? -eq 0 ]; do
		sleep 5;
		${OC_COMMAND} get rule | grep denyreviewsv3
	done
	set -e
	set +x
}

function apply_whitelist() {
	${OC_COMMAND} apply -f ${VS_ALL_V1}
	${OC_COMMAND} apply -f ${VS_REVIEW_V2_V3}

	${OC_COMMAND} apply -f ${WHITELIST}
	${OC_COMMAND} apply -f ${APPVERSION_INSTANCE}
	${OC_COMMAND} apply -f ${CHECK_VERSION_RULE}
}


function apply_ip_whitelist() {
	${OC_COMMAND} apply -f ${WHITELISTIP}
	${OC_COMMAND} apply -f ${SOURCEIP_INSTANCE}
	${OC_COMMAND} apply -f ${CHECKIP_RULE}
}

function main() {
	banner "TC_24 Denials White Black Listing"
	echo "bookinfo" | ./bookinfo_install.sh
	
	enable_policy_check
	apply_deny_policy
	check_deny_1
	echo "# Check productpage. Without login review shows ratings service is unavailable. Login user jason and see review black stars..."
	check_point
	cleanup
	sleep 10

	echo "bookinfo" | ./bookinfo_install.sh
	set -e
	check_deny_2
	banner "whitelists/blacklists"
	apply_whitelist
	echo "# Check productpage. Without login review shows ratings service is unavailable. Login user jason and see review black stars..."
	check_point

	banner "IP-based whitelists/blacklists"
	apply_ip_whitelist
	sleep 20
	echo "Check productpage. Get expected error: PERMISSION_DENIED:staticversion.istio-system:<your mesh source ip> is not whitelisted"
	check_point
	
	
	banner "TC_24 passed"
}

main
