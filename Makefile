
sat:
	go build -o .bin/sat -tags netgo,wasmtime -ldflags="-extldflags=-static" ./sat

sat/dynamic:
	go build -o .bin/sat -tags netgo,wasmtime ./sat

docker:
	docker build . -t suborbital/sat:dev

docker/publish:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/sat:$(shell date +%Y.%m.%d-%M) --push
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/sat:latest --push

docker/wasmtime:
	docker build ./ops -f ./ops/Dockerfile-wasmtime -t suborbital/wasmtime:dev

docker/wasmtime/publish:
	docker buildx build ./ops -f ./ops/Dockerfile-wasmtime --platform linux/amd64,linux/arm64 -t suborbital/wasmtime:dev --push

run:
	docker run -it -e SAT_HTTP_PORT=8080 -p 8080:8080 -v $(PWD)/examples:/runnables suborbital/sat:dev sat /runnables/hello-echo/hello-echo.wasm

.PHONY: sat docker