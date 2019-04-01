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
from istio.operator import Operator


class Moitt(object):

    def __init__(self):
        self.profile = None
        self.pullsec = None
        self.operatorfile = None
        self.crfile = None
        self.install = False
        self.uninstall = False
        self.component = None
        
    def envParse(self):
        if 'AWS_PROFILE' in os.environ:
            self.profile = os.environ['AWS_PROFILE']
        if 'PULL_SEC' in os.environ:
            self.pullsec = os.environ['PULL_SEC']
        if 'OPERATOR_FILE' in os.environ:
            self.operatorfile = os.environ['OPERATOR_FILE']
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
    
    if moitt.component == 'ocp':
        ocp = OCP(profile=moitt.profile)
        if moitt.install:
            ocp.install()
        elif moitt.uninstall:
            ocp.uninstall()
    
    if moitt.component == 'registry-puller':
        puller = Puller(secret_file=moitt.pullsec)
        if moitt.install:
            puller.build()
            puller.execute()
    
    if moitt.component == 'istio':
        operator = Operator()
        if moitt.install:
            operator.deploy(operator_file=moitt.operatorfile)
            operator.install(cr_file=moitt.crfile)
        elif moitt.uninstall:
            operator.uninstall(cr_file=moitt.crfile)


   
if __name__ == '__main__':
    main()
