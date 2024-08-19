#!/bin/bash
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
#
# Copyright 2024 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

# Generates certificates used for testing

set -ex

# httpbin.example.com
pushd httpbin.example.com

# root CA for httpbin.example.com, helloworld-v1.example.com, bookinfo.com
echo "Create sample CA"; echo
openssl req -x509 -sha256 -nodes -days 3650 -newkey rsa:2048 -subj '/O=example Inc./CN=*.example.com' -keyout example.com.key -out example.com.crt -addext "subjectAltName = DNS:*.example.com"

cat > "client.conf" <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = httpbin-client.example.com
DNS.2 = www.httpbin-client.example.com
EOF

cat > "server.conf" <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = httpbin.example.com
DNS.2 = www.httpbin.example.com
EOF

echo "Create httpbin server certs"; echo
openssl req -out httpbin.example.com.csr -newkey rsa:2048 -nodes -keyout httpbin.example.com.key -subj "/CN=httpbin.example.com/O=httpbin organization" -config "server.conf"
openssl x509 -req -days 3650 -CA example.com.crt -CAkey example.com.key -set_serial 0 -in httpbin.example.com.csr -out httpbin.example.com.crt -extensions v3_req -extfile "server.conf"

echo "Create httpbin client certs"; echo
openssl req -out httpbin-client.example.com.csr -newkey rsa:2048 -nodes -keyout httpbin-client.example.com.key -subj "/CN=httpbin-client.example.com/O=client organization" -config "client.conf"
openssl x509 -req -days 3650 -CA example.com.crt -CAkey example.com.key -set_serial 0 -in httpbin-client.example.com.csr -out httpbin-client.example.com.crt -extensions v3_req -extfile "client.conf"

echo "Create second httpbin client certs"; echo
openssl req -out httpbin-client-revoked.example.com.csr -newkey rsa:2048 -nodes -keyout httpbin-client-revoked.example.com.key -subj "/CN=httpbin-client.example.com/O=client organization" -config "client.conf"
openssl x509 -req -days 3650 -CA example.com.crt -CAkey example.com.key -set_serial 1 -in httpbin-client-revoked.example.com.csr -out httpbin-client-revoked.example.com.crt -extensions v3_req -extfile "client.conf"

# revoke db and config
cat /dev/null > "index.txt"
cat > "crl.conf" <<EOF
[ ca ]
default_ca      = CA_default             # The default ca section

[ CA_default ]
dir             = "./"                   # Where everything is kept
database        = "./index.txt"          # database index file.
certificate     = "./example.com.crt"    # The CA certificate
private_key     = "./example.com.key"    # The private key

# crlnumber must also be commented out to leave a V1 CRL.
crl_extensions = crl_ext

default_md      = sha256                # use SHA-256 by default
default_crl_days= 3650                  # how long before next CRL

[ crl_ext ]
# CRL extensions.
# Only issuerAltName and authorityKeyIdentifier make any sense in a CRL.
authorityKeyIdentifier=keyid:always
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
subjectAltName = @alt_names
[alt_names]
DNS = *.example.com
EOF
# revoke only client2 certificate
openssl ca -config "crl.conf" -revoke "httpbin-client-revoked.example.com.crt"
openssl ca -gencrl -out "example.com.crl" -config "crl.conf"

popd

# helloworld-v1.example.com
pushd helloworldv1
echo "Create helloworldv1 server certs"; echo
openssl req -out helloworld-v1.example.com.csr -newkey rsa:2048 -nodes -keyout helloworld-v1.example.com.key -subj "/CN=helloworld-v1.example.com/O=helloworld-v1 organization"
openssl x509 -req -sha256 -days 3650 -CA ../httpbin.example.com/example.com.crt -CAkey ../httpbin.example.com/example.com.key -set_serial 0 -in helloworld-v1.example.com.csr -out helloworld-v1.example.com.crt
echo "Add SANs"; echo
openssl x509 -req -extfile <(printf "subjectAltName=DNS:helloworld-v1.example.com,DNS:www.helloworld-v1.example.com") -days 3650 -in helloworld-v1.example.com.csr -CA ../httpbin.example.com/example.com.crt -CAkey ../httpbin.example.com/example.com.key -CAcreateserial -out helloworld-v1.example.com.crt
popd

