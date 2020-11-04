#!/bin/bash

oc new-project bookinfo
oc new-project foo
oc new-project bar
oc new-project legacy

oc adm policy add-scc-to-user anyuid -z default -n bookinfo
oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n bookinfo
oc adm policy add-scc-to-user anyuid -z httpbin -n bookinfo
oc adm policy add-scc-to-user anyuid -z httpbin -n foo
oc adm policy add-scc-to-user anyuid -z httpbin -n bar
oc adm policy add-scc-to-user anyuid -z httpbin -n legacy
sleep 5

go get -u github.com/jstemmer/go-junit-report
go test -timeout 3h -v 2>&1 | tee >($HOME/go/bin/go-junit-report > results.xml) test.log

echo "#Acc Tests completed#"
sleep 300