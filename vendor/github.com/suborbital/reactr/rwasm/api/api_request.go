package api

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rwasm/runtime"
)

func RequestGetFieldHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		fieldType := args[0].(int32)
		keyPointer := args[1].(int32)
		keySize := args[2].(int32)
		ident := args[3].(int32)

		ret := request_get_field(fieldType, keyPointer, keySize, ident)

		return ret, nil
	}

	return runtime.NewHostFn("request_get_field", 4, true, fn)
}

func request_get_field(fieldType int32, keyPointer int32, keySize int32, identifier int32) int32 {
	inst, err := runtime.InstanceForIdentifier(identifier, true)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	keyBytes := inst.ReadMemory(keyPointer, keySize)
	key := string(keyBytes)

	val, err := inst.Ctx().RequestHandler.GetField(fieldType, key)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "failed to GetField"))
	}

	result, err := inst.SetFFIResult(val, err)
	if err != nil {
		runtime.InternalLogger().ErrorString("[rwasm] failed to SetFFIResult", err.Error())
		return -1
	}

	return result.FFISize()
}

func RequestSetFieldHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		fieldType := args[0].(int32)
		keyPointer := args[1].(int32)
		keySize := args[2].(int32)
		valPointer := args[3].(int32)
		valSize := args[4].(int32)
		ident := args[5].(int32)

		ret := request_set_field(fieldType, keyPointer, keySize, valPointer, valSize, ident)

		return ret, nil
	}

	return runtime.NewHostFn("request_set_field", 6, true, fn)
}

func request_set_field(fieldType int32, keyPointer int32, keySize int32, valPointer int32, valSize int32, identifier int32) int32 {
	inst, err := runtime.InstanceForIdentifier(identifier, true)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	keyBytes := inst.ReadMemory(keyPointer, keySize)
	key := string(keyBytes)

	valBytes := inst.ReadMemory(valPointer, valSize)
	val := string(valBytes)

	if err := inst.Ctx().RequestHandler.SetField(fieldType, key, val); err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "failed to SetField"))
	}

	result, err := inst.SetFFIResult(nil, err)
	if err != nil {
		runtime.InternalLogger().ErrorString("[rwasm] failed to SetFFIResult", err.Error())
		return -1
	}

	return result.FFISize()
}
