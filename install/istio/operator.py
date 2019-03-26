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


    def deploy(self):
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

        sp.run(['oc', 'new-project', 'istio-operator'], stderr=sp.PIPE)
        sp.run(['oc', 'new-project', 'istio-system'], stderr=sp.PIPE)

        # pyyaml update operator image
        sp.run(['oc', 'apply', '-n', 'istio-operator', '-f', 'istio-operator/deploy/'])
        # TBD
        sp.run(['sleep', '60'])

    def install(self, cr_file=None):
        if cr_file is None:
            cr_file = 'istio-operator/deploy/examples/istio_v1alpha3_controlplane_cr_basic.yaml'
        
        sp.run(['oc', 'apply', '-n', 'istio-system', '-f', cr_file])

        # verify installation
        timeout = time.time() + 60 * 5
        template = '\'{{range .status.conditions}}{{printf "%s=%s, reason=%s, " .type .status .reason}}{{end}}\''
        while time.time() < timeout:
            proc = sp.run(['oc', 'get', 'controlplane/basic-install', '-n', 'istio-system', '--template=' + template], stdout=sp.PIPE, universal_newlines=True)
            if 'InstallSuccessful' in proc.stdout:
                break
        
        proc = sp.run(['oc', 'get', 'controlplane/basic-install', '-n', 'istio-system', '--template=' + template], stdout=sp.PIPE, universal_newlines=True)    
        if 'InstallSuccessful' in proc.stdout:
            print(proc.stdout)
        else:
            print('Error: ' + proc.stdout)
        

        
    def uninstall(self, cr_file=None):
        if cr_file is None:
            cr_file = 'istio-operator/deploy/examples/istio_v1alpha3_controlplane_cr_basic.yaml'

        sp.run(['oc', 'delete', '-n', 'istio-system', '-f', cr_file])
        

    