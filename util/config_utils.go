// Copyright 2019 Red Hat, Inc.
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
	"io"

	appsV1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

// ConfigCitadelCerts configuration for Plugging in External Certs test
func ConfigCitadelCerts(data []byte, w io.Writer) error {

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	obj, _, err := s.Decode(data, nil, nil)
	if err != nil {
		return err
	}
	deployment := obj.(*appsV1.Deployment)

	args := &(deployment.Spec.Template.Spec.Containers[0].Args)
	for i, item := range *args {
		if item == "--self-signed-ca=true" {
			(*args)[i] = "--self-signed-ca=false"
		}
	}
	newCertList := []string{
		"--signing-cert=/etc/cacerts/ca-cert.pem",
		"--signing-key=/etc/cacerts/ca-key.pem",
		"--root-cert=/etc/cacerts/root-cert.pem",
		"--cert-chain=/etc/cacerts/cert-chain.pem",
	}
	*args = append(*args, newCertList...)

	container := &(deployment.Spec.Template.Spec.Containers[0])
	vm := v1.VolumeMount{
		Name:      "cacerts",
		ReadOnly:  true,
		MountPath: "/etc/cacerts",
	}
	container.VolumeMounts = []v1.VolumeMount{vm}

	spec := &(deployment.Spec.Template.Spec)
	b := true
	secret := &v1.SecretVolumeSource{
		SecretName: "cacerts",
		Optional:   &b,
	}
	volume := v1.Volume{
		Name:         "cacerts",
		VolumeSource: v1.VolumeSource{Secret: secret},
	}
	spec.Volumes = []v1.Volume{volume}

	s.Encode(deployment, w)
	return nil
}

// ConfigCitadelHealthCheck configuration for Citadel Health Check Test
func ConfigCitadelHealthCheck(data []byte, w io.Writer) error {
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	obj, _, err := s.Decode(data, nil, nil)
	if err != nil {
		return err
	}
	deployment := obj.(*appsV1.Deployment)
	args := &(deployment.Spec.Template.Spec.Containers[0].Args)
	newCertList := []string{
		"--liveness-probe-path=/tmp/ca.liveness",
		"--liveness-probe-interval=10s",
		"--probe-check-interval=10s",
	}
	*args = append(*args, newCertList...)

	container := &(deployment.Spec.Template.Spec.Containers[0])
	commands := &v1.ExecAction{
		Command: []string{
			"/usr/local/bin/istio_ca",
			"probe",
			"--probe-path=/tmp/ca.liveness",
			"--interval=125s",
		},
	}
	probe := &v1.Probe{
		Handler:             v1.Handler{Exec: commands},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
	}
	container.LivenessProbe = probe

	s.Encode(deployment, w)
	return nil
}
