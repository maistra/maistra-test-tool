#!/bin/bash

VERSION=$1

echo "Install AWS CLI"
pip install --upgrade awscli
aws --version

echo "Downloading the installer..."
curl -o openshift-installer.tag.gz \
  https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${VERSION}/openshift-install-linux-${VERSION}.tar.gz
tar -xzvf openshift-installer.tag.gz
rm openshift-installer.tag.gz
mv client/openshift-install ./openshift-install
chmod 775 ./openshift-install

echo "Deploy a cluster"
export AWS_PROFILE=openshift-dev
cp ../resources/ocp-templates/install-config-ovn.yaml install-config.yaml

mkdir -p assets
./openshift-install --dir=assets create cluster

echo "Install operators"

echo "Create SMCP"

echo "Run tests"
./setup_ocp_scc_anyuid.sh
pushd ../tests
export GODEBUG=x509ignoreCN=0
go get -u github.com/jstemmer/go-junit-report
go test -timeout 3h -v 2>&1 | tee >(${GOPATH}/bin/go-junit-report > results.xml) test.log

popd
