#!/bin/bash

TESTNS="bookinfo"

oc adm policy add-scc-to-user anyuid -z default -n ${TESTNS}
oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n ${TESTNS}
oc adm policy add-scc-to-user anyuid -z httpbin -n ${TESTNS}
sleep 5