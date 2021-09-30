FROM golang:1.17 as builder

RUN mkdir -p /go/sat
COPY . /go/sat/
WORKDIR /go/sat

RUN make sat

FROM alpine:latest

COPY --from=builder /go/sat/.bin/sat /usr/local/bin