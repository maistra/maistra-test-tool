#!/bin/bash

SMCP="istio-system"
SMMR="configFiles/smmrTest.yaml"
TESTNS="bookinfo"

oc new-project bookinfo
oc new-project foo
oc new-project bar
oc new-project legacy
sleep 5

kubectl apply -n ${SMCP} -f ${SMMR}
sleep 5

oc adm policy add-scc-to-user anyuid -z default -n ${TESTNS}
oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n ${TESTNS}
sleep 5