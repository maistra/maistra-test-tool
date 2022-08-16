set -e

# shellcheck disable=SC1091
source common.sh

log "Creating projects for west-mesh"
oc1 new-project west-mesh-system || true
oc1 new-project bookinfo-ha || true

log "Creating projects for east-mesh"
oc2 new-project east-mesh-system || true
oc2 new-project bookinfo-ha || true

#log "Configure external CA cert"
#CACERT=cacerts/ca-cert.pem
#CAKEY=cacerts/ca-key.pem
#CAROOT=cacerts/root-cert.pem
#CACHAIN=cacerts/cert-chain.pem

#oc1 create -n west-mesh-system secret generic cacerts --from-file="${CACERT}" --from-file="${CAKEY}" --from-file="${CAROOT}" --from-file="${CACHAIN}" || true
#oc2 create -n east-mesh-system secret generic cacerts --from-file="${CACERT}" --from-file="${CAKEY}" --from-file="${CAROOT}" --from-file="${CACHAIN}" || true
#sleep 10

log "Installing control plane for west-mesh"
oc1 apply -f export/smcp.yaml
oc1 apply -f export/smmr.yaml
#oc1 patch -n west-mesh-system smcp/fed-export --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'

log "Installing control plane for east-mesh"
oc2 apply -f import/smcp.yaml
oc2 apply -f import/smmr.yaml
#oc2 patch -n mesh2-system smcp/fed-import --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'

log "Waiting for west-mesh installation to complete"
oc1 wait --for condition=Ready -n west-mesh-system smcp/fed-export --timeout 300s

log "Waiting for east-mesh installation to complete"
oc2 wait --for condition=Ready -n east-mesh-system smcp/fed-import --timeout 300s
