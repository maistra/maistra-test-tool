#!/bin/bash

go get -u github.com/jstemmer/go-junit-report
go install github.com/jstemmer/go-junit-report

if [ -z "${OCP_CRED_PSW}" ]
then
  oc login --token=${OCP_TOKEN} --server=${OCP_API_URL} --insecure-skip-tls-verify=true
else
  oc login -u ${OCP_CRED_USR} -p ${OCP_CRED_PSW} --server=${OCP_API_URL} --insecure-skip-tls-verify=true
fi
go test -timeout 3h -run ${TEST_CASE} -v 2>&1 | tee >(go-junit-report > results.xml) test.log

echo "#Testing Completed#"
sleep 90
