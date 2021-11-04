#!/bin/bash

export GODEBUG=x509ignoreCN=0

cd tests
go get -u github.com/jstemmer/go-junit-report
go test -timeout 3h -v 2>&1 | tee >($HOME/go/bin/go-junit-report > results.xml) test.log

echo "#Testing Completed#"
sleep 300