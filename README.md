![logo-sat-wide](https://user-images.githubusercontent.com/5942370/160295501-e6e39fba-8155-40e7-8892-6b4d829de122.svg)

> Sat (as in satellite) is a tiny WebAssembly edge compute server. It is in beta, and can be used for production-grade workloads with appropriate testing. Please try it out and give feedback!

[![Open in GitPod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/suborbital/sat)

Sat is a WebAssembly-powered server designed to have the maximum performance and smallest possible footprint. Our [Atmo](https://github.com/suborbital/atmo) project is a fully-fledged server framework and platform with support for running entire applications, whereas Sat takes the opposite approach: run a single Wasm module really f***ing fast.

Sat has no dependencies (it's statically compiled), and can run in a tiny Docker container (Distroless) or on bare metal. It's meant to live in small places such as edge compute environments.

### Documentation

For full usage documentation, visit the [Sat docs](https://docs.suborbital.dev/sat)

### Constellations

Sat is designed to run in a constellation, i.e. a meshed cluster of instances. This enables very interesting network topologies which can run applications in massively distributed and 'edge' environments. This repo includes the `constd` tool, which is an experiment-atop-experiment constellation manager that can run [Atmo](https://github.com/suborbital/atmo) applications in a distributed manner using a Sat constellation. You can learn more in our [`constd` docs](https://docs.suborbital.dev/sat/constellations/).

Copyright Suborbital contributors 2022.
