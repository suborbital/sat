FROM suborbital/wasmtime:dev as wasmtime

FROM golang:1.17 as builder

RUN mkdir -p /go/sat
COPY --from=wasmtime /tmp/wasmtime/libwasmtime.a /usr/local/lib
COPY . /go/sat/
WORKDIR /go/sat

RUN make sat

FROM gcr.io/distroless/static-debian11

# RUN addgroup -S satgroup && adduser -S sat -G satgroup
# RUN mkdir -p /home/sat && chown -R sat /home/sat && chmod -R 700 /home/sat

COPY --from=builder /go/sat/.bin/sat /usr/local/bin/
ENV PATH=/usr/local/bin

# WORKDIR /home/sat

# USER sat
