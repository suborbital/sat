//go:build wasmedge
// +build wasmedge

package engine

import (
	"github.com/suborbital/sat/api"
	"github.com/suborbital/sat/engine/moduleref"
	"github.com/suborbital/sat/engine/runtime"
	runtimewasmedge "github.com/suborbital/sat/engine/runtime/wasmedge"
)

func runtimeBuilder(ref *moduleref.WasmModuleRef, api api.HostAPI) runtime.RuntimeBuilder {
	return runtimewasmedge.NewBuilder(ref, api)
}
