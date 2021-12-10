package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/coordinator/capabilities"
	"github.com/suborbital/atmo/atmo/coordinator/executor"
	"github.com/suborbital/atmo/atmo/coordinator/sequence"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/fqfn"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/reactr/rwasm/runtime"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const (
	MsgTypeAtmoFnResult = "atmo.fnresult"
)

// sat is a sat server with annoyingly terse field names (because it's smol)
type sat struct {
	j string // the job name / FQFN

	v *vk.Server
	g *grav.Grav
	e *executor.Executor
	l *vlog.Logger
	a appsource.AppSource
}

var wait bool = false

// initSat initializes Reactr, Vektor, and Grav instances
// if config.useStdin is true, only Reactr will be created, returning r, nil, nil
func initSat(config *config) (*sat, error) {
	// logger for Sat, rwasm runtime, and Runnable output
	logger := vlog.Default(
		vlog.EnvPrefix("SAT"),
	)

	runtime.UseInternalLogger(logger)

	// first configure this instance's 'identity'
	jobName := config.runnableName

	if config.runnable != nil {
		ident, iExists := os.LookupEnv("SAT_RUNNABLE_IDENT")
		version, vExists := os.LookupEnv("SAT_RUNNABLE_VERSION")
		if iExists && vExists {
			logger.Debug("configuring with .runnable.yaml")
			jobName = fqfn.FromParts(ident, config.runnable.Namespace, config.runnable.Name, version)
		}
	}

	logger.Debug("registering", jobName)

	// next, determine if config should be fetched from a control plane
	var appSource appsource.AppSource
	caps := rcap.DefaultConfigWithLogger(logger)

	if config.controlPlaneUrl != "" {
		appSource = appsource.NewHTTPSource(config.controlPlaneUrl)

		// configure the appSource not to wait if the controlPlane isn't available
		opts := options.Options{Logger: logger, Wait: &wait}

		if err := appSource.Start(opts); err != nil {
			return nil, errors.Wrap(err, "failed to appSource.Start")
		}

		rendered, err := capabilities.Render(caps, appSource, logger)
		if err != nil {
			return nil, errors.Wrap(err, "failed to capabilities.Render")
		}

		caps = rendered
	}

	exec := executor.NewWithGrav(logger, nil)
	exec.UseCapabilityConfig(caps)

	exec.Register(
		jobName,
		rwasm.NewRunner(config.modulePath),
		rt.Autoscale(0),
		rt.MaxRetries(0),
		rt.RetrySeconds(0),
		rt.PreWarm(),
	)

	sat := &sat{
		j: jobName,
		e: exec,
		l: logger,
		a: appSource,
	}

	// no need to continue setup if we're in stdin mode, so return here
	if config.useStdin {
		return sat, nil
	}

	t := websocket.New()

	g := grav.New(
		grav.UseLogger(logger),
		grav.UseTransport(t),
		grav.UseDiscovery(local.New()),
		grav.UseEndpoint(config.portString, "/meta/message"),
	)

	// set up the Executor to listen for jobs and handle them
	exec.UseGrav(g)
	exec.ListenAndRun(jobName, sat.handleFnResult)

	if err := connectStaticPeers(logger, g); err != nil {
		log.Fatal(err)
	}

	v := vk.New(
		vk.UseLogger(logger),
		vk.UseAppName(config.runnableName),
		vk.UseHTTPPort(config.port),
		vk.UseEnvPrefix("SAT"),
	)

	v.HandleHTTP(http.MethodGet, "/meta/message", t.HTTPHandlerFunc())
	v.POST("/*any", sat.handler(exec))

	sat.v = v
	sat.g = g

	return sat, nil
}

// execFromStdin reads stdin, passes the data through the registered module, and writes the result to stdout
func (s *sat) execFromStdin() error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "failed to scanner.Scan")
	}

	input := scanner.Bytes()

	ctx := vk.NewCtx(s.l, nil, nil)

	// construct a fake HTTP request from the input
	req := &request.CoordinatedRequest{
		Method:      http.MethodPost,
		URL:         "/",
		ID:          ctx.RequestID(),
		Body:        input,
		Headers:     map[string]string{},
		RespHeaders: map[string]string{},
		Params:      map[string]string{},
		State:       map[string][]byte{},
	}

	result, err := s.e.Do(s.j, req, ctx)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	resp := request.CoordinatedResponse{}
	if err := json.Unmarshal(result.([]byte), &resp); err != nil {
		return errors.Wrap(err, "failed to Unmarshal response")
	}

	fmt.Print(string(resp.Output))

	return nil
}

