# syntax=docker/dockerfile:1.4
FROM --platform=$TARGETPLATFORM golang:1.21 as builder

WORKDIR /kloudmeter

COPY go.mod go.sum ./

RUN go mod download -x

COPY . .

ENV CGO_ENABLED=0

RUN go build -o ./bin/kloudmeter main.go


FROM --platform=$TARGETPLATFORM ubuntu:latest

WORKDIR /kloudmeter

# Install necessary packages and download NATS Server
RUN apt-get update && \
    apt-get install -y wget && \
    wget https://github.com/nats-io/nats-server/releases/download/v2.9.20/nats-server-v2.9.20-linux-amd64.tar.gz && \
    tar -xzf nats-server-v2.9.20-linux-amd64.tar.gz && \
    mv nats-server-v2.9.20-linux-amd64/nats-server /usr/local/bin/ && \
    rm -rf nats-server-v2.9.20-linux-amd64* && \
    apt-get remove -y wget && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /kloudmeter/bin/kloudmeter /kloudmeter/bin/kloudmeter
COPY --from=builder /kloudmeter/nats.conf /kloudmeter/nats.conf

ENTRYPOINT ["/bin/sh", "-c", "/usr/local/bin/nats-server -js -c /kloudmeter/nats.conf & /kloudmeter/bin/kloudmeter"]

