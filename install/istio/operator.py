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

        self.repo = 'git@github.com:Maistra/istio-operator.git'
        self.savedPath = os.getcwd()
        try:
            os.chdir('istio-operator')
            sp.run(['git', 'pull'])
        except OSError:
            os.chdir(self.savedPath)
            sp.run(['git', 'clone', self.repo])
            os.chdir('istio-operator')
        except WindowsError:
            os.chdir(self.savedPath)
            sp.run(['git', 'clone', self.repo])
            os.chdir('istio-operator')
        os.chdir(self.savedPath)


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
        if operator_file is not None:
            shutil.copy2(operator_file, 'istio-operator/deploy/operator.yaml')
        
        sp.run(['oc', 'new-project', 'istio-operator'], stderr=sp.PIPE)
        sp.run(['oc', 'new-project', 'istio-system'], stderr=sp.PIPE)

        sp.run(['oc', 'apply', '-n', 'istio-operator', '-f', 'istio-operator/deploy/'])


    def install(self, cr_file=None):
        if cr_file is None:
            cr_file = 'istio-operator/deploy/examples/istio_v1alpha3_controlplane_cr_basic.yaml'
        
        sp.run(['oc', 'apply', '-n', 'istio-system', '-f', cr_file])

        # verify installation
        timeout = time.time() + 60 * 20
        template = r"""'{{range .status.conditions}}{{printf "%s=%s, reason=%s, message=%s\n\n" .type .status .reason .message}}{{end}}'"""
        while time.time() < timeout:
            sp.run(['oc', 'get', 'pod', '-n', 'istio-system'])
            proc = sp.run(['oc', 'get', 'controlplane/basic-install', '-n', 'istio-system', '--template=' + template], stdout=sp.PIPE, universal_newlines=True)
            if 'Installed=True' in proc.stdout:
                break
        
        proc = sp.run(['oc', 'get', 'controlplane/basic-install', '-n', 'istio-system', '--template=' + template], stdout=sp.PIPE, universal_newlines=True)    
        if 'Installed=True' in proc.stdout and 'reason=InstallSuccessful' in proc.stdout:
            print(proc.stdout)
        else:
            print('Error: ' + proc.stdout)
        

        
    def uninstall(self, cr_file=None):
        if cr_file is None:
            cr_file = 'istio-operator/deploy/examples/istio_v1alpha3_controlplane_cr_basic.yaml'

        sp.run(['oc', 'delete', '-n', 'istio-system', '-f', cr_file])
        sp.run(['sleep', '10'])
        

    