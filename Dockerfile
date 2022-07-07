FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /bin

RUN microdnf install --nodocs tar gcc gzip git bind-utils sudo \
    && curl -Lo ./oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz \
    && tar -xf oc.tar.gz \
    && rm -f oc.tar.gz \
    && curl -Lo ./golang.tar.gz https://go.dev/dl/go1.16.15.linux-amd64.tar.gz \
    && tar -xf golang.tar.gz -C / \
    && rm -f golang.tar.gz \
    && microdnf update \
    && microdnf clean all

ENV GOROOT=/go
ENV PATH=$GOROOT/bin:$PATH
ENV SAMPLEARCH ${SAMPLEARCH}
ENV OCP_CRED_USR ${OCP_CRED_USR}
ENV OCP_CRED_PSW ${OCP_CRED_PSW}
ENV OCP_API_URL ${OCP_API_URL}
ENV NIGHTLY ${NIGHTLY}
ENV TEST_CASE ${TEST_CASE}
ENV ROSA ${ROSA}
ENV GODEBUG "x509ignoreCN=0"

COPY . /opt/maistra-test-tool
WORKDIR /opt/maistra-test-tool/tests

# ENTRYPOINT is not a shell, if you need export environment variables, use ["/bin/bash/", "-c", "scripts"]
ENTRYPOINT ["../scripts/pipeline/run_all_tests.sh"]