func (s *sat) handler(exec *executor.Executor) vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "unknown error")
		}

		result, err := exec.Do(s.j, req, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to exec"))
			return nil, vk.Wrap(http.StatusTeapot, err)
		}

		resp := request.CoordinatedResponse{}
		if err := json.Unmarshal(result.([]byte), &resp); err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to Unmarshal resp"))
			return nil, vk.E(http.StatusInternalServerError, "unknown error")
		}

		return resp.Output, nil
	}
}

// handleFnResult this is the function mounted onto exec.ListenAndRun, and receives all
// function results received from meshed peers (i.e. Grav)
func (s *sat) handleFnResult(msg grav.Message, result interface{}, fnErr error) {
	s.l.Info(msg.Type(), "finished executing")

	// first unmarshal the request and sequence information
	req, err := request.FromJSON(msg.Data())
	if err != nil {
		s.l.Error(errors.Wrap(err, "failed to request.FromJSON"))
		return
	}

	ctx := vk.NewCtx(s.l, nil, nil)
	ctx.UseRequestID(req.ID)

	seq, err := sequence.FromJSON(req.SequenceJSON, s.e, ctx)
	if err != nil {
		s.l.Error(errors.Wrap(err, "failed to sequence.FromJSON"))
		return
	}

	// load the request into the sequence so that it can help
	// updating state and checking error handling behaviour
	seq.UseRequest(req)

	// figure out where we are in the sequence
	step := seq.NextStep()
	if step == nil {
		s.l.Error(errors.New("got nil NextStep"))
		return
	}

	step.SetCompleted(true)

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
		respJSON := result.([]byte)
		if err := json.Unmarshal(respJSON, resp); err != nil {
			s.l.Error(errors.Wrap(err, "failed to Unmarshal response"))
			return
		}
	}

	// package everything up and shuttle it back to the parent (atmo-proxy)
	fnr := &sequence.FnResult{
		FQFN:     msg.Type(),
		Key:      step.Key(),
		Response: resp,
		RunErr:   runErr,
		ExecErr: func() string {
			if execErr != nil {
				return execErr.Error()
			}

			return ""
		}(),
	}

	pod := s.g.Connect()
	defer pod.Disconnect()

	if err := s.sendFnResult(pod, fnr, ctx); err != nil {
		s.l.Error(errors.Wrap(err, "failed to sendFnResult"))
		return
	}

	// determine if we ourselves should continue or halt the sequence
	if execErr != nil {
		s.l.ErrorString("stopping execution after error failed execution of", msg.Type(), ":", execErr.Error())
		return
	}

	if err := seq.HandleStepErrs([]sequence.FnResult{*fnr}, step); err != nil {
		s.l.Error(err)
		return
	}

	// load the results into the request state
	seq.HandleStepResults([]sequence.FnResult{*fnr})

	// prepare for the next step in the chain to be executed
	stepJSON, err := seq.StepsJSON()
	if err != nil {
		s.l.Error(errors.Wrap(err, "failed to StepsJSON"))
		return
	}

	req.SequenceJSON = stepJSON

	s.sendNextStep(pod, msg, seq, req)
}

func (s *sat) sendFnResult(pod *grav.Pod, result *sequence.FnResult, ctx *vk.Ctx) error {
	fnrJSON, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal function result")
	}

	s.l.Info("function", s.j, "completed, sending result")

	respMsg := grav.NewMsgWithParentID(MsgTypeAtmoFnResult, ctx.RequestID(), fnrJSON)
	pod.Send(respMsg)

	return nil
}

func (s *sat) sendNextStep(pod *grav.Pod, msg grav.Message, seq *sequence.Sequence, req *request.CoordinatedRequest) {
	nextStep := seq.NextStep()
	if nextStep == nil {
		s.l.Info("sequence completed, no nextStep message to send")
		return
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		s.l.Error(errors.Wrap(err, "failed to Marshal request"))
		return
	}

	s.l.Info("sending next message", nextStep.FQFN)

	nextMsg := grav.NewMsgWithParentID(seq.NextStep().FQFN, msg.ParentID(), reqJSON)
	pod.Send(nextMsg)
}
