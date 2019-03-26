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
from pathlib import Path


class Puller(object):
    """ Instances of this class wrap the project: https://github.com/knrc/registry-puller
    registry-puller:
    a webhook which monitors the creation of ServiceAccounts within kubernetes namespaces 
    and modifies those ServiceAccounts so they include an additional image pull secret. 
    This is of use when trying to deploy containers referencing private repos.

    Parameter:
        - `secret_file`: an image pull secret file from a private registry.  
    """

    def __init__(self, secret_file=None):

        if not Path(secret_file).is_file():
            raise ValueError('Missing a secret file')
        
        self.secret_file = secret_file
        self.repo = 'git@github.com:knrc/registry-puller.git'
        self.savedPath = os.getcwd()

    def build(self):
        try:
            os.chdir('registry-puller')
            sp.run(['git', 'pull'])
        except OSError:
            os.chdir(self.savedPath)
            sp.run(['git', 'clone', self.repo])
            os.chdir('registry-puller')
        except WindowsError:
            os.chdir(self.savedPath)
            sp.run(['git', 'clone', self.repo])
            os.chdir('registry-puller')

        sp.run(['make'])
        os.chdir(self.savedPath)
        
    def execute(self):
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

        sp.run(['oc', 'new-project', 'registry-puller'])
        sp.run(['oc', 'create', 'configmap', '-n', 'registry-puller', 'registry-secret', '--from-file=' + self.secret_file])
        os.chdir(self.savedPath)
        os.chdir('registry-puller')
        sp.run(['oc', 'create', '-f', 'registry-puller-4.0.yaml'])

        timeout = time.time() + 60 * 5
        while time.time() < timeout:
            proc = sp.run(['oc', 'get', 'pod', '-n', 'registry-puller'], stdout=sp.PIPE, universal_newlines=True)
            if 'Running' in proc.stdout:
                break
        
        proc = sp.run(['oc', 'get', 'pod', '-n', 'registry-puller'], stdout=sp.PIPE, universal_newlines=True)
        if 'Running' in proc.stdout:
            print('registry-puller pod is running')
        else:
            print('Error: registry-puller is not running\n' + proc.stdout)
        os.chdir(self.savedPath)

