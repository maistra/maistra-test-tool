#!/bin/bash

# Generates certificates used for testing

set -ex

# httpbin.example.com
pushd httpbin.example.com
echo "Create sample CA"; echo
openssl req -x509 -sha256 -nodes -days 3650 -newkey rsa:2048 -subj '/O=example Inc./CN=example.com' -keyout example.com.key -out example.com.crt

echo "Create httpbin server certs"; echo
openssl req -out httpbin.example.com.csr -newkey rsa:2048 -nodes -keyout httpbin.example.com.key -subj "/CN=httpbin.example.com/O=httpbin organization"
openssl x509 -req -sha256 -days 3650 -CA example.com.crt -CAkey example.com.key -set_serial 0 -in httpbin.example.com.csr -out httpbin.example.com.crt
echo "Add SANs"; echo
openssl x509 -req -extfile <(printf "subjectAltName=DNS:httpbin.example.com,DNS:www.httpbin.example.com") -days 3650 -in httpbin.example.com.csr -CA example.com.crt -CAkey example.com.key -CAcreateserial -out httpbin.example.com.crt

echo "Create httpbin client certs"; echo
openssl req -out httpbin-client.example.com.csr -newkey rsa:2048 -nodes -keyout httpbin-client.example.com.key -subj "/CN=httpbin-client.example.com/O=client organization"
openssl x509 -req -sha256 -days 3650 -CA example.com.crt -CAkey example.com.key -set_serial 1 -in httpbin-client.example.com.csr -out httpbin-client.example.com.crt
echo "Add SANs"; echo
openssl x509 -req -extfile <(printf "subjectAltName=DNS:httpbin-client.example.com,DNS:www.httpbin-client.example.com") -days 3650 -in httpbin-client.example.com.csr -CA example.com.crt -CAkey example.com.key -CAcreateserial -out httpbin-client.example.com.crt
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
