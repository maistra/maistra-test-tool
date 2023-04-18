// Copyright 2023 Red Hat, Inc.
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

package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// CertBuilder constructs a new set of keys and certificate.
// Usage examples: new(CertBuilder).NewCACert(); new(CertBuilder).NewServerCert(caCert)
type CertBuilder struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	template   *x509.Certificate
	parent     *x509.Certificate
	parentKey  *rsa.PrivateKey
	self       *x509.Certificate
	selfPEM    []byte
}

// NewCACert returns a new self-signed CA certificate.
func (cf *CertBuilder) NewCACert() *CertBuilder {
	return cf.newCert(true)
}

// NewServerCert returns a new server certificate. CA root or parent certificate can be specificed in the parameters.
func (cf *CertBuilder) NewServerCert(parentCert *x509.Certificate, parentKey *rsa.PrivateKey) *CertBuilder {
	if parentCert != nil {
		cf.SetParent(parentCert) // new servier certificate signed by the parent certificate
	}
	if parentKey != nil {
		cf.SetParentKey(parentKey) // new servier certificate signed by the parent private key
	}
	return cf.newCert(false)
}

func (cf *CertBuilder) newCert(isCA bool) *CertBuilder {
	if cf.privateKey == nil {
		cf.newKey(2048) // default new key bits 2048
	}

	if cf.template == nil {
		cf.SetTemplate("example Inc.", "example.com", 1, isCA) // default subject names and expiration 1 year
	}

	if cf.parent == nil {
		cf.parent = cf.template // self-signed certificate
	}

	signKey := cf.privateKey // default signed by new private key
	if cf.parentKey != nil {
		signKey = cf.parentKey
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cf.template, cf.parent, cf.publicKey, signKey)
	if err != nil {
		panic("Failed to create certificate:" + err.Error())
	}

	cf.self, err = x509.ParseCertificate(certBytes)
	if err != nil {
		panic("Failed to parse certificate:" + err.Error())
	}

	cf.selfPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return cf
}

func (cf *CertBuilder) newKey(bits int) error {
	// create new private and public key
	privKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return fmt.Errorf("Failed to create a private key:" + err.Error())
	}
	cf.privateKey, cf.publicKey = privKey, &privKey.PublicKey
	return nil
}

func (cf *CertBuilder) GetPrivateKey() *rsa.PrivateKey {
	return cf.privateKey
}

func (cf *CertBuilder) GetPrivateKeyPEM() []byte {
	if cf.privateKey == nil {
		return nil
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(cf.privateKey),
	})
}

func (cf *CertBuilder) GetPublicKeyPEM() []byte {
	if cf.publicKey == nil {
		return nil
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(cf.publicKey),
	})
}

func (cf *CertBuilder) GetCert() *x509.Certificate {
	return cf.self
}

func (cf *CertBuilder) GetCertPEM() []byte {
	return cf.selfPEM
}

func (cf *CertBuilder) SetTemplate(orgName, commonName string, expireYears int, isCA bool) {
	if isCA {
		cf.template = &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				Organization: []string{orgName},
				CommonName:   commonName,
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(expireYears, 0, 0),
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
			IsCA:                  true,
		}
	} else {
		cf.template = &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				Organization: []string{orgName},
				CommonName:   commonName,
			},
			NotBefore:   time.Now(),
			NotAfter:    time.Now().AddDate(expireYears, 0, 0),
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:    x509.KeyUsageDigitalSignature,
			IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		}
	}
}

func (cf *CertBuilder) SetParent(parent *x509.Certificate) {
	cf.parent = parent
}

func (cf *CertBuilder) SetParentKey(privateKey *rsa.PrivateKey) {
	cf.parentKey = privateKey
}
