// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
}

func ParseVersion(version string) Version {
	// This func can handle smcp versions and ocp versions. Example: smcp version v2.1.0 and ocp version 4.10.0
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}

	majorMinor := strings.Split(version, ".")

	if len(majorMinor) < 2 {
		panic(fmt.Sprintf("invalid version: %s", version))
	}

	major, err := strconv.Atoi(majorMinor[0])
	if err != nil {
		panic(fmt.Sprintf("invalid version: %s", version))
	}

	minor, err := strconv.Atoi(majorMinor[1])
	if err != nil {
		panic(fmt.Sprintf("invalid version: %s", version))
	}

	return Version{Major: major, Minor: minor}
}

func (this Version) Equals(that Version) bool {
	return this.Major == that.Major && this.Minor == that.Minor
}

func (this Version) GreaterThan(that Version) bool {
	return !this.LessThanOrEqual(that)
}

func (this Version) GreaterThanOrEqual(that Version) bool {
	return !this.LessThan(that)
}

func (this Version) LessThan(that Version) bool {
	if this.Major < that.Major {
		return true
	} else if this.Major > that.Major {
		return false
	} else { // this.Major == that.Major
		return this.Minor < that.Minor
	}
}

func (this Version) LessThanOrEqual(that Version) bool {
	return this.LessThan(that) || this.Equals(that)
}

func (this Version) String() string {
	return fmt.Sprintf("v%d.%d", this.Major, this.Minor)
}
