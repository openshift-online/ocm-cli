FROM registry.access.redhat.com/ubi9/go-toolset:latest AS builder
COPY . .

# For hermetic builds, The hermeto file must be sourced.
# https://hermetoproject.github.io/hermeto/usage/#write-the-dockerfile
ENV GOFLAGS=-buildvcs=false
RUN git config --global --add safe.directory /opt/app-root/src && \
    ( [ -f /tmp/hermeto.env ] && source /tmp/hermeto.env && make build_release_images ) || \
    ( echo "No hermetic configuration found, building non-hermatically" && make build_release_images )

FROM registry.access.redhat.com/ubi9/ubi-micro:latest
LABEL description="A CLI tool for working with OCM API"
LABEL io.k8s.description="A CLI tool for working with OCM API"
LABEL io.k8s.display-name="OCM CLI"
LABEL io.openshift.tags="ocm"
LABEL summary="Provides the ocm CLI binary"
LABEL com.redhat.component="ocm"
LABEL name="ocm-cli-clients/ocm-cli-rhel9"
LABEL cpe="cpe:/a:redhat:openshift_cluster_manager_cli:1.0::el9"


COPY LICENSE.txt /licenses
COPY --from=builder /opt/app-root/src/releases /releases
COPY --from=builder /opt/app-root/src/releases /usr/local/bin
COPY --from=builder /usr/bin/make /usr/local/bin
