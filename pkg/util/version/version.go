package version

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	SMCP_2_0 = ParseVersion("v2.0")
	SMCP_2_1 = ParseVersion("v2.1")
	SMCP_2_2 = ParseVersion("v2.2")
	SMCP_2_3 = ParseVersion("v2.3")
	SMCP_2_4 = ParseVersion("v2.4")

	VERSIONS = []*Version{
		&SMCP_2_0,
		&SMCP_2_1,
		&SMCP_2_2,
		&SMCP_2_3,
		&SMCP_2_4,
	}
)

type Version struct {
	Major int
	Minor int
}

func ParseVersion(version string) Version {
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}
	majorMinor := strings.Split(version, ".")
	if len(majorMinor) != 2 {
		panic(fmt.Sprintf("invalid SMCP version: %s", version))
	}
	major, err := strconv.Atoi(majorMinor[0])
	if err != nil {
		panic(fmt.Sprintf("invalid SMCP version: %s", version))
	}
	minor, err := strconv.Atoi(majorMinor[1])
	if err != nil {
		panic(fmt.Sprintf("invalid SMCP version: %s", version))
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

func (this Version) GetPreviousVersion() Version {
	var prevVersion *Version
	for _, v := range VERSIONS {
		if *v == this {
			if prevVersion == nil {
				panic(fmt.Sprintf("version %s is the first supported version", this))
			}
			return *prevVersion
		}
		prevVersion = v
	}
	panic(fmt.Sprintf("version %s not found in VERSIONS", this))
}

func (this Version) IsSupported() bool {
	for _, v := range VERSIONS {
		if *v == this {
			return true
		}
	}
	return false
}
