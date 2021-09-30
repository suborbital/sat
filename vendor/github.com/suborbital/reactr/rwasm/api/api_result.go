package api

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm/runtime"
)

func ReturnResultHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		pointer := args[0].(int32)
		size := args[1].(int32)
		ident := args[2].(int32)

		return_result(pointer, size, ident)

		return nil, nil
	}

	return runtime.NewHostFn("return_result", 3, false, fn)
}

func return_result(pointer int32, size int32, identifier int32) {
	inst, err := runtime.InstanceForIdentifier(identifier, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	result := inst.ReadMemory(pointer, size)

	inst.SendExecutionResult(result, nil)
}

func ReturnErrorHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		code := args[0].(int32)
		pointer := args[1].(int32)
		size := args[2].(int32)
		ident := args[3].(int32)

		return_error(code, pointer, size, ident)

		return nil, nil
	}

	return runtime.NewHostFn("return_error", 4, false, fn)
}

func return_error(code int32, pointer int32, size int32, identifier int32) {
	inst, err := runtime.InstanceForIdentifier(identifier, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	result := inst.ReadMemory(pointer, size)

	runErr := &rt.RunErr{Code: int(code), Message: string(result)}

	inst.SendExecutionResult(nil, runErr)
}
