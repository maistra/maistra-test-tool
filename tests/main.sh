#!/bin/bash

TESTNS="bookinfo"

oc adm policy add-scc-to-user anyuid -z default -n ${TESTNS}
oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n ${TESTNS}
oc adm policy add-scc-to-user anyuid -z httpbin -n ${TESTNS}
sleep 5

go get -u github.com/jstemmer/go-junit-report
go test -timeout 3h -v 2>&1 | tee >($HOME/go/bin/go-junit-report > results.xml) test.log

echo "#Acc Tests completed#"
sleep 300