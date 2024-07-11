FROM registry.access.redhat.com/ubi9/go-toolset:latest AS builder
COPY . .

ENV GOFLAGS=-buildvcs=false
RUN git config --global --add safe.directory /opt/app-root/src && \
    make build_release_images

FROM registry.access.redhat.com/ubi9/ubi-micro:latest
LABEL description="OCM CLI"
LABEL io.k8s.description="OCM CLI test"
LABEL com.redhat.component="ocm-cli"
LABEL distribution-scope="release"
LABEL name="ocm-cli" release="X.Y" url="https://github.com/openshift-online/ocm-cli"
LABEL vendor="Red Hat, Inc."
LABEL version="X.Y"

COPY LICENSE.txt /licenses
COPY --from=builder /opt/app-root/src/releases /releases
