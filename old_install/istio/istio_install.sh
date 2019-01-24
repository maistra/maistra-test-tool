#!/bin/bash

set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../../"

OC_COMMAND="oc"
PROJECT="istio-operator"
OPENSHIFT_ISTIO_VERSION="0.5.0"
OPENSHIFT_ANSIBLE_GIT_REPO_URL="https://github.com/maistra/openshift-ansible"
ISTIO_RELEASE_VERSION="1.0.4"

# local
OPENSHIFT_ISTIO_MASTER_PUBLIC_URL="https://127.0.0.1:8443"
# Brew images
REPO=""
# community
OPERATOR_FILE="${BASE_DIR}/samples/maistra/istio_community_operator_template.yaml"
# tls not enabled
CR_FILE="${BASE_DIR}/samples/maistra/cr_template.yaml"

function usage() {

  cat <<EOF
  Usage: ${BASH_SOURCE[0]} [options ...]
    options:
      -r        Use remote OCP Cluster
      -s        Use Staging images registry
      -p        Use product operator template
      -t        Global mutual TLS enabled
      -v        Maistra Openshift Istio version (Default: 0.5.0) for Installation
      -w        Upstream Istio Samples Release Version (Default: 1.0.4) only for samples yaml files
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

while getopts 'rs:ptv:w:h' OPTION; do
  case "$OPTION" in
    r)
      echo -n "Openshift Istio Master public URL ? " 
      read OPENSHIFT_ISTIO_MASTER_PUBLIC_URL
      ;;
    s) REPO="${OPTARG}" ;;
    p) OPERATOR_FILE="${BASE_DIR}/samples/maistra/istio_product_operator_template.yaml" ;;
    t) CR_FILE="${BASE_DIR}/samples/maistra/cr_tls_template.yaml" ;;
    v) OPENSHIFT_ISTIO_VERSION="${OPTARG}" ;;
    w) ISTIO_RELEASE_VERSION="${OPTARG}" ;;
    h|*) usage; exit 2 ;;
  esac
done
shift $((OPTIND-1))

echo "OPENSHIFT_ISTIO_VERSION: ${OPENSHIFT_ISTIO_VERSION}"
echo "ISTIO_RELEASE_VERSION: ${ISTIO_RELEASE_VERSION}"

OC_RELEASE="https://github.com/Maistra/origin/releases/download/v3.11.0%2Bmaistra-${OPENSHIFT_ISTIO_VERSION}/istiooc_linux"
ISTIO_RELEASE_URL="https://github.com/istio/istio/releases/download/${ISTIO_RELEASE_VERSION}/istio-${ISTIO_RELEASE_VERSION}-linux.tar.gz"

echo "OC_RELEASE: ${OC_RELEASE}"
echo "ISTIO_RELEASE_URL: ${ISTIO_RELEASE_URL}"


function get_istiooc_linux() {
  wget ${OC_RELEASE}
  sudo mv -f istiooc_linux /usr/bin/${OC_COMMAND}
  sudo cp -f /usr/bin/${OC_COMMAND} /usr/bin/kubectl
  sudo chmod +x /usr/bin/${OC_COMMAND}
  sudo chmod +x /usr/bin/kubectl
  ${OC_COMMAND} version
}

function get_istio_release_samples() {
  wget ${ISTIO_RELEASE_URL}
  tar xzf istio-${ISTIO_RELEASE_VERSION}-linux.tar.gz
  cp -rf istio-${ISTIO_RELEASE_VERSION}/bin ${BASE_DIR}
  cp -rf istio-${ISTIO_RELEASE_VERSION}/samples ${BASE_DIR}
  rm -f istio-${ISTIO_RELEASE_VERSION}-linux.tar.gz
  pushd istio-${ISTIO_RELEASE_VERSION} && rm -rf *
  popd && rmdir istio-${ISTIO_RELEASE_VERSION}
}

