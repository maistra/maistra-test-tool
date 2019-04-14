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
import subprocess as sp
import shutil
import requests
from tqdm import tqdm


class OCP(object):
    """ Instances of this class handle installing or uninstalling an OCP cluster on AWS
    
    Parameters:
        - `profile`: AWS Profile name
        - `assets`: OpenShift cluster assets directory path
        - `installer_version`: OpenShift installer version
        - `oc_version`: OpenShift oc client version
    """

    def __init__(self, profile='', assets='assets', installer_version='', oc_version=''):
        """ Initialize configuration parameters
        """
        self.profile = profile
        self.assets = assets
        self.installer_version = installer_version
        self.installer_url = 'https://github.com/openshift/installer/releases/download/v' + installer_version + '/openshift-install-linux-amd64'
        self.oc_version = oc_version
        self.oc_url = 'https://github.com/Maistra/origin/releases/download/v3.11.0+maistra-' + oc_version + '/istiooc_linux'
        #self.oc_url = 'https://mirror.openshift.com/pub/openshift-v3/clients/' + oc_version + '/linux/oc.tar.gz'


    def install(self):
        """ 
        Download the installer and deploy an OCP cluster on AWS. 
        Download the oc client and creat a soft link from kubectl to oc
        """
        # check awscli installed
        proc = sp.run(['aws', '--version'])
        if proc.returncode != 0:
            print('Please run scripts/setup.sh to install the awscli first.')
            raise RuntimeError

        os.environ['AWS_PROFILE'] = self.profile

        # download the installer
        print('Downloading the installer...')
        r = requests.get(self.installer_url, stream=True)
        chunkSize = 1024
        fileSize = int(r.headers['Content-length'])
        wrote = 0
        with open('openshift-install', 'wb') as f:
            for chunk in tqdm(r.iter_content(chunkSize), total=int(fileSize / chunkSize), unit='KB', unit_scale=True):
                wrote = wrote + len(chunk)
                f.write(chunk)
        if fileSize != 0 and wrote != fileSize:
            print('Error. Download installer not complete.')
            raise RuntimeError

        os.chmod('openshift-install', 0o775)

        # deploy the cluster
        print('Deploying the cluster...')
        os.makedirs(self.assets, mode=0o775, exist_ok=True)
        proc = sp.run(['./openshift-install', '--dir=' + self.assets, 'create', 'cluster'], check=True)

        print('Cluster deployment completed.')
        os.environ['KUBECONFIG'] = self.assets + '/auth/kubeconfig'

        print('Downloading the oc client...')
        r = requests.get(self.oc_url, stream=True)
        chunkSize = 1024
        fileSize = int(r.headers['Content-length'])
        wrote = 0

        with open('oc', 'wb') as f:
            for chunk in tqdm(r.iter_content(chunkSize), total=int(fileSize / chunkSize), unit='KB', unit_scale=True):
                wrote = wrote + len(chunk)
                f.write(chunk)
        if fileSize != 0 and wrote != fileSize:
            print('Error. Download oc client not complete.')
            raise RuntimeError
        
        """
        with open('oc.tar.gz', 'wb') as f:
            for chunk in tqdm(r.iter_content(chunkSize), total=int(fileSize / chunkSize), unit='KB', unit_scale=True):
                wrote = wrote + len(chunk)
                f.write(chunk)
        if fileSize != 0 and wrote != fileSize:
            print('Error. Download oc client not complete.')
            raise RuntimeError
        shutil.unpack_archive('oc.tar.gz')
        os.remove('oc.tar.gz')
        """

        os.chmod('oc', 0o755)
        sp.run(['sudo', 'mv', 'oc', '/usr/bin/oc'])
        sp.run(['sudo', 'ln', '-s', 'oc', '/usr/bin/kubectl'])
        
        print('Check cluster info')
        sp.run(['kubectl', 'cluster-info'])


    def uninstall(self):
        """ Destroy a cluster
        """
        os.environ['AWS_PROFILE'] = self.profile
        print('Destroying a cluster...')
        proc = sp.run(['./openshift-install', '--dir=' + self.assets, 'destroy', 'cluster', '--log-level=debug'], check=True)
        if proc.returncode == 0:
            print('Uninstall completed')
            shutil.rmtree(self.assets)
            os.remove('openshift-install')
            sp.run(['sudo', 'rm', '-f', '/usr/bin/kubectl'])
            sp.run(['sudo', 'rm', '-f', '/usr/bin/oc'])


