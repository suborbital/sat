package api

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm/runtime"
)

func AbortHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		msgPtr := args[0].(int32)
		msgSize := args[1].(int32)
		filePtr := args[2].(int32)
		fileSize := args[3].(int32)
		lineNum := args[4].(int32)
		columnNum := args[5].(int32)
		ident := args[6].(int32)

		return_abort(msgPtr, msgSize, filePtr, fileSize, lineNum, columnNum, ident)

		return nil, nil
	}

	return runtime.NewHostFn("return_abort", 7, false, fn)
}

func return_abort(msgPtr int32, msgSize int32, filePtr int32, fileSize int32, lineNum int32, columnNum int32, ident int32) int32 {
	inst, err := runtime.InstanceForIdentifier(ident, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] alert: failed to InstanceForIdentifier"))
		return -1
	}

	msg := inst.ReadMemory(msgPtr, msgSize)
	fileName := inst.ReadMemory(filePtr, fileSize)

	errMsg := fmt.Sprintf("runnable abort: %s; file: %s, line: %d, col: %d", msg, fileName, lineNum, columnNum)
	runtime.InternalLogger().ErrorString(errMsg)

	inst.SendExecutionResult(nil, &rt.RunErr{Code: -1, Message: errMsg})

	return 0
}
