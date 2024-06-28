package cni

import "github.com/maistra/maistra-test-tool/pkg/util/version"

type Resource struct {
	Obj            string
	Name           string
	UsedInVersions []version.Version
}

// Resources source: https://github.com/maistra/istio-operator/blob/maistra-2.6/pkg/controller/servicemesh/controlplane/cni_pruner.go
var CniResources = []Resource{
	toResource("ClusterRole", "istio-cni", version.SMCP_2_0, version.SMCP_2_1, version.SMCP_2_2, version.SMCP_2_3),
	toResource("ClusterRole", "ossm-cni", version.SMCP_2_4, version.SMCP_2_5, version.SMCP_2_6),
	toResource("ClusterRoleBinding", "istio-cni", version.SMCP_2_0, version.SMCP_2_1, version.SMCP_2_2, version.SMCP_2_3),
	toResource("ClusterRoleBinding", "ossm-cni", version.SMCP_2_4, version.SMCP_2_5, version.SMCP_2_6),
	toResource("ConfigMap", "istio-cni-config", version.SMCP_2_0, version.SMCP_2_1, version.SMCP_2_2),
	toResource("ConfigMap", "istio-cni-config-v2-3", version.SMCP_2_3),
	toResource("ConfigMap", "ossm-cni-config-v2-4", version.SMCP_2_4),
	toResource("ConfigMap", "ossm-cni-config-v2-5", version.SMCP_2_5),
	toResource("ConfigMap", "ossm-cni-config-v2-6", version.SMCP_2_6),
	toResource("DaemonSet", "istio-cni-node", version.SMCP_2_0, version.SMCP_2_1, version.SMCP_2_2),
	toResource("DaemonSet", "istio-cni-node-v2-3", version.SMCP_2_3),
	toResource("DaemonSet", "istio-cni-node-v2-4", version.SMCP_2_4),
	toResource("DaemonSet", "istio-cni-node-v2-5", version.SMCP_2_5),
	toResource("DaemonSet", "istio-cni-node-v2-6", version.SMCP_2_6),
	toResource("ServiceAccount", "istio-cni", version.SMCP_2_0, version.SMCP_2_1, version.SMCP_2_2, version.SMCP_2_3),
	toResource("ServiceAccount", "ossm-cni", version.SMCP_2_4, version.SMCP_2_5, version.SMCP_2_6),
}

func toResource(obj string, name string, versions ...version.Version) Resource {
	r := Resource{
		Obj:  obj,
		Name: name,
	}
	r.UsedInVersions = append(r.UsedInVersions, versions...)

	return r
}
