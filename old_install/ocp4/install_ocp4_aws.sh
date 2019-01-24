#!/bin/bash

set -e

echo -n "aws_access_key_id ? " 
read ACCESS_KEY_ID
 
echo -n "aws_secret_access_key ? " 
read SECRET_ACCESS_KEY

while getopts 'v:' OPTION; do
  case "$OPTION" in
    v) INSTALLER_VERSION="${OPTARG}" ;;
  esac
done
shift $((OPTIND-1))


sudo dnf install -y awscli

mkdir -p ~/.aws
cat >> ~/.aws/config <<"EOF"
[profile openshift-dev]
region = us-east-1
output = text
EOF

cat >> ~/.aws/credentials <<"EOF"
[openshift-dev]
aws_access_key_id = ${ACCESS_KEY_ID}
aws_secret_access_key = ${SECRET_ACCESS_KEY}
EOF

export AWS_PROFILE=openshift-dev

`wget https://github.com/openshift/installer/releases/download/v${INSTALLER_VERSION}/openshift-install-linux-amd64`

mv openshift-install-linux-amd64 openshift-install
chmod +x openshift-install

./openshift-install --dir=./assets create cluster

export KUBECONFIG=./assets/auth/kubeconfig

kubectl cluster-info
