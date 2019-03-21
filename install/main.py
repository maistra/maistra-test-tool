#!/usr/bin/env python
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


def main():
    parser = argparse.ArgumentParser(description='Select an operation and component(s)')
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('-i', '--install', help='install operation', action='store_true')
    group.add_argument('-u', '--uninstall', help='uninstall operation', action='store_true')
    parser.add_argument('-p', '--profile', type=str, required=True, help='AWS profile name')
    parser.add_argument('-c', '--component', nargs='+', type=str, required=True, choices=['ocp', 'registry-puller', 'istio', 'bookinfo', 'all'], default='all' , help='Specify Component(s) from ocp, registry-puller, istio, bookinfo and all')
    
    args = parser.parse_args()
    
    if 'ocp' in args.component or 'all' in args.component:
        ocp = OCP(profile=args.profile)
        if args.install:
            ocp.install()
        elif args.uninstall:
            ocp.uninstall()
    


   
if __name__ == '__main__':
    main()
