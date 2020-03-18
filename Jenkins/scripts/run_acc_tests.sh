#!/bin/bash

# Copyright 2020 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# Prepare environment
python3 -m venv .env
source .env/bin/activate

pip install -r requirements.txt

unset GOOS && unset GOARCH && CGO_ENABLED=1 go get "github.com/knrc/registry-puller/cmd"
unset GOOS && unset GOARCH && CGO_ENABLED=1 go get -u github.com/jstemmer/go-junit-report

# Prepare config files
mkdir -p $WORKSPACE/configs

echo "${INSTALL_CONFIG_TEMPLATE}" | CLUSTER_NAME=${CLUSTER_NAME} envsubst | AWS_OCP_SECRET=${AWS_OCP_SECRET} envsubst > $WORKSPACE/configs/install-config.yaml
echo "${QUAY_MAISTRA_PULL_SEC_TEMPLATE}" | CONFIGJSON=${CONFIGJSON} envsubst > $WORKSPACE/configs/secret.yaml
echo "${SMCP_CR}" | TAG=${TAG} envsubst > $WORKSPACE/configs/cr.yaml

# Parse parameters
export AWS_PROFILE=openshift-dev
export INSTALL_CONFIG=$WORKSPACE/configs/install-config.yaml
export PULL_SEC=$WORKSPACE/configs/secret.yaml
export CR_FILE=$WORKSPACE/configs/cr.yaml
export OCP_VERSION=${OCP_VERSION}
export TAG=${TAG}
export OCP_SERVER=https://api.${CLUSTER_NAME}.devcluster.openshift.com:6443
export QE1_PWD=qe1pw
export QE2_PWD=qe2pw

# Log names
OCP_LOGNAME=$(echo OCP-$(date +"%Y%m%d_%H%M%S")'.log')
PULLER_LOGNAME=$(echo Puller-$(date +"%Y%m%d_%H%M%S")'.log')
SMCP_LOGNAME=$(echo SMCP-$(date +"%Y%m%d_%H%M%S")'.log')
JUNIT_FILE1=$(echo JUnit1-$(date +"%Y%m%d_%H%M%S")'.xml')
TEST_LOGNAME1=$(echo Test1-$(date +"%Y%m%d_%H%M%S")'.log')
JUNIT_FILE2=$(echo JUnit2-$(date +"%Y%m%d_%H%M%S")'.xml')
TEST_LOGNAME2=$(echo Test2-$(date +"%Y%m%d_%H%M%S")'.log')
mkdir -p $WORKSPACE/report
rm -f $WORKSPACE/report/*

# Create OCP4.x AWS Cluster
pushd install
mkdir -p ./assets
            
export KUBECONFIG=$WORKSPACE/install/assets/auth/kubeconfig
mv -f ${INSTALL_CONFIG} $WORKSPACE/install/assets/install-config.yaml

if [ -f $KUBECONFIG ]; then
	echo "OCP4 daily cluster exists." >> $WORKSPACE/report/$OCP_LOGNAME
else
    echo "Creating OCP4 AWS cluster...Wait 50mins..."
	python main.py -i -c ocp -v $OCP_VERSION | tee $WORKSPACE/report/$OCP_LOGNAME
fi

cp -f $WORKSPACE/install/assets/.openshift_install.log $WORKSPACE/report/

# Install Registry Puller
python main.py -i -c registry-puller | tee $WORKSPACE/report/$PULLER_LOGNAME
rm -f $WORKSPACE/configs/secret.yaml

# Deploy Service Mesh
python main.py -i -c istio -t ${TAG} -q | tee $WORKSPACE/report/$SMCP_LOGNAME
popd

# Run ACC Tests
pushd test/maistra

# login OCP
oc login -u qe1 -p ${QE1_PWD} --server=${OCP_SERVER} --insecure-skip-tls-verify=true

go test -run "\d+$" -timeout 3h -v 2>&1 | tee >(~/go/bin/go-junit-report > $WORKSPACE/report/$JUNIT_FILE1) $WORKSPACE/report/$TEST_LOGNAME1

# Patch mtls enabled to true
oc patch -n service-mesh-1 smcp/basic-install --type merge -p '{"spec":{"istio":{"global":{"controlPlaneSecurityEnabled":true,"mtls":{"enabled":true}}}}}'

# Run mtls tests
go test -run "\d+mtls$" -timeout 3h -v 2>&1 | tee >(~/go/bin/go-junit-report > $WORKSPACE/report/$JUNIT_FILE2) $WORKSPACE/report/$TEST_LOGNAME2

# Patch mtls enabled to false
oc patch -n service-mesh-1 smcp/basic-install --type merge -p '{"spec":{"istio":{"global":{"controlPlaneSecurityEnabled":false,"mtls":{"enabled":false}}}}}'

popd
