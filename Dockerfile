FROM golang:1.18 as builder
WORKDIR /go/sat

COPY go.* ./
RUN go mod download

COPY main.go .
COPY Makefile .
COPY ./sat ./sat
RUN make sat/static


FROM gcr.io/distroless/static-debian11
COPY ./ops/passwd /etc/passwd
COPY --from=builder /go/sat/.bin/sat /usr/local/bin/

ENV PATH=/usr/local/bin
USER sat
