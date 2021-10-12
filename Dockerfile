FROM golang:1.17 as deps

RUN apt-get update && apt-get install -y xz-utils
RUN go install github.com/itchyny/gojq/cmd/gojq@latest
RUN mkdir -p /tmp/wasmtime
COPY ./docker-deps.sh /tmp/wasmtime/
WORKDIR /tmp/wasmtime
RUN bash ./docker-deps.sh

FROM golang:1.17 as builder

RUN mkdir -p /go/sat
COPY --from=deps /tmp/wasmtime/libwasmtime.a /usr/local/lib
COPY . /go/sat/
WORKDIR /go/sat

RUN make sat

FROM alpine:latest

RUN addgroup -S satgroup && adduser -S sat -G satgroup
RUN mkdir -p /home/sat && chown -R sat /home/sat && chmod -R 700 /home/sat

COPY --from=builder /go/sat/.bin/sat /usr/local/bin

WORKDIR /home/sat

USER sat