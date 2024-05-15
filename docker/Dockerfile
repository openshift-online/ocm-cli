FROM registry.access.redhat.com/ubi9/go-toolset:latest AS builder

WORKDIR /opt/app-root/src
COPY . .


#TODO Identify the  full set of supported arch
#TODO Create a Make Target to build all supported release binaries
RUN go mod download
RUN go build -buildvcs=false -o ./releases/ocm ./cmd/ocm


FROM registry.access.redhat.com/ubi9/ubi-micro:latest
LABEL name="ocm-cli"
LABEL description="OCM CLI"
LABEL vendor="Red Hat, Inc."

COPY LICENSE.txt /licenses
COPY --from=builder /opt/app-root/src/releases /releases