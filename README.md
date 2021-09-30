# Sat, the tiny WebAssembly compute module
> Sat (as in satellite) is an experiment, and isn't ready for production use. Please try it out and give feedback!

Sat is a compute module designed to have the maximum performance and smallest footprint available. Where our [Atmo](https://github.com/suborbital/atmo) project is a fully-fledged platform with support for running entire applications, Sat takes the opposite approach: run a single module really f****ing fast.

Sat has no dependencies (it's statically compiled), can run in a tiny Docker container (Alpine). It's meant to run in small places such as edge compute instances.

To run Sat, Docker is the easiest. Clone this repo and run:
```bash
make docker run
```
This will build the `suborbital/sat:dev` Docker image and then run it, using the `hello-echo.wasm` module found in the `testmodule` directory.

You can then make a POST request to `localhost:8080` with a body, and that body will be echoed back to you.

Sat executes modules with the [Runnable API](https://atmo.suborbital.dev/runnable-api/introduction) enabled, so you can create modules using our [Subo CLI](https://github.com/suborbital/subo) and all of the capabilities are available for use.

Sat's only 'fancy' feature is the ability to create a mesh with other instances using local network discovery and websockets. By default, Sat starts on a random port, and listens for requests from its peers. In the future, this will enable some very interesting network topologies and potentially an integration with Atmo, but for now we are focused on being tiny and fast.

Note that statically compiling Sat on macOS is not currently possible, so Sat is Linux-only for now.

Copyright Suborbital contributors 2021.