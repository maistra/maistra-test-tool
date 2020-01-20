#!/bin/bash

# Copyright 2019 Red Hat, Inc.
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

cd $HOME/workspace/run_acc_tests_all/

python3 -m venv .env
source .env/bin/activate
pip install -r requirements.txt

# Run Install
cd install

export AWS_PROFILE=openshift-dev
export PULL_SEC=$HOME/daily/secret.yaml
export CR_FILE=$HOME/daily/cr/cr_mt_quay.yaml

OCP_LOGNAME=$(echo OCP-$(date +"%Y%m%d_%H%M%S")'.log')

export KUBECONFIG=$HOME/workspace/run_acc_tests_all/install/assets/auth/kubeconfig

python main.py -u -c ocp -v $OCP_VERSION | tee $WORKSPACE/report/$OCP_LOGNAME

