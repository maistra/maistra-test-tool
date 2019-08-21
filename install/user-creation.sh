#!/bin/bash

# Instructions: https://docs.openshift.com/container-platform/4.1/authentication/identity_providers/configuring-htpasswd-identity-provider.html

function create_htpasswd_file() {
  htpasswd -c -B -b users.htpasswd qe1 "${QE1_PWD:-qe1pw}"
  htpasswd -B -b users.htpasswd qe2 "${QE2_PWD:-qe2pw}"
}

function create_htpasswd_secret() {
  oc -n openshift-config create secret generic htpass-secret --from-file=htpasswd=users.htpasswd
}

function update_oauth() {
  oc apply -f <(cat <<EOF
apiVersion: config.openshift.io/v1
kind: OAuth
metadata:
  name: cluster
spec:
  identityProviders:
  - name: my_htpasswd_provider 
    mappingMethod: claim 
    type: HTPasswd
    htpasswd:
      fileData:
        name: htpass-secret
EOF
)
}


function main() {
  create_htpasswd_file
  create_htpasswd_secret
  update_oauth
}

main