// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
 * This package includes an entrypoint of running tests.
 * main_test.go is calling Golang testing.Main framework and it reloads all packages from pkg directory.
 * All test cases are mapped in the test_cases.go file.
 */

package tests

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	fmt.Println("*****************************************************************************************************************")
	fmt.Println("*****************************************************************************************************************")
	fmt.Println("ERROR: You can no longer run tests this way")
	fmt.Println()
	fmt.Println("The new way to run the tests is to execute the following command in the project's root directory (not in tests/):")
	fmt.Println()
	fmt.Println("    make test                  # to run all tests")
	fmt.Println()
	fmt.Println("    make test TestSomething    # to run a specific test")
	fmt.Println()
	fmt.Println("*****************************************************************************************************************")
	fmt.Println("*****************************************************************************************************************")
	os.Exit(1)
}
