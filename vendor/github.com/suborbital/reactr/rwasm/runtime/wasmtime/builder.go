package runtimewasmtime

import (
	"github.com/bytecodealliance/wasmtime-go"
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rwasm/moduleref"
	"github.com/suborbital/reactr/rwasm/runtime"
)

// WasmTimeBuilder is a Wasmer implementation of the instanceBuilder interface
type WasmTimeBuilder struct {
	ref     *moduleref.WasmModuleRef
	hostFns []runtime.HostFn
	module  *wasmtime.Module
	engine  *wasmtime.Engine
	linker  *wasmtime.Linker
}

// NewBuilder creates a new WasmTimeBuilder
func NewBuilder(ref *moduleref.WasmModuleRef, hostFns ...runtime.HostFn) runtime.RuntimeBuilder {
	w := &WasmTimeBuilder{
		ref:     ref,
		hostFns: hostFns,
	}

	return w
}

func (w *WasmTimeBuilder) New() (runtime.RuntimeInstance, error) {
	module, engine, linker, err := w.internals()
	if err != nil {
		return nil, errors.Wrap(err, "failed to internals")
	}

	store := wasmtime.NewStore(engine)

	wasiConfig := wasmtime.NewWasiConfig()
	store.SetWasi(wasiConfig)

	wasmTimeInst, err := linker.Instantiate(store, module)
	if err != nil {
		return nil, errors.Wrap(err, "failed to linker.Instantiate")
	}

	inst := &WasmtimeInstance{
		inst:  *wasmTimeInst,
		store: store,
	}

	if _, err := inst.Call("_start"); err != nil {
		if errors.Is(err, runtime.ErrExportNotFound) {
			// that's ok, not all modules will have _start
		} else {
			return nil, errors.Wrap(err, "failed to call _start")
		}
	}

	// the deprecated `init` is not used in the Wasmtime runtime

	return inst, nil
}

func (w *WasmTimeBuilder) internals() (*wasmtime.Module, *wasmtime.Engine, *wasmtime.Linker, error) {
	if w.module == nil {
		moduleBytes, err := w.ref.Bytes()
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to get ref ModuleBytes")
		}

		engine := wasmtime.NewEngine()

		// Compiles the module
		mod, err := wasmtime.NewModule(engine, moduleBytes)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to NewModule")
		}

		// Create a linker with WASI functions defined within it
		linker := wasmtime.NewLinker(engine)
		if err := linker.DefineWasi(); err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to DefineWasi")
		}

		// mount the Runnable API
		addHostFns(linker, w.hostFns...)

		w.module = mod
		w.engine = engine
		w.linker = linker
	}

	return w.module, w.engine, w.linker, nil
}
