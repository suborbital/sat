//go:build !wasmtime
// +build !wasmtime

package rwasm

import (
	"github.com/suborbital/reactr/rwasm/api"
	"github.com/suborbital/reactr/rwasm/moduleref"
	"github.com/suborbital/reactr/rwasm/runtime"
	runtimewasmer "github.com/suborbital/reactr/rwasm/runtime/wasmer"
)

func runtimeBuilder(ref *moduleref.WasmModuleRef) runtime.RuntimeBuilder {
	return runtimewasmer.NewBuilder(ref, api.API()...)
}
