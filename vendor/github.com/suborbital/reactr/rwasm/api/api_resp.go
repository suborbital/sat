package api

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/reactr/rwasm/runtime"
)

func RespSetHeaderHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		keyPointer := args[0].(int32)
		keySize := args[1].(int32)
		valPointer := args[2].(int32)
		valSize := args[3].(int32)
		ident := args[4].(int32)

		response_set_header(keyPointer, keySize, valPointer, valSize, ident)

		return nil, nil
	}

	return runtime.NewHostFn("resp_set_header", 5, false, fn)
}

func response_set_header(keyPointer int32, keySize int32, valPointer int32, valSize int32, ident int32) int32 {
	inst, err := runtime.InstanceForIdentifier(ident, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] alert: failed to InstanceForIdentifier"))
		return -1
	}

	keyBytes := inst.ReadMemory(keyPointer, keySize)
	key := string(keyBytes)

	valBytes := inst.ReadMemory(valPointer, valSize)
	val := string(valBytes)

	if err := inst.Ctx().RequestHandler.SetResponseHeader(key, val); err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] failed to SetResponseHeader"))

		if err == rcap.ErrReqNotSet {
			return -2
		} else {
			return -5
		}
	}

	return 0
}
