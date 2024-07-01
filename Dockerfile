FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ARG HELM_VERSION="v3.11.3"
ARG GO_VERSION="1.20.3"
ARG OCP_VERSION="stable"

ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH
# we need to set HOME when running on OCP with random UID, otherwise the home is set to / and any writing there will fail with permission denied
ENV HOME=$GOPATH/src/maistra-test-tool

RUN microdnf install -y --nodocs tar gzip openssl findutils make git && \
    curl -Lo ./oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${OCP_VERSION}/openshift-client-linux.tar.gz && \
    tar -xf oc.tar.gz -C /usr/bin && \
    rm -f oc.tar.gz && \
    curl -Lo ./golang.tar.gz https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xf golang.tar.gz -C /usr/local && \
    rm -f golang.tar.gz && \
    curl -LOk https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    tar -xzf helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/bin/ && \
    rm -rf helm-${HELM_VERSION}-linux-amd64.tar.gz linux-amd64 && \
    microdnf update && \
    microdnf clean all && \
    mkdir -p "$GOPATH/src/maistra-test-tool" "$GOPATH/bin"

COPY . $HOME
WORKDIR $HOME

RUN go install gotest.tools/gotestsum@latest \
    && go mod download

# Set required permissions for OpenShift usage
RUN chgrp -R 0 $GOPATH \
    && chmod -R g=u $GOPATH

ENTRYPOINT ["scripts/runtests.sh"]
