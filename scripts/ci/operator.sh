#!/bin/bash
# shellcheck disable=SC2119

##### subscription
CATALOG_SOURCE="${CATALOG_SOURCE:-redhat-operators}"
SUBSCRIPTION_NAMES=("jaeger-product" "kiali-ossm" "servicemeshoperator")

createSubscription() {
  local name=$1

  echo "Create Subscription resource for $name"
  echo
  kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: "${name}"
  namespace: openshift-operators
spec:
  channel: stable
  installPlanApproval: Automatic
  name: "${name}"
  source: "${CATALOG_SOURCE}"
  sourceNamespace: openshift-marketplace
EOF
}

create_subscription() {
  echo "Install any operator by creating a Subscription resource."
  echo
  for name in "${SUBSCRIPTION_NAMES[@]}"; do createSubscription "$name"; done
}

delete_subscription() {
  echo "Remove any operator deployed using a Subscription resource."
  echo

  echo "Delete Subscription resource"
  for name in "${SUBSCRIPTION_NAMES[@]}"; do kubectl delete subscription "$name" -n openshift-operators; done
}

create_other_operators() {
  echo "Install the Jaeger and Kiali operators. You need these operators to deploy a ServiceMeshControlPlane with Jaeger tracing and Kiali enabled."
  echo

  createSubscription jaeger-product
  createSubscription kiali-ossm
}

get_clusterserviceversion() {
  local ns=${ns:-openshift-operators}
  echo "Get clusterserviceversion in ${ns}"
  echo
  for name in "${SUBSCRIPTION_NAMES[@]}"; do kubectl get subscription "${name}" -n "${ns}" -o yaml | grep currentCSV; done
}

delete_clusterserviceversion() {
  local ns=${ns:-openshift-operators}
  echo "Delete all clusterserviceversion in ${ns}"
  echo
  kubectl delete clusterserviceversion -n "${ns}" --all
}

delete_cni() {
  local ns=${ns:-openshift-operators}
  echo "Delete configmap"
  kubectl delete cm -l app.kubernetes.io/component=istio_cni -n "${ns}" --ignore-not-found

  echo "Delete istio cni DaemonSet(s)"
  kubectl delete ds -l app.kubernetes.io/component=istio_cni -n "${ns}" --ignore-not-found
}

get_operator() {
  echo "Get the Operator Pod via $(hl "kubectl get")."
  echo

  local ns=${ns:-openshift-operators}
  kubectl -n "$ns" get pods -l name=istio-operator "${args[*]}"
}

watch_operator() {
  echo "Watch the Operator Pod via $(hl "kubectl get -w") or $(hl "watch kubectl get")."
  echo

  local ns=${ns:-openshift-operators}
  kubectl -n "$ns" get pods -l name=istio-operator "${args[*]}"
}

edit_operator() {
  echo "Edit the Operator deployment"
  echo

  local ns=${ns:-openshift-operators}
  echo "Edit istio-operator Deployment with kubectl edit"
  kubectl -n "$ns" edit "deploy/istio-operator"
}

logs_operator() {
  echo "Display the Operator logs"
  echo

  local ns=${ns:-openshift-operators}
  kubectl -n "$ns" logs deploy/istio-operator "${args[*]}"
}

restart_operator() {
  echo "Restart the operator."
  echo

  local ns=${ns:-openshift-operators}
  echo "Restart operator by deleting the Pod"
  kubectl -n "$ns" delete pod -l name=istio-operator --force --grace-period 0
}

# we can remove this function in testing 2.4
wait_operator() {
  echo "Temporary workaround for running operator 2.3.x"
  echo

  echo -n "Waiting for Jaeger operator deployment to be created..."
  while ! kubectl get deployment -n openshift-operators -o name | grep jaeger >& /dev/null ; do echo -n '.'; sleep 1; done
  echo "done."
  jaeger_deployment="$(kubectl get deployment -n openshift-operators -o name | grep jaeger)"

  echo -n "Waiting for Kiali operator deployment to be created..."
  while ! kubectl get deployment -n openshift-operators -o name | grep kiali >& /dev/null ; do echo -n '.'; sleep 1; done
  echo "done."
  kiali_deployment="$(kubectl get deployment -n openshift-operators -o name | grep kiali)"

  echo -n "Waiting for Service Mesh operator deployment to be created..."
  while ! kubectl get deployment -n openshift-operators -o name | grep istio >& /dev/null ; do echo -n '.'; sleep 1; done
  echo "done."
  servicemesh_deployment="$(kubectl get deployment -n openshift-operators -o name | grep istio)"

  echo "Waiting for CRDs to be established."
  for crd in servicemeshcontrolplanes.maistra.io servicemeshmemberrolls.maistra.io jaegers.jaegertracing.io
  do
    echo -n "Waiting for CRD [${crd}]..."
    while ! kubectl get crd "${crd}" >& /dev/null ; do echo -n '.'; sleep 1; done
    kubectl wait --for condition=established crd/"${crd}"
  done

  echo "Waiting for operator deployments to start..."
  for op in "${servicemesh_deployment}" "${jaeger_deployment}"
  do
    echo -n "Waiting for ${op} to be ready..."
    readyReplicas="0"
    while [ "$?" != "0" -o "$readyReplicas" == "0" ]
    do
      sleep 1
      echo -n '.'
      readyReplicas="$(kubectl get ${op} -n openshift-operators -o jsonpath='{.status.readyReplicas}' 2> /dev/null)"
    done
    echo "done."
  done

  echo "Wait for the servicemesh validating webhook to be created."
  while [ "$(kubectl get validatingwebhookconfigurations -o name | grep servicemesh)" == "" ]; do echo -n '.'; sleep 5; done
  echo "done."

  echo "Wait for the servicemesh mutating webhook to be created."
  while [ "$(kubectl get mutatingwebhookconfigurations -o name | grep servicemesh)" == "" ]; do echo -n '.'; sleep 5; done
  echo "done."

  echo "Wait for the smcp.mutation.maistra.io webhook to be created."
  while [ "$(kubectl get mutatingwebhookconfigurations -o name | grep smcp.mutation.maistra.io)" == "" ]; do echo -n '.'; sleep 5; done
  echo "done."
}

for arg in "$@"
do
  case "$arg" in
    create) create_subscription;;
    delete) delete_subscription;;
    get) get_operator;;
    watch) watch_operator;;
    logs) logs_operator;;
    edit) edit_operator;;
    restart) restart_operator;;
    wait) wait_operator;;
    get_csv) get_clusterserviceversion;;
    delete_csv) delete_clusterserviceversion;;
    delete_cni) delete_cni;;

    "") echo "Missing parameter...";;
    *) echo "Unknown parameter $@";;
  esac
done
