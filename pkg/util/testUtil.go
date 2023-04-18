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

package util

import (
	"fmt"
)

func IsWithinPercentage(count int, total int, rate float64, tolerance float64) bool {
	minimum := int((rate - tolerance) * float64(total))
	maximum := int((rate + tolerance) * float64(total))
	return count >= minimum && count <= maximum
}

func GenerateStrings(prefix string, count int) []string {
	arr := make([]string, count)
	for i := 0; i < count; i++ {
		arr[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return arr
}
