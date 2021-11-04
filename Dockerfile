FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /bin
RUN microdnf install --nodocs golang tar gzip git bind-utils sudo \
    && curl -Lo ./oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz \
    && tar -xf oc.tar.gz \
    && rm -f oc.tar.gz \
    && microdnf update \
    && microdnf clean all

COPY . /opt/maistra-test-tool
WORKDIR /opt/maistra-test-tool
ENTRYPOINT /opt/maistra-test-tool/scripts/pipeline/main.sh
