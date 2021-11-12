#!/bin/bash

go get -u github.com/jstemmer/go-junit-report

oc login -u ${OCP_CRED_USR} -p ${OCP_CRED_PSW} --server=${OCP_API_URL} --insecure-skip-tls-verify=true
go test -timeout 3h -v 2>&1 | tee >($HOME/go/bin/go-junit-report > results.xml) test.log

echo "#Testing Completed#"
sleep 300