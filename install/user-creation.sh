#!/bin/bash

# Instructions: https://docs.openshift.com/container-platform/4.1/authentication/identity_providers/configuring-htpasswd-identity-provider.html

function create_htpasswd_file() {
  htpasswd -c -B -b users.htpasswd qe1 "${QE1_PWD:-qe1pw}"
  htpasswd -B -b users.htpasswd qe2 "${QE2_PWD:-qe2pw}"
  htpasswd -B -b users.htpasswd ike "${IKE_PWD:-let_ike_in}"
  htpasswd -B -b users.htpasswd aslak "${ASLAK_PWD:-let_aslak_in}"
  htpasswd -B -b users.htpasswd bartosz "${BARTOSZ_PWD:-let_bartosz_in}"
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

function create_cluster_admin_user() {
  oc adm policy add-cluster-role-to-user cluster-admin "ike"
  oc adm policy add-cluster-role-to-user cluster-admin "aslak"
  oc adm policy add-cluster-role-to-user cluster-admin "bartosz"

}

function main() {
  create_htpasswd_file
  create_htpasswd_secret
  update_oauth
  create_cluster_admin_user
}

main