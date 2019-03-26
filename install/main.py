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


def main():
    parser = argparse.ArgumentParser(description='Select an operation and component(s)')
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('-i', '--install', help='install operation', action='store_true')
    group.add_argument('-u', '--uninstall', help='uninstall operation', action='store_true')
    parser.add_argument('-p', '--profile', type=str, help='AWS profile name. Alternatively, export AWS_PROFILE environment variable')
    parser.add_argument('-s', '--pullsec', type=str, help='Istio private registry pull secret.yaml file path (This is not the OpenShift pull secret). Alternatively, export PULL_SEC environment variable')
    parser.add_argument('-c', '--component', type=str, choices=['ocp', 'registry-puller', 'istio'], help='Specify Component from ocp, registry-puller, istio')
    
    args = parser.parse_args()
    arg_profile = args.profile
    arg_pullsec = args.pullsec
    arg_component = args.component

    if arg_profile is None and os.environ['AWS_PROFILE'] is not None:
        arg_profile = os.environ['AWS_PROFILE']
    elif arg_profile is None and os.environ['AWS_PROFILE'] is None:
        raise RuntimeError('Missing -p argument or missing AWS_PROFILE environment variable')

    if arg_pullsec is None and os.environ['PULL_SEC'] is not None:
        arg_pullsec = os.environ['PULL_SEC']
    elif arg_pullsec is None and os.environ['PULL_SEC'] is None:
        raise RuntimeError('Missing -s argument or missing PULL_SEC environment variable')
            
    
    if 'ocp' in args.component:
        ocp = OCP(profile=arg_profile)
        if args.install:
            ocp.install()
        elif args.uninstall:
            ocp.uninstall()
    
    if 'registry-puller' in args.component:
        puller = Puller(secret_file=arg_pullsec)
        if args.install:
            puller.build()
            puller.execute()


   
if __name__ == '__main__':
    main()
