VERSION = $(shell cat .image-ver)

sat:
	go build -o .bin/sat -tags netgo,wasmtime .

sat/static:
	go build -o .bin/sat -tags netgo,wasmtime -ldflags="-extldflags=-static" .

sat/install:
	go install -tags netgo,wasmtime .

docker:
	docker build . -t suborbital/sat:dev

docker/publish:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/sat:$(VERSION) --push
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/sat:latest --push

docker/dev/publish:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/sat:dev --push

docker/wasmtime:
	docker build ./ops -f ./ops/Dockerfile-wasmtime -t suborbital/wasmtime:dev

docker/wasmtime/publish:
	docker buildx build ./ops -f ./ops/Dockerfile-wasmtime --platform linux/amd64,linux/arm64 -t suborbital/wasmtime:dev --push

run:
	docker run -it -e SAT_HTTP_PORT=8080 -p 8080:8080 -v $(PWD)/examples:/runnables suborbital/sat:dev sat /runnables/hello-echo/hello-echo.wasm

test:
	go test -v ./...

int:
	go test -v ./tests --tags=integration
# CONSTD TARGETS

constd:
	go build -o .bin/constd -tags wasmtime,netgo ./constd

constd/install:
	go install -tags wasmtime,netgo ./constd

constd/static:
	go build -o .bin/constd -tags wasmtime,netgo -ldflags="-extldflags=-static" ./constd

constd/docker: constd
	CONSTD_ATMO_VERSION=dev CONSTD_SAT_VERSION=dev .bin/constd $(PWD)/constd/example-project/runnables.wasm.zip

constd/metal: constd
	CONSTD_EXEC_MODE=metal .bin/constd $(PWD)/constd/example-project/runnables.wasm.zip

constd/metal/otel/hc: constd
	ATMO_TRACER_TYPE="honeycomb" \
	ATMO_TRACER_PROBABILITY=${ATMO_TRACER_PROBABILITY} \
	ATMO_TRACER_HONEYCOMB_ENDPOINT=${ATMO_TRACER_HONEYCOMB_ENDPOINT} \
	ATMO_TRACER_HONEYCOMB_APIKEY=${ATMO_TRACER_HONEYCOMB_APIKEY} \
	ATMO_TRACER_HONEYCOMB_DATASET=${ATMO_TRACER_HONEYCOMB_DATASET} \
	CONSTD_EXEC_MODE=metal .bin/constd $(PWD)/constd/example-project/runnables.wasm.zip

constd/metal/otel/collector: constd
	ATMO_TRACER_TYPE="collector" \
	ATMO_TRACER_COLLECTOR_ENDPOINT="localhost:4317" \
	ATMO_TRACER_PROBABILITY=${ATMO_TRACER_PROBABILITY} \
	CONSTD_EXEC_MODE=metal .bin/constd $(PWD)/constd/example-project/runnables.wasm.zip

lint:
	docker compose up linter

importfix:
	docker compose up lintfixer

.PHONY: sat constd
