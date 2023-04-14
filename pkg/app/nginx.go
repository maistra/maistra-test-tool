package app

import (
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type nginx struct {
	ns   string
	mTLS bool
}

var _ App = &nginx{}

func Nginx(ns string) App {
	return &nginx{ns: ns, mTLS: false}
}

func NginxWithMTLS(ns string) App {
	return &nginx{ns: ns, mTLS: true}
}

func (a *nginx) Name() string {
	return "nginx"
}

func (a *nginx) Namespace() string {
	return a.ns
}

func (a *nginx) Install(t test.TestHelper) {
	t.T().Helper()
	oc.CreateGenericSecretFromFiles(t, a.Namespace(),
		"nginx-ca-certs",
		"example.com.crt="+examples.NginxServerCACert())
	if a.mTLS {
		oc.CreateTLSSecret(t, a.Namespace(), "nginx-server-certs", examples.MeshExtServerCertKey(), examples.MeshExtServerCert())
		oc.CreateConfigMapFromFiles(t, a.Namespace(),
			"nginx-configmap",
			"nginx.conf="+examples.NginxConfMTlsFile())
	} else {
		oc.CreateTLSSecret(t, a.Namespace(), "nginx-server-certs", examples.NginxServerCertKey(), examples.NginxServerCert())
		oc.CreateConfigMapFromFiles(t, a.Namespace(),
			"nginx-configmap",
			"nginx.conf="+examples.NginxConfFile())
	}
	oc.ApplyFile(t, a.Namespace(), examples.NginxYamlFile())
}

func (a *nginx) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFile(t, a.Namespace(), examples.NginxYamlFile())
	oc.DeleteSecret(t, a.Namespace(), "nginx-server-certs")
	oc.DeleteSecret(t, a.Namespace(), "nginx-ca-certs")
	oc.DeleteConfigMap(t, a.Namespace(), "nginx-configmap")
}

func (a *nginx) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "my-nginx")
}
