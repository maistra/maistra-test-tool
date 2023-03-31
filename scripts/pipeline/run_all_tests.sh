#!/bin/bash

go get -u github.com/jstemmer/go-junit-report

if [ -z "${OCP_CRED_PSW}" ]
then
  oc login --token=${OCP_TOKEN} --server=${OCP_API_URL} --insecure-skip-tls-verify=true
else
  oc login -u ${OCP_CRED_USR} -p ${OCP_CRED_PSW} --server=${OCP_API_URL} --insecure-skip-tls-verify=true
fi

../runtests.sh

echo "#Testing Completed#"
sleep 90
