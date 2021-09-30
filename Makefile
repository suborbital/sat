
sat:
	go build -o .bin/sat -tags netgo,wasmtime -ldflags="-extldflags=-static" ./sat

docker:
	docker build . -t suborbital/sat:dev

.PHONY: sat docker