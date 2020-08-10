FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1
WORKDIR /bin
RUN microdnf install python3 golang tar gzip git\
    && ln -s /usr/bin/python3 /usr/bin/python \
    && curl -Lo ./oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz \
    && tar -xf oc.tar.gz \
    && rm -f oc.tar.gz \
    && microdnf update \
    && microdnf clean all

COPY tests /opt/tests
WORKDIR /opt/tests
ENTRYPOINT /opt/tests/main.sh
