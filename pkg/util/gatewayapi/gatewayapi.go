package gatewayapi

import (
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func GetSupportedVersion(smcp version.Version) string {
	switch smcp {
	case version.SMCP_2_3:
		return "v0.5.1"
	case version.SMCP_2_4:
		return "v0.5.1"
	case version.SMCP_2_5:
		return "v0.6.2"
	case version.SMCP_2_6:
		return "v1.0.0"
	default:
		return "v0.6.2"
	}
}

func GetDefaultServiceName(smcp version.Version, gatewayName string, className string) string {
	switch smcp {
	case version.SMCP_2_3:
		return gatewayName
	case version.SMCP_2_4:
		return gatewayName
	case version.SMCP_2_5:
		return gatewayName + "-" + className
	case version.SMCP_2_6:
		return gatewayName + "-" + className
	default:
		return gatewayName + "-" + className
	}
}

func GetWaitingCondition(smcp version.Version) string {
	switch smcp {
	case version.SMCP_2_3:
		return "Ready"
	case version.SMCP_2_4:
		return "Ready"
	case version.SMCP_2_5:
		return "Programmed"
	case version.SMCP_2_6:
		return "Programmed"
	default:
		return "Programmed"
	}
}
