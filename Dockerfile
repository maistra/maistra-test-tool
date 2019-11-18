FROM docker.io/fedora:31 AS builder

RUN dnf install -y --setopt=tsflags=nodocs \
    python37 && \
    dnf clean all

RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

COPY requirements.txt .
RUN pip install -r requirements.txt


FROM docker.io/fedora:31 AS installer

RUN dnf install -y --setopt=tsflags=nodocs \
    awscli && \
    dnf clean all

COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
COPY . /tmp/

WORKDIR /tmp/install 
CMD ["python", "main.py", "-h"]
