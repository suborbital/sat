//go:build wasmtime

package rwasm

import (
	"github.com/suborbital/reactr/rwasm/api"
	"github.com/suborbital/reactr/rwasm/moduleref"
	"github.com/suborbital/reactr/rwasm/runtime"
	runtimewasmtime "github.com/suborbital/reactr/rwasm/runtime/wasmtime"
)

func runtimeBuilder(ref *moduleref.WasmModuleRef) runtime.RuntimeBuilder {
	return runtimewasmtime.NewBuilder(ref, api.API()...)
}
