#!/bin/bash

oc delete namespace devex istio-operator istio-system bookinfo logging foo bar
oc delete csr istio-sidecar-injector.istio-system
oc get crd | grep istio | awk '{print $1}' | xargs oc delete crd
oc get mutatingwebhookconfigurations | grep istio | awk '{print $1}' | xargs oc delete mutatingwebhookconfigurations
oc get validatingwebhookconfiguration | grep istio | awk '{print $1}' | xargs oc delete validatingwebhookconfiguration
oc get clusterroles | grep istio | awk '{print $1}' | xargs oc delete clusterroles
oc get clusterroles | grep kiali | awk '{print $1}' | xargs oc delete clusterroles
oc get clusterrolebindings | grep istio | awk '{print $1}' | xargs oc delete clusterrolebindings
oc get clusterrolebindings | grep kiali | awk '{print $1}' | xargs oc delete clusterrolebindings

