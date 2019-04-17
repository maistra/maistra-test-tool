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

import sys
import datetime
import yaml
from yaml import Loader  
from pylarion.test_run import TestRun  
from pylarion.test_record import TestRecord  
from pylarion.work_item import TestCase, Requirement  
from pylarion.document import Document
from pylarion.test_step import TestStep  




def main():
    """ This script will create and import all test data from work_item_all_data.yaml
        to Polarion new workitems (Requirements and test cases)
        This scripts was executed and we don't need to run this again. 
        To update or add new test case, you should run an update work item script.

        argument sys.argv[1]: work item data yaml file ,e.g. work_item_all_data.yaml
    """
    
    PROJECT = "MaistraIstio"
    SETUP = "Istio system is up and running on an OCP cluster. User has completed the oc login. A test namespace 'bookinfo' has been created."
    TEARDOWN = "This test will automatically clean testing configurations."

    with open(sys.argv[1]) as f:
        data = f.read()

    for case in yaml.load_all(data, Loader=Loader):
        STEP = "Run script: " + case['automation_script']
        RESULT = "Test should return message PASS ok in the end"

        # Create a Requirement
        req = Requirement.create(project_id=PROJECT,
                            title=case['name'],
                            desc=case['description'],
                            reqtype=case['testtype'],
                            subtype1=case['subtype1'],
                            severity=case['severity'])

        # Creating a specific TestCase Work Item
        tc = TestCase.create(project_id=PROJECT,
                        title=case['name'],
                        desc=case['description'],
                        caseimportance=case['importance'],
                        caselevel=case['caselevel'],
                        casecomponent="-",
                        setup=SETUP,
                        teardown=TEARDOWN,
                        automation_script=case['automation_script'], 
                        testtype=case['testtype'],
                        subtype1=case['subtype1'],   
                        caseautomation=case['automation'], 
                        caseposneg="positive")  

        # Update Requirement
        req.add_assignee("yuaxu")
        req.add_approvee("yuaxu")
        req.edit_approval("yuaxu", "approved")

        # Update TestCase  
        tc.add_linked_item(req.work_item_id, "verifies")
        tc.add_assignee("yuaxu")
        tc.add_approvee("yuaxu")
        tc.edit_approval("yuaxu", "approved")
        tc.update()

        # Add TestStep
        test1 = [STEP, RESULT]
        ts1 = TestStep()
        ts1.values = test1
        set_steps = [ts1]
        tc.set_test_steps(set_steps)

        # Approve tc
        tc.status = "approved"
        tc.update()
    
        # Approve req
        req.status = "approved"
        req.update()


if __name__ == '__main__':
    main()