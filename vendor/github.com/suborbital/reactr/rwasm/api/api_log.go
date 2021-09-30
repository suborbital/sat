package api

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/reactr/rwasm/runtime"
)

type logScope struct {
	RequestID  string `json:"request_id,omitempty"`
	Identifier int32  `json:"ident"`
}

func LogMsgHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		pointer := args[0].(int32)
		size := args[1].(int32)
		level := args[2].(int32)
		ident := args[3].(int32)

		log_msg(pointer, size, level, ident)

		return nil, nil
	}

	return runtime.NewHostFn("log_msg", 4, false, fn)
}

func log_msg(pointer int32, size int32, level int32, identifier int32) {
	inst, err := runtime.InstanceForIdentifier(identifier, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	msgBytes := inst.ReadMemory(pointer, size)

	scope := logScope{Identifier: identifier}

	// if this job is handling a request, add the Request ID for extra context
	if inst.Ctx().RequestHandler != nil {
		requestID, err := inst.Ctx().RequestHandler.GetField(rcap.RequestFieldTypeMeta, "id")
		if err != nil {
			// do nothing, we won't fail the log call because of this
		} else {
			scope.RequestID = string(requestID)
		}
	}

	inst.Ctx().LoggerSource.Log(level, string(msgBytes), scope)
}
