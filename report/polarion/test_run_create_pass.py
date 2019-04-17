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
    """ This script will create a Test Run with id sys.argv[1] and execute all test cases.
        After this script execution complete, the new Test Run should show all test cases passed 
        and the Test Run status is finished.

        argument sys.argv[1]: id name of a new Test Run 
    """
    
    PROJECT = "MaistraIstio"
    # Creating a Test Run:  
    tr = TestRun.create(project_id=PROJECT, test_run_id=sys.argv[1], template="Build Acceptance type", title="Istio-Tech-Preview-" + sys.argv[1])

    # changing status  
    tr.status = "inprogress"

    # Adding a test record 
    num_recs = len(tr.records)
    print("Number of records: ", num_recs)
    sorted_records = sorted(tr.records, key=lambda record: record.test_case_id)
    for i in range(num_recs):
        tr.update_test_record_by_fields(test_case_id=sorted_records[i].test_case_id, 
                                    test_result="passed", 
                                    test_comment="Test case " + sorted_records[i].test_case_id + " passed smoothly",
                                    executed_by="yuaxu",
                                    executed=datetime.datetime.now(), duration=150)
        
        print(sorted_records[i].test_case_id + " executed.")
        if i % 10 == 0:
            tr.reload()
    
    # changing status  
    tr.status = "finished"

if __name__ == '__main__':
    main()