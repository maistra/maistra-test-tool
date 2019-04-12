#!/bin/bash

set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../../"

OC_COMMAND="oc"
BOOKINFO_FILE="testdata/bookinfo/platform/kube/bookinfo.yaml"
GATEWAY_FILE="testdata/bookinfo/networking/bookinfo-gateway.yaml"
RULE_FILE="testdata/bookinfo/networking/destination-rule-all.yaml"

function banner() {
  message="$1"
  border="$(echo ${message} | sed -e 's+.+=+g')"
  echo "${border}"
  echo "${message}"
  echo "${border}"
}

function usage() {

  cat <<EOF
  Usage: ${BASH_SOURCE[0]} [options ...]
    options:
      -t        Enabled mutual TLS
      -h        help
EOF
}

while getopts 'th' OPTION; do
  case "$OPTION" in
    t) RULE_FILE="testdata/bookinfo/networking/destination-rule-all-mtls.yaml" ;;
    h|*) usage; exit 2 ;;
  esac
done
shift $((OPTIND-1))


function deploy_bookinfo() {

  	echo -n "namespace ? [bookinfo] "
  	read PROJECT
	
	if [[ -z ${PROJECT} ]];then
  		PROJECT="bookinfo"
	fi

	echo "using NAMESPACE=${PROJECT}"

	set +e
	${OC_COMMAND} new-project ${PROJECT}
	${OC_COMMAND} project ${PROJECT}
	set -e
	
	# grant priviledged permission
	${OC_COMMAND} adm policy add-scc-to-user privileged -z default -n ${PROJECT}
	${OC_COMMAND} adm policy add-scc-to-user anyuid -z default -n ${PROJECT}

	# enable automatic sidecar injection
	#${OC_COMMAND} label namespace ${PROJECT} istio-injection=enabled

	# install bookinfo
	${OC_COMMAND} apply -f ${BOOKINFO_FILE}
}

function check_bookinfo() {
	# verify all pods are Running
	set +e
	${OC_COMMAND} get pods -n ${PROJECT} | grep -viE 'Running|STATUS'
	while [ $? -eq 0 ]; do
		sleep 5;
		${OC_COMMAND} get pods -n ${PROJECT} | grep -viE 'Running|STATUS'
	done
	set -e

	${OC_COMMAND} get pods -n ${PROJECT}
}

function deploy_gateway() {
	# create gateway
	${OC_COMMAND} apply -f ${GATEWAY_FILE}
	# check gateway
	${OC_COMMAND} get gateway
}

function deploy_rule() {
	# apply destination rules
	${OC_COMMAND} apply -f ${RULE_FILE}
}

function deploy() {
	banner "Deploy bookinfo"
	deploy_bookinfo
	check_bookinfo
	banner "Configure Gateway"
	deploy_gateway
	banner "Configure Destination Rule"
	deploy_rule
}

deploy
echo "Application installation successful"