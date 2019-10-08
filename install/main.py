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
import argparse

from ocp.ocp import OCP
from puller import Puller
from istio.operator import Operator, ControlPlane


class Moitt(object):

    def __init__(self):
        self.profile = None
        self.pullsec = None
        self.crfile = None
        self.install = False
        self.uninstall = False
        self.component = None
        self.assets = os.getcwd() + '/assets'
        
    def envParse(self):
        if 'AWS_PROFILE' in os.environ:
            self.profile = os.environ['AWS_PROFILE']
        if 'PULL_SEC' in os.environ:
            self.pullsec = os.environ['PULL_SEC']
        if 'CR_FILE' in os.environ:
            self.crfile = os.environ['CR_FILE']

    
    def argParse(self):
        parser = argparse.ArgumentParser(description='Select an operation and component(s)')
        group = parser.add_mutually_exclusive_group(required=True)
        group.add_argument('-i', '--install', help='install operation', action='store_true')
        group.add_argument('-u', '--uninstall', help='uninstall operation', action='store_true')
        parser.add_argument('-c', '--component', type=str, choices=['ocp', 'registry-puller', 'istio'], help='Specify Component from ocp, registry-puller, istio')
        args = parser.parse_args()
        self.install = args.install
        self.uninstall = args.uninstall
        self.component = args.component        
      

def main():
    moitt = Moitt()
    moitt.envParse()
    moitt.argParse()
    
    if not moitt.profile:
        raise KeyError("Missing AWS_PROFILE environment variable")
    if not moitt.pullsec:
        raise KeyError("Missing PULL_SEC environment variable")

    ocp = OCP(profile=moitt.profile, assets=moitt.assets)
    os.environ['KUBECONFIG'] = moitt.assets + '/auth/kubeconfig'
    
    if moitt.component == 'ocp':
        if moitt.install:
            # Install ocp cluster
            ocp.install()
            
            # Read kubeadmin password
            with open(moitt.assets + '/auth/kubeadmin-password') as f:
                pw = f.read()
            ocp.login('kubeadmin', pw)
            
            # Create testing users, qe1 and qe2
            ocp.create_users()
            ocp.logout()

        elif moitt.uninstall:
            ocp.uninstall()
    
    if moitt.component == 'registry-puller':
        puller = Puller(secret_file=moitt.pullsec)

        # Read kubeadmin password
        with open(moitt.assets + '/auth/kubeadmin-password') as f:
            pw = f.read() 
        ocp.login('kubeadmin', pw)
        if moitt.install:
            puller.build()
            puller.execute()

        ocp.logout()
    
    if moitt.component == 'istio':
        operator = Operator(operator_version="maistra-1.0")
        operator.mutate()

        nslist = ['bookinfo', 'foo', 'bar', 'legacy']
        smmr = os.getcwd() + '/member-roll.yaml'
        sample = os.getcwd() + '/bookinfo.yaml'
        cp = ControlPlane("basic-install", "service-mesh-1", "bookinfo", nslist, smmr, sample)
        if moitt.install:
            # deploy operators
            # Read kubeadmin password
            with open(moitt.assets + '/auth/kubeadmin-password') as f:
                pw = f.read() 
            ocp.login('kubeadmin', pw)
            operator.deploy_es()
            operator.deploy_jaeger()
            operator.deploy_kiali()
            operator.deploy_istio()
            operator.check()
            ocp.logout()

            # deploy controlplane
            ocp.login('qe1', os.getenv('QE1_PWD', 'qe1pw'))
            cp.install(cr_file=moitt.crfile)
            cp.create_ns(cp.nslist)
            cp.apply_smmr()
            cp.smoke_check()
            cp.check()
            ocp.logout()

            # add scc anyuid
            with open(moitt.assets + '/auth/kubeadmin-password') as f:
                pw = f.read() 
            ocp.login('kubeadmin', pw)
            operator.add_anyuid("bookinfo-productpage", "bookinfo")
            operator.add_anyuid("bookinfo-reviews", "bookinfo")
            operator.add_anyuid("bookinfo-ratings-v2", "bookinfo")
            operator.add_anyuid("default", "bookinfo")
            ocp.logout()

        elif moitt.uninstall:
            # uninstall controlplane
            ocp.login('qe1', os.getenv('QE1_PWD', 'qe1pw'))
            cp.uninstall(cr_file=moitt.crfile)
            ocp.logout()

            # uninstall operators
            # Read kubeadmin password

            with open(moitt.assets + '/auth/kubeadmin-password') as f:
                pw = f.read() 
            ocp.login('kubeadmin', pw)
            operator.uninstall()
            ocp.logout()


   
if __name__ == '__main__':
    main()