function update_samples() {
  pushd ${BASE_DIR}
  # bookinfo
  cp -f maistra_samples/bookinfo/networking/bookinfo-db.yaml samples/bookinfo/networking/bookinfo-db.yaml
  cp -f maistra_samples/bookinfo/networking/bookinfo-ratings-mysql-service-entry.yaml samples/bookinfo/networking/bookinfo-ratings-mysql-service-entry.yaml
  cp -f maistra_samples/bookinfo/networking/bookinfo-ratings-v2-mysql.yaml samples/bookinfo/networking/bookinfo-ratings-v2-mysql.yaml
  cp -f maistra_samples/bookinfo/networking/bookinfo-ratings-v2.yaml samples/bookinfo/networking/bookinfo-ratings-v2.yaml
  cp -f maistra_samples/bookinfo/networking/virtual-service-ratings-test-delay-2.yaml samples/bookinfo/networking/virtual-service-ratings-test-delay-2.yaml
  cp -f maistra_samples/bookinfo/networking/virtual-service-reviews-timeout.yaml samples/bookinfo/networking/virtual-service-reviews-timeout.yaml
  # certs
  cp -rf maistra_samples/certs/bookinfo.com samples/certs/
  cp -rf maistra_samples/certs/httpbin.example.com samples/certs/
  # egress
  cp -rf maistra_samples/egress samples/
  # httpbin
  cp -f maistra_samples/httpbin/httpbin-v1.yaml samples/httpbin/httpbin-v1.yaml
  cp -f maistra_samples/httpbin/httpbin-v2.yaml samples/httpbin/httpbin-v2.yaml
  cp -rf maistra_samples/httpbin/networking samples/httpbin/
  # maistra
  cp -rf maistra_samples/maistra samples/
  # policy
  cp -rf maistra_samples/policy samples/
  # sleep
  cp -f maistra_samples/sleep/sleep-ip-range.yaml samples/sleep/sleep-ip-range.yaml
  # Must keep sleep-v2. sleep-v2 is using a different curl image from the one in sleep
  cp -f maistra_samples/sleep/sleep-v2.yaml samples/sleep/sleep-v2.yaml
  # telemetry
  cp -rf maistra_samples/telemetry samples/
  # security
  cp -rf maistra_samples/security samples/
  popd
}

function start_cluster() {
  # set up a single instance OpenShift cluster
  ${OC_COMMAND} cluster up --enable="*,-istio"
}

function login_admin() {
  RESULT=$(${OC_COMMAND} login -u system:admin)
}

function login_remote() {
  echo "## Get oc login command from remote OCP cluster console"
  read -p "oc login command: " login
  eval $login
}

function start_operator() {
  ${OC_COMMAND} new-project "${PROJECT}"
  ${OC_COMMAND} project "${PROJECT}"

  if [ ! -f "${OPERATOR_FILE}" ]; then 
    echo "ERROR: Operator Template file does not exist"
    echo "${OPERATOR_FILE}"
    exit
  fi

  # install Istio Operator
  ${OC_COMMAND} new-app -f ${OPERATOR_FILE} \
  -p OPENSHIFT_ISTIO_MASTER_PUBLIC_URL=$OPENSHIFT_ISTIO_MASTER_PUBLIC_URL \
  -p OPENSHIFT_ISTIO_PREFIX=${REPO}openshift-istio-tech-preview/ \
  -p OPENSHIFT_ISTIO_VERSION=${OPENSHIFT_ISTIO_VERSION}
}

function deploy_istio() {
  if [ ! -f "${CR_FILE}" ]; then 
    echo "ERROR: Custom Resource file does not exist"
    echo "${CR_FILE}"
    exit
  fi

  # install istio-system, kiali and jaeger
  ${OC_COMMAND} new-app -f ${CR_FILE} \
    -p OPENSHIFT_ISTIO_PREFIX=${REPO}openshift-istio-tech-preview/ \
    -p OPENSHIFT_JAEGER_PREFIX=${REPO}distributed-tracing-tech-preview/ \
    -p OPENSHIFT_KIALI_PREFIX=${REPO}openshift-istio-tech-preview/ \
    -p OPENSHIFT_ISTIO_VERSION=${OPENSHIFT_ISTIO_VERSION}
}

function check_istio() {
  set +e
  set -x
  ${OC_COMMAND} get pods -n istio-system | grep Completed
  while [ $? -ne 0 ]; do
	  sleep 10;
	  ${OC_COMMAND} get pods -n istio-system | grep Completed
  done

  # check operator log
  ${OC_COMMAND} logs -n ${PROJECT} $(${OC_COMMAND} -n ${PROJECT} get pods -l name=istio-operator --output=jsonpath={.items..metadata.name})

  ${OC_COMMAND} get pods -n istio-system | grep -viE 'Completed|Running|STATUS'
  while [ $? -eq 0 ]; do
	  sleep 5;
	  ${OC_COMMAND} get pods -n istio-system | grep -viE 'Completed|Running|STATUS'
  done

  set -e
  set +x

  ${OC_COMMAND} get pods -n istio-system
}

function deploy() {
  usage

  banner "Download istiooc_linux"
  get_istiooc_linux
  banner "Download Istio release samples"
  get_istio_release_samples
  update_samples

  if [ "${OPENSHIFT_ISTIO_MASTER_PUBLIC_URL}" = "https://127.0.0.1:8443" ]; then
    banner "Start Local istiooc Cluster Up"
    start_cluster
    login_admin
  else
    banner "Login Remote OCP Cluster"
    login_remote
  fi

  banner "Start Istio Operator"
  start_operator
  banner "Deploy Istio System"
  deploy_istio
  check_istio
}

deploy
echo "Istio intsallation successful"