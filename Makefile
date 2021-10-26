
sat:
	go build -o .bin/sat -tags netgo,wasmtime -ldflags="-extldflags=-static" ./sat

sat/dynamic:
	go build -o .bin/sat -tags netgo,wasmtime ./sat

docker:
	docker build . -t suborbital/sat:dev

docker/publish: docker
	docker push suborbital/sat:dev

docker/wasmtime:
	docker build ./ops -f ./ops/Dockerfile-wasmtime -t suborbital/wasmtime:dev

docker/wasmtime/publish: docker/wasmtime
	docker push suborbital/wasmtime:dev

run:
	docker run -it -e SAT_HTTP_PORT=8080 -p 8080:8080 -v $(PWD)/examples:/home/sat suborbital/sat:dev sat ./hello-echo/hello-echo.wasm

.PHONY: sat docker