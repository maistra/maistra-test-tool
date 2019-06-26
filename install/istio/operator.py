#!/usr/bin/env python3
# -*- coding: utf-8 -*-

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

import os
import time
import subprocess as sp
import shutil

class Operator(object):
    """ Instances of this class wrap the project: https://github.com/Maistra/istio-operator
    istio-operator: an operator (controller) that can be used to manage the installation of an Istio control plane

    Parameter:
         
    """

    def __init__(self):
        pass
        

    def deploy_jaeger(self, jaeger_version="v1.12.1"):
        # install the Jaeger operator as a prerequisit
        sp.run(['oc', 'new-project', 'observability'], stderr=sp.PIPE)
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/crds/jaegertracing_v1_jaeger_crd.yaml" % jaeger_version])
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/service_account.yaml" % jaeger_version])
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/role.yaml" % jaeger_version])
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/role_binding.yaml" % jaeger_version])
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/operator.yaml" % jaeger_version])
        sp.run(['sleep', '10'])


    def deploy(self, operator_file=None):
        # check environment variable KUBECONFIG
        try:
            os.environ['KUBECONFIG']
        except KeyError:
            raise KeyError('Missing environment variable KUBECONFIG')
        # check if oc is installed
        proc = sp.run(['oc', 'version'])
        if proc.returncode != 0:
            raise RuntimeError('Missing oc client')
        # check os login
        proc = sp.run(['oc', 'status'])
        if proc.returncode != 0:
            raise RuntimeError('Login not completed')
        if operator_file is None:
            raise RuntimeError('Missing operator.yaml file')
        
        sp.run(['oc', 'new-project', 'istio-operator'], stderr=sp.PIPE)
        sp.run(['oc', 'new-project', 'istio-system'], stderr=sp.PIPE)

        sp.run(['oc', 'apply', '-n', 'istio-operator', '-f', operator_file])


    def check(self):
        # verify installation
        print("\n# istio-system namespace pods: ")
        sp.run(['oc', 'get', 'pod', '-n', 'istio-system'])
        print("\n# bookinfo namespace pods: ")
        sp.run(['oc', 'get', 'pod', '-n', 'bookinfo'])

        print("\n# istio-operator log installation result: ")
        template = r"""'{{range .status.conditions}}{{printf "%s=%s, reason=%s, message=%s\n\n" .type .status .reason .message}}{{end}}'"""
        proc = sp.run(['oc', 'get', 'ServiceMeshControlPlane/basic-install', '-n', 'istio-system', '--template=' + template], stdout=sp.PIPE, universal_newlines=True)    
        if 'Installed=True' in proc.stdout and 'reason=InstallSuccessful' in proc.stdout:
            print(proc.stdout)
        else:
            print('Error: ' + proc.stdout)

        print("\n# verify all images name: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'istio-operator', '-o', 'jsonpath="{..image}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'istio-system', '-o', 'jsonpath="{..image}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'bookinfo', '-o', 'jsonpath="{..image}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("\n# verify all images ID: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'istio-operator', '-o', 'jsonpath="{..imageID}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'istio-system', '-o', 'jsonpath="{..imageID}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'bookinfo', '-o', 'jsonpath="{..imageID}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)        
        
        print("\n# verify all rpms names: ")
        template = r"""'{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'"""
        podNames = sp.run(['oc', 'get', 'pods', '-n', 'istio-system', '-o', 'go-template', '--template=' + template], stdout=sp.PIPE, universal_newlines=True)
        for line in podNames.stdout.split('\n'):
            if 'istio' in line:
                rpmNames = sp.run(['oc', 'rsh', '-n', 'istio-system', line, 'rpm', '-q', '-a'], stdout=sp.PIPE, universal_newlines=True)
                for row in rpmNames.stdout.split('\n'):
                    if 'servicemesh' in row:
                        print(row)
                

    def install(self, cr_file=None):
        if cr_file is None:
            raise RuntimeError('Missing cr yaml file')
        
        sp.run(['oc', 'apply', '-n', 'istio-system', '-f', cr_file])
        print("\n# Waiting installation complete...")
        # verify installation
        timeout = time.time() + 60 * 20
        template = r"""'{{range .status.conditions}}{{printf "%s=%s, reason=%s, message=%s\n\n" .type .status .reason .message}}{{end}}'"""
        while time.time() < timeout:
            proc = sp.run(['oc', 'get', 'ServiceMeshControlPlane/basic-install', '-n', 'istio-system', '--template=' + template], stdout=sp.PIPE, universal_newlines=True)
            if 'Installed=True' in proc.stdout:
                break

        # verify bookinfo deployment
        print("\n# Installing bookinfo Application")
        sp.run(['./bookinfo_install.sh'], input="bookinfo\n", cwd="../test/maistra", stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)
        print("\n# Waiting installation complete...")

        # verify images ID and rpm names
        self.check()

        # uninstall bookinfo
        print("\n# Uninstalling bookinfo Application")
        sp.run(['./bookinfo_uninstall.sh'], input="bookinfo\n", cwd="../test/maistra", stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)
        
        
    def uninstall(self, operator_file=None, cr_file=None, jaeger_version="v1.12.1"):
        if operator_file is None:
            raise RuntimeError('Missing operator.yaml file')
        if cr_file is None:
            raise RuntimeError('Missing cr yaml file')

        sp.run(['oc', 'delete', '-n', 'istio-system', '-f', cr_file])
        sp.run(['sleep', '10'])
        sp.run(['oc', 'delete', '-n', 'istio-operator', '-f', operator_file])

        # uninstall the Jaeger Operator
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/operator.yaml" % jaeger_version])
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/role_binding.yaml" % jaeger_version])
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/role.yaml" % jaeger_version])
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/service_account.yaml" % jaeger_version])
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/crds/jaegertracing_v1_jaeger_crd.yaml" % jaeger_version])
        sp.run(['sleep', '10'])
