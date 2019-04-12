#!/bin/bash
set -e

DIR=$(cd $(dirname $0); pwd -P)
BASE_DIR="${DIR}/../"

OC_COMMAND="oc"
CERTS_DIR="testdata/certs"


function banner() {
  message="$1"
  border="$(echo ${message} | sed -e 's+.+=+g')"
  echo "${border}"
  echo "${message}"
  echo "${border}"
}

function cleanup() {
    set +e
    banner "Cleanup"
    ${OC_COMMAND} delete secret cacerts -n istio-system
    # Roll back Citadel deployment
    ${OC_COMMAND} rollout undo deployment -n istio-system istio-citadel
    rm -f /tmp/istio-citadel-new.yaml
    rm -f /tmp/istio-citadel-bak.yaml
    echo "bookinfo" | ./bookinfo_uninstall.sh
    ${OC_COMMAND} delete meshpolicy default
}
trap cleanup EXIT

function enable_mtls() {
    cat <<EOF | ${OC_COMMAND} apply -f -
apiVersion: "authentication.istio.io/v1alpha1"
kind: "MeshPolicy"
metadata:
  name: "default"
spec:
  peers:
  - mtls: {}
EOF
}

function update_citadel_yaml() {
    ${OC_COMMAND} get deployment -n istio-system istio-citadel -o yaml > /tmp/istio-citadel-bak.yaml
    cp -f /tmp/istio-citadel-bak.yaml /tmp/istio-citadel-new.yaml
    TAB="        "
    sed -i "s@self-signed-ca=true@self-signed-ca=false@" /tmp/istio-citadel-new.yaml
    sed -i $"/self-signed-ca=false/a\\${TAB}- --signing-cert=/etc/cacerts/ca-cert.pem\n${TAB}- --signing-key=/etc/cacerts/ca-key.pem\n${TAB}- --root-cert=/etc/cacerts/root-cert.pem\n${TAB}- --cert-chain=/etc/cacerts/cert-chain.pem" \
        /tmp/istio-citadel-new.yaml

    sed -i $"/image:/a\\${TAB}volumeMounts:\n${TAB}- name: cacerts\n${TAB}  mountPath:  /etc/cacerts\n${TAB}  readOnly: true" \
        /tmp/istio-citadel-new.yaml
    TAB="      "
    sed -i $"/restartPolicy:/a\\${TAB}volumes:\n${TAB}- name: cacerts\n${TAB}  secret:\n${TAB}    secretName: cacerts\n${TAB}    optional: true" \
        /tmp/istio-citadel-new.yaml

}

function apply_certs() {
    # create a secret
    ${OC_COMMAND} create secret generic cacerts -n istio-system --from-file=${CERTS_DIR}/ca-cert.pem \
        --from-file=${CERTS_DIR}/ca-key.pem --from-file=${CERTS_DIR}/root-cert.pem \
        --from-file=${CERTS_DIR}/cert-chain.pem
    sleep 5

    # edit Citadel deployment
    #CITADELPOD=`${OC_COMMAND} get pods -n istio-system -l istio=citadel -o jsonpath='{.items[0].metadata.name}'`
    # oc edit deployment -n istio-system istio-citadel
    update_citadel_yaml
    ${OC_COMMAND} apply -f /tmp/istio-citadel-new.yaml
    sleep 10

    # delete existing secret
    set +e
    ${OC_COMMAND} delete secret istio.default
    set -e
}

function deploy_bookinfo() {
    
    echo "# Deploying bookinfo"
    echo "bookinfo" | ./bookinfo_install.sh -t
}

function verify_certs() {
    
    RATINGSPOD=`${OC_COMMAND} get pods -l app=ratings -o jsonpath='{.items[0].metadata.name}'`
    ${OC_COMMAND} exec -it $RATINGSPOD -c istio-proxy -- /bin/cat /etc/certs/root-cert.pem > /tmp/pod-root-cert.pem
    ${OC_COMMAND} exec -it $RATINGSPOD -c istio-proxy -- /bin/cat /etc/certs/cert-chain.pem > /tmp/pod-cert-chain.pem

    # verify certs
    openssl x509 -in ${CERTS_DIR}/root-cert.pem -text -noout > /tmp/root-cert.crt.txt
    openssl x509 -in /tmp/pod-root-cert.pem -text -noout > /tmp/pod-root-cert.crt.txt
    diff /tmp/root-cert.crt.txt /tmp/pod-root-cert.crt.txt
    if [ $? != 0 ]; then
        echo "# Error: crts are not the same"
        exit 1
    fi

    tail -n 22 /tmp/pod-cert-chain.pem > /tmp/pod-cert-chain-ca.pem
    openssl x509 -in ${CERTS_DIR}/ca-cert.pem -text -noout > /tmp/ca-cert.crt.txt
    openssl x509 -in /tmp/pod-cert-chain-ca.pem -text -noout > /tmp/pod-cert-chain-ca.crt.txt
    diff /tmp/ca-cert.crt.txt /tmp/pod-cert-chain-ca.crt.txt
    if [ $? != 0 ]; then
        echo "# Error: crts are not the same"
        exit 1
    fi

    head -n 21 /tmp/pod-cert-chain.pem > /tmp/pod-cert-chain-workload.pem
    openssl verify -CAfile <(cat ${CERTS_DIR}/ca-cert.pem ${CERTS_DIR}/root-cert.pem) /tmp/pod-cert-chain-workload.pem

}


function main() {
    banner "TC_21 Plugging in root certificate, signing certificate and key"
    enable_mtls
    apply_certs
    sleep 10
    deploy_bookinfo
    sleep 30
    verify_certs

    banner "TC_21 passed"

}

main
