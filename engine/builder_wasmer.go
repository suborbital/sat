//go:build wasmer
// +build wasmer

package engine

import (
	"github.com/suborbital/sat/api"
	"github.com/suborbital/sat/engine/moduleref"
	"github.com/suborbital/sat/engine/runtime"
	runtimewasmer "github.com/suborbital/sat/engine/runtime/wasmer"
)

func runtimeBuilder(ref *moduleref.WasmModuleRef, api api.HostAPI) runtime.RuntimeBuilder {
	return runtimewasmer.NewBuilder(ref, api)
}
