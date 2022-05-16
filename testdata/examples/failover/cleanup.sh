#!/bin/bash

# Copyright Red Hat, Inc.
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

# shellcheck disable=SC1091
source common.sh

oc2 delete -f import/importedserviceset.yaml
oc2 delete -f examples/destinationrule-failover.yaml
sleep 20

oc1 delete -n west-mesh-system servicemeshpeer/east-mesh
oc2 delete -n east-mesh-system servicemeshpeer/west-mesh

oc1 delete project west-mesh-system
oc2 delete project east-mesh-system
# oc1 delete project mesh1-exports
# oc2 delete project mesh2-imports

oc1 delete project bookinfo-ha
oc2 delete project bookinfo-ha

sleep 120