# bookinfo.com
pushd bookinfo.com
echo "Create bookinfo server certs"; echo
openssl req -out bookinfo.com.csr -newkey rsa:2048 -nodes -keyout bookinfo.com.key -subj "/CN=bookinfo.com/O=bookinfo organization"
openssl x509 -req -sha256 -days 3650 -CA ../httpbin.example.com/example.com.crt -CAkey ../httpbin.example.com/example.com.key -set_serial 0 -in bookinfo.com.csr -out bookinfo.com.crt
echo "Add SANs"; echo
openssl x509 -req -extfile <(printf "subjectAltName=DNS:bookinfo.com,DNS:www.bookinfo.com") -days 3650 -in bookinfo.com.csr -CA ../httpbin.example.com/example.com.crt -CAkey ../httpbin.example.com/example.com.key -CAcreateserial -out bookinfo.com.crt
popd

# nginx.example.com
pushd nginx.example.com
echo "Create sample CA"; echo
openssl req -x509 -sha256 -nodes -days 3650 -newkey rsa:2048 -subj '/O=example Inc./CN=example.com' -keyout example.com.key -out example.com.crt

echo "Create nginx server certs"; echo
openssl req -out nginx.example.com.csr -newkey rsa:2048 -nodes -keyout nginx.example.com.key -subj "/CN=nginx.example.com/O=nginx organization"
openssl x509 -req -sha256 -days 3650 -CA example.com.crt -CAkey example.com.key -set_serial 0 -in nginx.example.com.csr -out nginx.example.com.crt
echo "Add SANs"; echo
openssl x509 -req -extfile <(printf "subjectAltName=DNS:nginx.example.com,DNS:www.nginx.example.com") -days 3650 -in nginx.example.com.csr -CA example.com.crt -CAkey example.com.key -CAcreateserial -out nginx.example.com.crt

echo "Create nginx client certs"; echo
openssl req -out nginx-client.example.com.csr -newkey rsa:2048 -nodes -keyout nginx-client.example.com.key -subj "/CN=nginx-client.example.com/O=client organization"
openssl x509 -req -sha256 -days 3650 -CA example.com.crt -CAkey example.com.key -set_serial 1 -in nginx-client.example.com.csr -out nginx-client.example.com.crt
echo "Add SANs"; echo
openssl x509 -req -extfile <(printf "subjectAltName=DNS:nginx-client.example.com,DNS:www.nginx-client.example.com") -days 3650 -in nginx-client.example.com.csr -CA example.com.crt -CAkey example.com.key -CAcreateserial -out nginx-client.example.com.crt

# my-nginx.mesh-external.svc.cluster.local
echo "Create my-nginx.mesh-external server certs"; echo
openssl req -out my-nginx.mesh-external.svc.cluster.local.csr -newkey rsa:2048 -nodes -keyout my-nginx.mesh-external.svc.cluster.local.key -subj "/CN=my-nginx.mesh-external.svc.cluster.local/O=nginx organization"
openssl x509 -req -sha256 -days 3650 -CA example.com.crt -CAkey example.com.key -set_serial 0 -in my-nginx.mesh-external.svc.cluster.local.csr -out my-nginx.mesh-external.svc.cluster.local.crt
echo "Add SANs"; echo
openssl x509 -req -extfile <(printf "subjectAltName=DNS:my-nginx.mesh-external.svc.cluster.local,DNS:www.my-nginx.mesh-external.svc.cluster.local") -days 3650 -in my-nginx.mesh-external.svc.cluster.local.csr -CA example.com.crt -CAkey example.com.key -CAcreateserial -out my-nginx.mesh-external.svc.cluster.local.crt
popd
