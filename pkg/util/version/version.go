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

	OCP_4_9  = ParseOCPVersion("4.9.0")
	OCP_4_10 = ParseOCPVersion("4.10.0")
	OCP_4_11 = ParseOCPVersion("4.11.0")
	OCP_4_12 = ParseOCPVersion("4.12.0")
	OCP_4_13 = ParseOCPVersion("4.13.0")
	OCP_4_14 = ParseOCPVersion("4.14.0")

	VERSIONS = []*Version{
		&SMCP_2_0,
		&SMCP_2_1,
		&SMCP_2_2,
		&SMCP_2_3,
		&SMCP_2_4,
		&OCP_4_9,
		&OCP_4_10,
		&OCP_4_11,
		&OCP_4_12,
		&OCP_4_13,
		&OCP_4_14,
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

func ParseOCPVersion(version string) Version {
	// OCP version is in the form of "major.minor.patch", example: 4.10.59
	majorMinorPatch := strings.Split(version, ".")
	if len(majorMinorPatch) != 3 {
		panic(fmt.Sprintf("invalid OCP version: %s", version))
	}
	major, err := strconv.Atoi(majorMinorPatch[0])
	if err != nil {
		panic(fmt.Sprintf("invalid OCP version: %s", version))
	}
	minor, err := strconv.Atoi(majorMinorPatch[1])
	if err != nil {
		panic(fmt.Sprintf("invalid OCP version: %s", version))
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
