# Sat, the tiny WebAssembly compute module
> Sat (as in satellite) is an experiment, and isn't ready for production use. Please try it out and give feedback!

Sat is a compute module designed to have the maximum performance and smallest possible footprint. Where our [Atmo](https://github.com/suborbital/atmo) project is a fully-fledged platform with support for running entire applications, Sat takes the opposite approach: run a single module really f***ing fast.

Sat has no dependencies (it's statically compiled), and can run in a tiny Docker container (Alpine). It's meant to live in small places such as edge compute instances.

### Using Sat
To run Sat, Docker is easiest. Clone this repo and run:
```bash
make docker run
```
This will build the `suborbital/sat:dev` Docker image and then launch it using the `hello-echo.wasm` module found in the `testmodule` directory.

You can then make a POST request to `localhost:8080` with a body, and that body will be echoed back to you.

Sat executes modules with the [Runnable API](https://atmo.suborbital.dev/runnable-api/introduction) enabled, so you can create modules using our [Subo CLI](https://github.com/suborbital/subo) and all of the capabilities are available for use.

Note that statically compiling Sat on macOS is not currently possible, so it is Linux-only for now.

### Stdin mode
As an alternative to running Sat as a server, you can also use it in `stdin` mode. First, build Sat:
```bash
make sat
OR
make sat/dynamic #on macOS
```
Then, run Sat with an input on stdin:
```bash
echo "world" | .bin/sat --stdin ./testmodule/hello-echo.wasm
```
Sat will write the response to stdout and exit.

### More details
Sat's only 'fancy' feature is the ability to create a mesh with other instances using local network discovery and websockets. By default, Sat starts on a random port, and listens for requests from its peers. In the future, this will enable some very interesting network topologies and potentially an integration with Atmo, but for now we are focused on being tiny and fast.

Copyright Suborbital contributors 2021.