package api

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rwasm/runtime"
)

func CacheSetHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		keyPointer := args[0].(int32)
		keySize := args[1].(int32)
		valPointer := args[2].(int32)
		valSize := args[3].(int32)
		ttl := args[4].(int32)
		ident := args[5].(int32)

		ret := cache_set(keyPointer, keySize, valPointer, valSize, ttl, ident)

		return ret, nil
	}

	return runtime.NewHostFn("cache_set", 6, true, fn)
}

func cache_set(keyPointer int32, keySize int32, valPointer int32, valSize int32, ttl int32, identifier int32) int32 {
	inst, err := runtime.InstanceForIdentifier(identifier, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	key := inst.ReadMemory(keyPointer, keySize)
	val := inst.ReadMemory(valPointer, valSize)

	runtime.InternalLogger().Debug("[rwasm] setting cache key", string(key))

	if err := inst.Ctx().Cache.Set(string(key), val, int(ttl)); err != nil {
		runtime.InternalLogger().ErrorString("[rwasm] failed to set cache key", string(key), err.Error())
		return -2
	}

	return 0
}

func CacheGetHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		keyPointer := args[0].(int32)
		keySize := args[1].(int32)
		ident := args[2].(int32)

		ret := cache_get(keyPointer, keySize, ident)

		return ret, nil
	}

	return runtime.NewHostFn("cache_get", 3, true, fn)
}

func cache_get(keyPointer int32, keySize int32, identifier int32) int32 {
	inst, err := runtime.InstanceForIdentifier(identifier, true)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	key := inst.ReadMemory(keyPointer, keySize)

	runtime.InternalLogger().Debug("[rwasm] getting cache key", string(key))

	val, err := inst.Ctx().Cache.Get(string(key))
	if err != nil {
		runtime.InternalLogger().ErrorString("[rwasm] failed to get cache key", string(key), err.Error())
	}

	result, err := inst.SetFFIResult(val, err)
	if err != nil {
		runtime.InternalLogger().ErrorString("[rwasm] failed to SetFFIResult", err.Error())
		return -1
	}

	return result.FFISize()
}
