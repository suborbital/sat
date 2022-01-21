package sat

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/coordinator/sequence"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
)

// handleFnResult is mounted onto exec.ListenAndRun...
// when a meshed peer sends us a job, it is executed by Reactr and then
// the result is passed into this function for handling
func (s *Sat) handleFnResult(msg grav.Message, result interface{}, fnErr error) {
	// first unmarshal the request and sequence information
	req, err := request.FromJSON(msg.Data())
	if err != nil {
		s.l.Error(errors.Wrap(err, "failed to request.FromJSON"))
		return
	}

	ctx := vk.NewCtx(s.l, nil, nil)
	ctx.UseRequestID(req.ID)
	ctx.UseScope(loggerScope{req.ID})

	seq, err := sequence.FromJSON(req.SequenceJSON, req, s.e, ctx)
	if err != nil {
		s.l.Error(errors.Wrap(err, "failed to sequence.FromJSON"))
		return
	}

	// figure out where we are in the sequence
	step := seq.NextStep()
	if step == nil {
		ctx.Log.Error(errors.New("got nil NextStep"))
		return
	}

	step.Completed = true

	// start evaluating the result of the function call
	resp := &request.CoordinatedResponse{}
	var runErr rt.RunErr
	var execErr error

	if fnErr != nil {
		if fnRunErr, isRunErr := fnErr.(rt.RunErr); isRunErr {
			// great, it's a runErr
			runErr = fnRunErr
		} else {
			execErr = fnErr
		}
	} else {
		resp = result.(*request.CoordinatedResponse)
	}

	// package everything up and shuttle it back to the parent (atmo-proxy)
	fnr := &sequence.FnResult{
		FQFN:     msg.Type(),
		Key:      step.Exec.CallableFn.Key(), // to support groups, we'll need to find the correct CallableFn in the list
		Response: resp,
		RunErr:   runErr,
		ExecErr: func() string {
			if execErr != nil {
				return execErr.Error()
			}

			return ""
		}(),
	}

	if err := s.sendFnResult(fnr, ctx); err != nil {
		ctx.Log.Error(errors.Wrap(err, "failed to sendFnResult"))
		return
	}

	// determine if we ourselves should continue or halt the sequence
	if execErr != nil {
		ctx.Log.ErrorString("stopping execution after error failed execution of", msg.Type(), ":", execErr.Error())
		return
	}

	if err := seq.HandleStepErrs([]sequence.FnResult{*fnr}, step.Exec); err != nil {
		ctx.Log.Error(err)
		return
	}

	// load the results into the request state
	seq.HandleStepResults([]sequence.FnResult{*fnr})

	// prepare for the next step in the chain to be executed
	stepJSON, err := seq.StepsJSON()
	if err != nil {
		ctx.Log.Error(errors.Wrap(err, "failed to StepsJSON"))
		return
	}

	req.SequenceJSON = stepJSON

	s.sendNextStep(msg, seq, req, ctx)
}

func (s *Sat) sendFnResult(result *sequence.FnResult, ctx *vk.Ctx) error {
	fnrJSON, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal function result")
	}

	respMsg := grav.NewMsgWithParentID(MsgTypeAtmoFnResult, ctx.RequestID(), fnrJSON)

	ctx.Log.Info("function", s.j, "completed, sending result message", respMsg.UUID())

	if s.p.Send(respMsg) == nil {
		return errors.New("failed to Send fnResult")
	}

	return nil
}

func (s *Sat) sendNextStep(msg grav.Message, seq *sequence.Sequence, req *request.CoordinatedRequest, ctx *vk.Ctx) {
	nextStep := seq.NextStep()
	if nextStep == nil {
		ctx.Log.Debug("sequence completed, no nextStep message to send")
		return
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		ctx.Log.Error(errors.Wrap(err, "failed to Marshal request"))
		return
	}

	nextMsg := grav.NewMsgWithParentID(nextStep.Exec.FQFN, ctx.RequestID(), reqJSON)

	ctx.Log.Info("sending next message", nextStep.Exec.FQFN, nextMsg.UUID())

	if err := s.g.Tunnel(nextStep.Exec.FQFN, nextMsg); err != nil {
		// nothing much we can do here
		ctx.Log.Error(errors.Wrap(err, "failed to Tunnel nextMsg"))
	}
}
