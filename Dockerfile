FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH
# we need to set HOME when running on OCP with random UID, otherwise the home is set to / and any writing there will fail with permission denied
ENV HOME=$GOPATH/src/maistra-test-tool

ENV TEST_GROUP ${TEST_GROUP}
ENV SAMPLEARCH ${SAMPLEARCH}
ENV OCP_CRED_USR ${OCP_CRED_USR}
ENV OCP_CRED_PSW ${OCP_CRED_PSW}
ENV OCP_TOKEN ${OCP_TOKEN}
ENV OCP_API_URL ${OCP_API_URL}
ENV NIGHTLY ${NIGHTLY}
ENV TEST_CASE ${TEST_CASE}
ENV ROSA ${ROSA}
ENV GODEBUG "x509ignoreCN=0"
ENV MUSTGATHERTAG ${MUSTGATHERTAG}
ENV IPV6 ${IPV6}

WORKDIR /bin
RUN microdnf install --nodocs tar gcc gzip git bind-utils sudo \
    && curl -Lo ./oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz \
    && tar -xf oc.tar.gz \
    && rm -f oc.tar.gz \
    && curl -Lo ./golang.tar.gz https://go.dev/dl/go1.20.2.linux-amd64.tar.gz \
    && tar -xf golang.tar.gz -C /usr/local \
    && rm -f golang.tar.gz \
    && microdnf update \
    && microdnf clean all \
    && mkdir -p "$GOPATH/src/maistra-test-tool" "$GOPATH/bin"


COPY . $HOME
WORKDIR $HOME/tests

# Set required permissions for OpenShift usage
RUN chgrp -R 0 $GOPATH \
    && chmod -R g=u $GOPATH

# using CMD here so it can be easily overwritten when using this in OpenShiftCI
CMD ["/bin/bash", "-c", "../scripts/pipeline/run_all_tests.sh"]
