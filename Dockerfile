FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /bin
RUN microdnf install --nodocs golang tar gzip git bind-utils sudo \
    && curl -Lo ./oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz \
    && tar -xf oc.tar.gz \
    && rm -f oc.tar.gz \
    && microdnf update \
    && microdnf clean all

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
