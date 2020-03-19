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
import re
import time
import subprocess as sp
import shutil


class Operator(object):
    """ An instance of this class installs operators from OLM openshift-marketplace."""

    def __init__(self, maistra_branch="maistra-1.0", maistra_tag="latest-1.0-qe"):
        self.es_sub_channel = "4.2"
        self.jaeger_sub_channel = "stable"
        self.kiali_sub_channel = "stable"
        self.ossm_sub_channel = "1.0"
        self.namespace = "openshift-operators"
        self.maistra_branch = maistra_branch
        self.maistra_tag = maistra_tag

    # def updateTemplate(self):

    def mutate(self, cr_file="cr_mt_quay.yaml"):
        image = re.compile('tag: .*')
        with open(cr_file, 'r') as f:
            lines = f.readlines()
        with open(cr_file, 'w') as f:
            for line in lines:
                f.write(image.sub("tag: {:s}".format(self.maistra_tag), line))


    def checkRunning(self):
        proc = sp.run(['oc', 'get', 'pod', '-n', self.namespace], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)
        timeout = 240
        while ('ContainerCreating' in proc.stdout) or ('Pending' in proc.stdout):
            sp.run(['sleep', '5'])
            timeout -= 5
            if timeout < 0:
                print("\n Error: pod is not runing.")
                print(proc.stdout)
                break
            proc = sp.run(['oc', 'get', 'pod', '-n', self.namespace], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)


    def check(self):
        sp.run(['sleep', '40'])
        self.checkRunning()

        print("# Verify image name: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', self.namespace, '-o', 'jsonpath="{..image}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("# Verify image ID: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', self.namespace, '-o', 'jsonpath="{..imageID}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)


    def add_anyuid(self, account, namespace):
        proc = sp.run(['oc', 'adm', 'policy', 'add-scc-to-user', 'anyuid', '-z', account, '-n', namespace], stdout=sp.PIPE, universal_newlines=True)
        print(proc.stdout)


    def deploy_es(self):
        sp.run(['oc', 'apply', '-f', 'olm/elastic_search_subscription.yaml'])

    def deploy_jaeger(self):
        sp.run(['oc', 'apply', '-f', 'olm/jaeger_subscription.yaml'])

    def deploy_kiali(self):
        sp.run(['oc', 'apply', '-f', 'olm/kiali_subscription.yaml'])

    def deploy_istio(self):
        sp.run(['oc', 'apply', '-f', 'olm/ossm_subscription.yaml'])

    # TBD patch41 is a temporary patch method for OCP 4.1
    def patch41(self):
        proc = sp.run(['oc', 'get', 'clusterversion'], stdout=sp.PIPE, universal_newlines=True)
        if "4.1" in proc.stdout:
            sp.run(['oc', 'patch', 'csc/redhat-operators', '-n', 'openshift-marketplace', '--type', 'merge',
             '-p', r'{"spec":{"targetNamespace": "openshift-operators"}}'])
            sp.run(['oc', 'patch', 'subscription/elasticsearch-operator', '-n', self.namespace, '--type', 'merge',
             '-p', r'{"spec":{"sourceNamespace": "openshift-operators"}}'])
            sp.run(['oc', 'patch', 'subscription/jaeger-product', '-n', self.namespace, '--type', 'merge',
             '-p', r'{"spec":{"sourceNamespace": "openshift-operators"}}'])
            sp.run(['oc', 'patch', 'subscription/kiali-ossm', '-n', self.namespace, '--type', 'merge',
             '-p', r'{"spec":{"sourceNamespace": "openshift-operators"}}'])
            sp.run(['oc', 'patch', 'subscription/servicemeshoperator', '-n', self.namespace, '--type', 'merge',
             '-p', r'{"spec":{"sourceNamespace": "openshift-operators"}}'])

    def get_quay_yaml(self):
        sp.run(['curl', '-o', 'ossm_operator.yaml', '-L',
            "https://raw.githubusercontent.com/Maistra/istio-operator/{:s}/deploy/servicemesh-operator.yaml".format(self.maistra_branch)])

        imageP1 = re.compile('image:.*istio-.*-operator.*')
        imageP2 = re.compile('value:.*istio-cni-rhel8:.*')
        imageP3 = re.compile('namespace:.*istio-operator')

        with open('ossm_operator.yaml', 'r') as f:
            lines = f.readlines()
        with open('ossm_operator.yaml', 'w') as f:
            for line in lines:
                f.write(imageP1.sub("image: quay.io/maistra/istio-rhel8-operator:{:s}".format(self.maistra_tag), line))

        with open('ossm_operator.yaml', 'r') as f:
            lines = f.readlines()
        with open('ossm_operator.yaml', 'w') as f:
            for line in lines:
                f.write(imageP2.sub("value: quay.io/maistra/istio-cni-rhel8:{:s}".format(self.maistra_tag), line))

        with open('ossm_operator.yaml', 'r') as f:
            lines = f.readlines()
        with open('ossm_operator.yaml', 'w') as f:
            for line in lines:
                f.write(imageP3.sub("namespace: " + self.namespace, line))


    def deploy_quay_istio(self):
        self.get_quay_yaml()
        sp.run(['oc', 'create', '-n', self.namespace, '-f', 'ossm_operator.yaml'])


    def uninstall_quay_istio(self):
        self.get_quay_yaml()
        sp.run(['oc', 'delete', '-n', self.namespace, '-f', 'ossm_operator.yaml'])

    def uninstall(self):
        # delete subscription
        sp.run(['oc', 'delete', '-f', 'olm/ossm_subscription.yaml'])
        sp.run(['oc', 'delete', '-f', 'olm/kiali_subscription.yaml'])
        sp.run(['oc', 'delete', '-f', 'olm/jaeger_subscription.yaml'])
        sp.run(['oc', 'delete', '-f', 'olm/elastic_search_subscription.yaml'])
        sp.run(['sleep', '10'])

        # delete all CSV
        sp.run(['oc', 'delete', 'csv', '-n', self.namespace, '--all'])
        sp.run(['sleep', '30'])


class ControlPlane(object):
    """An instance of istio system ControlPlane created by istio-operator"""

    def __init__(self, name, namespace, testNamespace, nslist, smmr, smoke_sample):
        self.name = name
        self.namespace = namespace
        self.testNamespace = testNamespace
        self.nslist = nslist
        self.smmr = smmr
        self.smoke_sample = smoke_sample


    def check(self):
        
        print("# Verify istio images name: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', self.namespace, '-o', 'jsonpath="{..image}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("# Verify istio images ID: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', self.namespace, '-o', 'jsonpath="{..imageID}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("# Verify all rpms names: ")
        template = r"""'{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'"""
        podNames = sp.run(['oc', 'get', 'pods', '-n', self.namespace, '-o', 'go-template', '--template=' + template], stdout=sp.PIPE, universal_newlines=True)
        for line in podNames.stdout.split('\n'):
            if 'istio' in line:
                rpmNames = sp.run(['oc', 'rsh', '-n', self.namespace, line, 'rpm', '-q', '-a'], stdout=sp.PIPE, universal_newlines=True)
                for row in rpmNames.stdout.split('\n'):
                    if 'servicemesh' in row:
                        print(row)
                

    def install(self, cr_file=None):
        if cr_file is None:
            raise RuntimeError('Missing cr yaml file')

        sp.run(['oc', 'new-project', self.namespace], stderr=sp.PIPE)
        
        sp.run(['oc', 'apply', '-n', self.namespace, '-f', cr_file])
        print("Waiting installation complete...")
        # verify installation
        timeout = time.time() + 60 * 20
        template = r"""'{{range .status.conditions}}{{printf "%s=%s, reason=%s, message=%s\n\n" .type .status .reason .message}}{{end}}'"""
        while time.time() < timeout:
            proc = sp.run(['oc', 'get', 'ServiceMeshControlPlane/' + self.name, '-n', self.namespace, '--template=' + template], stdout=sp.PIPE, universal_newlines=True)
            if 'Installed=True' in proc.stdout:
                break

        sp.run(['sleep', '40'])


    def create_ns(self, nslist: list):
        # create namespaces
        for ns in nslist:
            sp.run(['oc', 'new-project', ns])

        sp.run(['sleep', '5'])
    
    def apply_smmr(self):
        # apply SMMR
        proc = sp.run(['oc', 'apply', '-n', self.namespace, '-f', self.smmr], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)
        print(proc.stdout)
        print(proc.stderr)
        sp.run(['sleep', '5'])


    def smoke_check(self):
        # verify installation
        print( self.namespace + " namespace pods: ")
        proc = sp.run(['oc', 'get', 'pod', '-n', self.namespace], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)
        print(proc.stdout)

        print("# Installation result: ")
        proc = sp.run(['oc', 'get', 'smcp', '-n', self.namespace, '-o', 'wide'], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)
        print(proc.stdout)

        proc = sp.run(['oc', 'get', 'smcp/' + self.name, '-n', self.namespace, '-o', "jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}'"], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)
        print(proc.stdout)

        template = r"""'{{range .status.conditions}}{{printf "%s=%s, reason=%s, message=%s\n\n" .type .status .reason .message}}{{end}}'"""
        proc = sp.run(['oc', 'get', 'ServiceMeshControlPlane/' + self.name, '-n', self.namespace, '--template=' + template], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)    
        if 'Installed=True' in proc.stdout and 'reason=InstallSuccessful' in proc.stdout:
            print(proc.stdout)
        else:
            print(proc.stdout)
            print(proc.stderr)

        print("# Install bookinfo application")
        sp.run(['oc', 'new-project', self.testNamespace])
        sp.run(['oc', 'apply', '-n', self.testNamespace, '-f', self.smoke_sample])
        print("Waiting bookinfo application deployment...")
        proc = sp.run(['oc', 'get', 'pod', '-n', self.testNamespace], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)
        timeout = 240
        while ('ContainerCreating' in proc.stdout) or ('Pending' in proc.stdout) or ('Running' not in proc.stdout) or ('2/2' not in proc.stdout):
            sp.run(['sleep', '5'])
            timeout -= 5
            if timeout < 0: 
                print("\n\n Error: bookinfo not working !!")
                break
            proc = sp.run(['oc', 'get', 'pod', '-n', self.testNamespace], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)

        print(self.testNamespace + " namespace pods: ")
        proc = sp.run(['oc', 'get', 'pod', '-n', self.testNamespace], stdout=sp.PIPE, stderr=sp.PIPE, universal_newlines=True)
        print(proc.stdout)

        imageIDs = sp.run(['oc', 'get', 'pods', '-n', self.testNamespace, '-o', 'jsonpath="{..image}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        imageIDs = sp.run(['oc', 'get', 'pods', '-n', self.testNamespace, '-o', 'jsonpath="{..imageID}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("# Uninstall bookinfo application")
        sp.run(['oc', 'delete', '-n', self.testNamespace, '-f', self.smoke_sample])
        sp.run(['sleep', '20'])


    def uninstall(self, cr_file=None):
        if cr_file is None:
            raise RuntimeError('Missing cr yaml file')

        sp.run(['oc', 'delete', '-n', self.namespace, '-f', self.smmr])
        for ns in self.nslist:
            sp.run(['oc', 'delete', 'project', ns])

        sp.run(['oc', 'delete', '-n', self.namespace, '-f', cr_file])
        sp.run(['oc', 'delete', 'project', self.namespace])
        print("Waiting 40 seconds...")
        sp.run(['sleep', '40'])
