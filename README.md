# Sat, the tiny WebAssembly edge compute server
> Sat (as in satellite) is an experiment, and isn't ready for production use. Please try it out and give feedback!

[![Open in GitPod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/suborbital/sat)

Sat is a WebAssembly-powered server designed to have the maximum performance and smallest possible footprint. Our [Atmo](https://github.com/suborbital/atmo) project is a fully-fledged platform with support for running entire applications, whereas Sat takes the opposite approach: run a single Wasm module really f***ing fast.

Sat has no dependencies (it's statically compiled), and can run in a tiny Docker container (Distroless) or on bare metal. It's meant to live in small places such as edge compute environments.

### Using Sat

To run Sat, Docker is easiest:
```bash
docker run -it -e SAT_HTTP_PORT=8080 -p 8080:8080 suborbital/sat:latest sat https://github.com/suborbital/reactr/blob/main/rwasm/testdata/hello-echo/hello-echo.wasm\?raw\=true
```
Sat will start up, download the `hello-echo` module from the `examples` directory, and make it available on port 8080. You can then make a POST request to `localhost:8080`, and the body will be echoed back to you.
```bash
curl localhost:8080 -d 'my friend'
```
Sat executes modules with the [Runnable API](https://atmo.suborbital.dev/runnable-api/introduction) enabled, so you can create modules using our [Subo CLI](https://github.com/suborbital/subo) and all of the capabilities are available for use.

### Building Sat
If you'd like to build Sat yourself, clone this repo and run:
```bash
# On M1, you may need to run `export DOCKER_BUILDKIT=0`
make docker run
```
This will build the `suborbital/sat:dev` Docker image and then launch it using the same `hello-echo` module.

### Stdin mode
As an alternative to running Sat as a server, you can also use it in `stdin` mode. First, build Sat:
```bash
make sat
OR
make sat/dynamic #on macOS
```
Then, run Sat with an input on stdin:
```bash
echo "world" | .bin/sat --stdin ./examples/hello-echo/hello-echo.wasm
```
Sat will write the response to stdout and exit.

Note that statically compiling Sat on macOS is not currently possible, and compiling on M1 Macs is not possible unless you build Wasmtime from source, hence Docker as the reccomended method.

### Run from URL
If you provide a URL as the path argument to Sat, it will download the module from that URL, write it to a temp directory, and use it for execution:
```bash
.bin/sat "https://github.com/suborbital/reactr/blob/main/rwasm/testdata/hello-echo/hello-echo.wasm?raw=true"
```
The URL must be HTTPS and must have a `.wasm` suffix (excluding query parameters)

### Constellations
Sat is designed to run in a constellation, i.e. a meshed cluster of instances. This enables very interesting network topologies which can run applications in massively distributed and 'edge' environments. This repo includes the `constd` tool, which is an experiment-atop-experiment constellation manager that can run [Atmo](https://github.com/suborbital/atmo) applications in a distributed manner. You can learn more [here](./constd).

Copyright Suborbital contributors 2021.
