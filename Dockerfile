FROM --platform=linux/amd64 ubuntu:22.04 as base

# Install pre-reqs
ARG DEBIAN_FRONTEND=noninteractive
RUN apt update >/dev/null
RUN apt install -y apt-utils jq build-essential unzip curl git pkg-config software-properties-common apt-transport-https ca-certificates >/dev/null

WORKDIR /root

# Install Go
RUN curl -sOL https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin:/go/bin
ENV GOPATH=/go
RUN rm -rf go*

# Install protoc
RUN curl -sOL https://github.com/protocolbuffers/protobuf/releases/download/v3.19.1/protoc-3.19.1-linux-x86_64.zip
RUN unzip protoc-3.19.1-linux-x86_64.zip -d protoc
RUN mv protoc/bin/* /usr/local/bin/
RUN mv protoc/include/* /usr/local/include/
RUN rm -rf protoc*

# Install librdkafka
RUN curl -sL https://packages.confluent.io/deb/6.1/archive.key | apt-key add - 2>/dev/null
RUN add-apt-repository "deb [arch=amd64] https://packages.confluent.io/deb/6.1 stable main"
RUN apt-get update && apt-get install -y librdkafka1 librdkafka-dev >/dev/null

# Init repo.
WORKDIR /go/src/github.com/DataDog/kafka-kit
COPY cluster cluster
COPY cmd cmd
COPY internal internal
COPY kafkaadmin kafkaadmin
COPY kafkametrics kafkametrics
COPY kafkazk kafkazk
COPY mapper mapper
COPY proto proto
COPY tools.go tools.go
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Install protoc / gRPC deps; these versions are managed in go.mod
RUN go get -d github.com/googleapis/googleapis
RUN go install \
  google.golang.org/protobuf/cmd/protoc-gen-go \
  google.golang.org/grpc/cmd/protoc-gen-go-grpc \
  github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway

# Codegen
RUN protoc -I ./proto/registrypb -I $GOPATH/pkg/mod/$(awk '/googleapis/ {printf "%s@%s", $1, $2}' go.mod) \
    --go_out ./proto/registrypb \
    --go_opt paths=source_relative \
    --go-grpc_out ./proto/registrypb \
    --go-grpc_opt paths=source_relative \
    --grpc-gateway_out ./proto/registrypb \
    --grpc-gateway_opt logtostderr=true \
    --grpc-gateway_opt paths=source_relative \
    --grpc-gateway_opt generate_unbound_methods=true \
    proto/registrypb/registry.proto

# Build
RUN go install ./cmd/...

# Clean
RUN apt autoremove
RUN apt clean

COPY entrypoint.sh /
ENTRYPOINT ["/entrypoint.sh"]

FROM registry.ddbuild.io/images/base/gbi-ubuntu_2204 as dd-image

COPY --from=base /entrypoint.sh /
COPY --from=base /go/src /go/src
COPY --from=base /go/bin /usr/bin

ENTRYPOINT ["/entrypoint.sh"]
