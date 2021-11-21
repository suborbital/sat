FROM suborbital/wasmtime:dev as wasmtime

FROM golang:1.17 as builder

COPY --from=wasmtime /tmp/wasmtime/libwasmtime.a /usr/local/lib

RUN mkdir -p /go/sat
WORKDIR /go/sat

# Get dependencies first
COPY go.* ./
RUN go mod download

# Then everything else
COPY . /go/sat/
RUN make sat

FROM gcr.io/distroless/static-debian11

COPY ./ops/passwd /etc/passwd
COPY --from=builder /go/sat/.bin/sat /usr/local/bin/
ENV PATH=/usr/local/bin

USER sat
