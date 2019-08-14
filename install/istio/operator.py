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
import re
import time
import subprocess as sp
import shutil

class Operator(object):
    """ Instances of this class wrap the project: https://github.com/Maistra/istio-operator
    istio-operator: an operator (controller) that can be used to manage the installation of an Istio control plane

    Parameter:
         
    """

    def __init__(self, operator_version="maistra-1.0"):
        sp.run(['curl', '-o', 'operator_quay.yaml', '-L', "https://raw.githubusercontent.com/Maistra/istio-operator/%s/deploy/servicemesh-operator.yaml" % operator_version])
        
        
    def mutate(self, operator_file="operator_quay.yaml"):
        imageP1 = re.compile('image:.*istio-operator-rhel8:.*')
        imageP2 = re.compile('value:.*istio-cni-rhel8:.*')

        with open('operator_quay.yaml', 'r') as f:
            lines = f.readlines()
        with open('operator_quay.yaml', 'w') as f:
            for line in lines:
                f.write(imageP1.sub("image: quay.io/maistra/istio-operator-rhel8:latest-qe", line))

        with open('operator_quay.yaml', 'r') as f:
            lines = f.readlines()
        with open('operator_quay.yaml', 'w') as f:
            for line in lines:
                f.write(imageP2.sub("value: quay.io/maistra/istio-cni-rhel8:latest-qe", line))


    def deploy_es(self, es_version="4.1"):
        # install the Elasticsearch Operator
        sp.run(['oc', 'new-project', 'openshift-logging'], stderr=sp.PIPE)
        sp.run(['oc', 'create', '-n', 'openshift-logging', '-f', "https://raw.githubusercontent.com/openshift/elasticsearch-operator/release-%s/manifests/01-service-account.yaml" % es_version])
        sp.run(['oc', 'create', '-f', "https://raw.githubusercontent.com/openshift/elasticsearch-operator/release-%s/manifests/02-role.yaml" % es_version])
        sp.run(['oc', 'create', '-f', "https://raw.githubusercontent.com/openshift/elasticsearch-operator/release-%s/manifests/03-role-bindings.yaml" % es_version])
        sp.run(['oc', 'create', '-n', 'openshift-logging', '-f', "https://raw.githubusercontent.com/openshift/elasticsearch-operator/release-%s/manifests/04-crd.yaml" % es_version])
        # curl https://raw.githubusercontent.com/openshift/elasticsearch-operator/release-4.1/manifests/05-deployment.yaml | sed 's/latest/4.1/g' | oc create -n openshift-logging -f -
        sp.run(['sleep', '10'])


    def deploy_jaeger(self, jaeger_version="v1.13.1"):
        # install the Jaeger operator as a prerequisit
        sp.run(['oc', 'new-project', 'observability'], stderr=sp.PIPE)
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/crds/jaegertracing_v1_jaeger_crd.yaml" % jaeger_version])
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/service_account.yaml" % jaeger_version])
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/role.yaml" % jaeger_version])
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/role_binding.yaml" % jaeger_version])
        
        """
        sp.run(['curl', '-o', 'jaeger_operator.yaml', '-L', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/operator.yaml" % jaeger_version])
        imageP1 = re.compile('image: jaegertracing/jaeger-operator:.*')
        with open('jaeger_operator.yaml', 'r') as f:
            lines = f.readlines()
        with open('jaeger_operator.yaml', 'w') as f:
            for line in lines:
                f.write(imageP1.sub("image: quay.io/maistra/jaeger-operator:1.13.1", line))

        sp.run(['oc', 'create', '-n', 'observability', '-f', 'jaeger_operator.yaml'])
        """
        
        sp.run(['oc', 'create', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/operator.yaml" % jaeger_version])
        # curl https://raw.githubusercontent.com/jaegertracing/jaeger-operator/v1.13.1/deploy/operator.yaml | sed 's@jaegertracing/jaeger-operator:1.13.1@quay.io/maistra/jaeger-operator:1.13.1@g' | oc create -n observability -f -
        sp.run(['sleep', '10'])


    def deploy_kiali(self, kiali_version="v1.0.5"):
        # install the Kiali operator as a prerequisit
        sp.run(['curl', '-o', 'deploy-kiali-operator.sh', '-L', 'https://git.io/getLatestKialiOperator'])
        os.chmod('deploy-kiali-operator.sh', 0o755)
        sp.call("./deploy-kiali-operator.sh %s %s %s %s %s" % ("--operator-image-version", kiali_version, "--kiali-image-version", kiali_version, "--operator-watch-namespace '**' --accessible-namespaces '**' --operator-install-kiali false"), shell=True)
        sp.run(['sleep', '10'])


    def deploy_istio(self, operator_file="operator_quay.yaml"):
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

        sp.run(['oc', 'create', '-n', 'istio-operator', '-f', operator_file])
        sp.run(['sleep', '30'])


    def uninstall(self, operator_file="operator_quay.yaml", jaeger_version="v1.13.1", kiali_version="v1.0.0"):

        sp.run(['oc', 'delete', '-n', 'istio-operator', '-f', operator_file])

        # uninstall the Jaeger Operator
        ##sp.run(['oc', 'delete', '-n', 'observability', '-f', 'jaeger_operator.yaml'])
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/operator.yaml" % jaeger_version])
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/role_binding.yaml" % jaeger_version])
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/role.yaml" % jaeger_version])
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/service_account.yaml" % jaeger_version])
        sp.run(['oc', 'delete', '-n', 'observability', '-f', "https://raw.githubusercontent.com/jaegertracing/jaeger-operator/%s/deploy/crds/jaegertracing_v1_jaeger_crd.yaml" % jaeger_version])
        sp.run(['sleep', '10'])

        # uninstall the Kiali Operator
        sp.run(['curl', '-o', 'deploy-kiali-operator.sh', '-L', 'https://git.io/getLatestKialiOperator'])
        os.chmod('deploy-kiali-operator.sh', 0o755)
        sp.call("./deploy-kiali-operator.sh %s" % "--uninstall-mode true --operator-watch-namespace '**'", shell=True)
        
        sp.run(['oc', 'delete', 'project', 'istio-operator'])
        sp.run(['oc', 'delete', 'project', 'observability'])
        sp.run(['sleep', '10'])


    def check(self):
        print("# Verify istio-operator image name: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'istio-operator', '-o', 'jsonpath="{..image}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("# Verify kiali-operator image name: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'kiali-operator', '-o', 'jsonpath="{..image}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("# Verify jaeger-operator image name: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'observability', '-o', 'jsonpath="{..image}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("# Verify istio-operator image ID: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'istio-operator', '-o', 'jsonpath="{..imageID}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("# Verify kiali-operator image ID: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'kiali-operator', '-o', 'jsonpath="{..imageID}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

        print("# Verify jaeger-operator image ID: ")
        imageIDs = sp.run(['oc', 'get', 'pods', '-n', 'observability', '-o', 'jsonpath="{..imageID}"'], stdout=sp.PIPE, universal_newlines=True)
        for line in imageIDs.stdout.split(' '):
            print(line)

   
    def add_anyuid(self, account, namespace):
        proc = sp.run(['oc', 'adm', 'policy', 'add-scc-to-user', 'anyuid', '-z', account, '-n', namespace], stdout=sp.PIPE, universal_newlines=True)
        print(proc.stdout)



class ControlPlane(object):
    """Instances of istio system ControlPlane created by istio-operator"""

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
        sp.run(['sleep', '40'])


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
        print("Waiting 30 seconds...")
        sp.run(['sleep', '30'])
