#!/bin/bash

set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../../"

PROJECT=istio-operator
OC_COMMAND="oc"
# community
OPERATOR_FILE="${BASE_DIR}/samples/maistra/istio_community_operator_template.yaml"

function usage() {

  cat <<EOF
  Usage: ${BASH_SOURCE[0]} [options ...]
    options:
      -p        Use product operator template
      -h        help
EOF
}

function banner() {
  message="$1"
  border="$(echo ${message} | sed -e 's+.+=+g')"
  echo "${border}"
  echo "${message}"
  echo "${border}"
}

while getopts 'ph' OPTION; do
  case "$OPTION" in
    p) OPERATOR_FILE="${BASE_DIR}/samples/maistra/istio_product_operator_template.yaml" ;;
    h|*) usage; exit 2 ;;
  esac
done
shift $((OPTIND-1))

function delete_istio() {
	${OC_COMMAND} delete -n $PROJECT Installation istio-installation
}

function check_istio() {
	set +e
	set -x
	while [ -n "$(${OC_COMMAND} get pods -n istio-system)" ]; do
		sleep 10;
	done

	${OC_COMMAND} get projects | grep istio-system
	while [ $? -eq 0 ]; do
		sleep 5;
		${OC_COMMAND} get projects | grep istio-system
	done
	set -e
	set +x
}

function delete_operator() {
	${OC_COMMAND} project $PROJECT
	${OC_COMMAND} process -f ${OPERATOR_FILE} | ${OC_COMMAND} delete -f -
}

function check_operator() {
	set +e
	set -x
	while [ -n "$(${OC_COMMAND} get pods -n $PROJECT)" ]; do
		sleep 5;
	done
	set -e
	set +x
}

function delete_project() {
	${OC_COMMAND} delete project $PROJECT
}

function uninstall() {
	banner "Delete Istio System"
	delete_istio
	check_istio
	banner "Delete Istio Operator"
	delete_operator
	check_operator
	banner "Delete Istio Operator Project"
	delete_project
}

uninstall
echo "Istio cleanup successful"
