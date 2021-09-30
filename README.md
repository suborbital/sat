# Sat, the tiny WebAssembly compute module
> Sat (as in satellite) is an experiment, and isn't ready for production use. Please try it out and give feedback!

Sat is a compute module designed to have the maximum performance and smallest footprint available. Where our [Atmo](https://github.com/suborbital/atmo) project is a fully-fledged platform with support for running entire applications, Sat takes the opposite approach: run a single module really f****ing fast.

Sat has no dependencies (it's statically compiled), can run in the smallest Docker container (alpine), and includes no bells and whistles. It's designed to run in small environments such as the edge.

To run Sat, Docker is the easiest. Clone this repo and run:
```bash
make docker run
```
This will build the `suborbital/sat:dev` Docker image and then run it, using the `hello-echo.wasm` module found in the `testmodule` directory.

You can then make a POST request to `localhost:8080` with a body, and that body will be echoed back to you.

Sat executes modules with the [Runnable API](https://atmo.suborbital.dev/runnable-api/introduction) enabled, so you can create modules using our [Subo CLI](https://github.com/suborbital/subo) and all of the capabilities are available for use.

Note that statically compiling Sat on macOS is not currently possible, so Sat is Linux-only for now.

Copyright Suborbital contributors 2021.