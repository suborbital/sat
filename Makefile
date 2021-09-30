
sat:
	go build -o .bin/sat -tags netgo,wasmtime -ldflags="-extldflags=-static" ./sat

docker:
	docker build . -t suborbital/sat:dev

run:
	docker run -it -e SAT_HTTP_PORT=8080 -p 8080:8080 -v $(PWD)/testmodule:/home/sat suborbital/sat:dev sat hello-echo.wasm

.PHONY: sat docker