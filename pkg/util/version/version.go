package version

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	SMCP_2_1 = ParseVersion("2.1")
	SMCP_2_2 = ParseVersion("2.2")
	SMCP_2_3 = ParseVersion("2.3")
	SMCP_2_4 = ParseVersion("2.4")
)

type Version struct {
	Major int
	Minor int
}

func ParseVersion(version string) Version {
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

func (this Version) GreaterThanOrEqualTo(that Version) bool {
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

func (this Version) String() string {
	return fmt.Sprintf("%d.%d", this.Major, this.Minor)
}
