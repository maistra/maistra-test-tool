FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /bin

RUN microdnf install --nodocs tar gcc gzip git bind-utils findutils sudo \
    && curl -Lo ./oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz \
    && tar -xf oc.tar.gz \
    && rm -f oc.tar.gz \
    && curl -Lo ./golang.tar.gz https://go.dev/dl/go1.20.3.linux-amd64.tar.gz \
    && tar -xf golang.tar.gz -C / \
    && rm -f golang.tar.gz \
    && microdnf update \
    && microdnf clean all

ENV GOROOT=/go
ENV GOPATH=/root/go
ENV PATH=$GOROOT/bin:$PATH

ENV OCP_API_URL ${OCP_API_URL}
ENV OCP_CRED_USR ${OCP_CRED_USR}
ENV OCP_CRED_PSW ${OCP_CRED_PSW}
ENV OCP_TOKEN ${OCP_TOKEN}

ENV TEST_GROUP ${TEST_GROUP}
ENV TEST_CASE ${TEST_CASE}

ENV SAMPLEARCH ${SAMPLEARCH}
ENV NIGHTLY ${NIGHTLY}
ENV ROSA ${ROSA}
ENV MUSTGATHERTAG ${MUSTGATHERTAG}

COPY . /opt/maistra-test-tool
WORKDIR /opt/maistra-test-tool

RUN go install github.com/jstemmer/go-junit-report/v2@latest && go mod download

ENTRYPOINT ["scripts/runtests.sh"]
