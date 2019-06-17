#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
DENY_POLICY="testdata/policy/mixer-rule-deny-label.yaml"
HTTPBIN="testdata/httpbin/httpbin-all.yaml"
KEYVAL="testdata/httpbin/policy/keyval.yaml"
KEYVAL_TEMPLATE="testdata/httpbin/policy/keyval-template.yaml"

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
	${OC_COMMAND} delete rule/keyval -n istio-system
  sleep 30
  ${OC_COMMAND} delete handler/keyval instance/keyval adapter/keyval template/keyval -n istio-system
  ${OC_COMMAND} delete service keyval -n istio-system
  ${OC_COMMAND} delete deployment keyval -n istio-system
	${OC_COMMAND} delete -f ${HTTPBIN}
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
	echo "# Explicitly deny access to v3 reviews"
	${OC_COMMAND} apply -f ${DENY_POLICY}
}

function deploy_httpbin() {
  # grant privileged permission
	${OC_COMMAND} adm policy add-scc-to-user privileged -z default -n default
	${OC_COMMAND} adm policy add-scc-to-user anyuid -z default -n default

  ${OC_COMMAND} apply -f ${HTTPBIN}
}

function set_key_val() {
    ${OC_COMMAND} run keyval --image=gcr.io/istio-testing/keyval:release-1.1 --namespace istio-system --port 9070 --expose
    ${OC_COMMAND} apply -f ${KEYVAL_TEMPLATE} --namespace istio-system
    ${OC_COMMAND} apply -f ${KEYVAL} --namespace istio-system
}

function demo_adapter_rule() {
    ${OC_COMMAND} apply --namespace istio-system -f - <<EOF
apiVersion: config.istio.io/v1alpha2
kind: handler
metadata:
  name: keyval
  namespace: istio-system
spec:
  adapter: keyval
  connection:
    address: keyval:9070
  params:
    table:
      jason: admin
EOF

    ${OC_COMMAND} apply --namespace istio-system  -f - <<EOF
apiVersion: config.istio.io/v1alpha2
kind: instance
metadata:
  name: keyval
  namespace: istio-system
spec:
  template: keyval
  params:
    key: request.headers["user"] | ""
EOF
}

function redirect_to_teapot() {
    ${OC_COMMAND} apply --namespace istio-system -f - <<EOF
apiVersion: config.istio.io/v1alpha2
kind: rule
metadata:
  name: keyval
  namespace: istio-system
spec:
  match: source.labels["istio"] == "ingressgateway"
  actions:
  - handler: keyval.istio-system
    instances: [ keyval ]
  requestHeaderOperations:
  - name: :path
    values: [ '"/status/418"' ]
EOF
}

function enable_user_groups() {
    ${OC_COMMAND} apply --namespace istio-system -f - <<EOF
apiVersion: config.istio.io/v1alpha2
kind: rule
metadata:
  name: keyval
  namespace: istio-system
spec:
  actions:
  - handler: keyval.istio-system
    instances: [ keyval ]
    name: x
  requestHeaderOperations:
  - name: user-group
    values: [ x.output.value ]
EOF
}

function main() {
	banner "TC_31 Control Headers and Routing"

	enable_policy_check

	deploy_httpbin
  set_key_val

  echo "# Creates a rule for adapter"
  demo_adapter_rule
  sleep 10

	echo "# Ensures the httpbin service is accessible through the ingress gateway"
	set -x
    curl http://$INGRESS_HOST:$INGRESS_PORT/headers
    set +x
	read -p "Press enter to continue: "

    enable_user_groups
    sleep 30

    echo "# Verify that user:jason has  \"User-Group\": \"admin\" header"
    set -x
    curl -Huser:jason http://$INGRESS_HOST:$INGRESS_PORT/headers
    set +x
    read -p "Press enter to continue: "


    echo "# Redirects to another virtual service route (status, which in this case returns 418 teapot)"
    redirect_to_teapot
    sleep 30
    set -x
    curl -v -Huser:jason http://$INGRESS_HOST:$INGRESS_PORT/headers
    set +x
    read -p "Press enter to continue: "

    banner "TC_32 passed"
}

main
