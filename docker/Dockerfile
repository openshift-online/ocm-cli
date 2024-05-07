FROM registry.access.redhat.com/ubi9/go-toolset:1.21.9-1.1714671022

COPY . .
RUN go mod download


RUN go build -buildvcs=false -o ./ocm  ./cmd/ocm/main.go 


CMD [ "./ocm --help" ]
