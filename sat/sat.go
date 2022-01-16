package sat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/coordinator/executor"
	"github.com/suborbital/atmo/atmo/coordinator/sequence"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
	wruntime "github.com/suborbital/reactr/rwasm/runtime"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const (
	MsgTypeAtmoFnResult = "atmo.fnresult"
)

// sat is a sat server with annoyingly terse field names (because it's smol)
type Sat struct {
	j string // the job name / FQFN

	c *Config
	v *vk.Server
	g *grav.Grav
	p *grav.Pod
	t *websocket.Transport
	e *executor.Executor
	l *vlog.Logger
}

type loggerScope struct {
	RequestID string `json:"request_id"`
}

var wait bool = false
var headless bool = false

// New initializes Reactr, Vektor, and Grav in a Sat instance
// if config.UseStdin is true, only Reactr will be created
func New(config *Config) (*Sat, error) {
	wruntime.UseInternalLogger(config.Logger)

	exec := executor.NewWithGrav(config.Logger, nil)
	exec.UseCapabilityConfig(config.CapConfig)

	var runner rt.Runnable
	if config.Runnable != nil && len(config.Runnable.ModuleRef.Data) > 0 {
		runner = rwasm.NewRunnerWithRef(config.Runnable.ModuleRef)
	} else {
		runner = rwasm.NewRunner(config.RunnableArg)
	}

	exec.Register(
		config.JobType,
		runner,
		rt.Autoscale(0),
		rt.MaxRetries(0),
		rt.RetrySeconds(0),
		rt.PreWarm(),
	)

	sat := &Sat{
		c: config,
		j: config.JobType,
		e: exec,
		t: websocket.New(),
		l: config.Logger,
	}

	// no need to continue setup if we're in stdin mode, so return here
	if config.UseStdin {
		return sat, nil
	}

	// Grav and Vektor will be started on call to s.Start()

	sat.v = vk.New(
		vk.UseLogger(config.Logger),
		vk.UseAppName(config.PrettyName),
		vk.UseHTTPPort(config.Port),
		vk.UseEnvPrefix("SAT"),
		vk.UseQuietRoutes("/meta/metrics"),
	)

	sat.v.HandleHTTP(http.MethodGet, "/meta/message", sat.t.HTTPHandlerFunc())
	sat.v.GET("/meta/metrics", sat.metricsHandler())
	sat.v.POST("/*any", sat.handler(exec))

	return sat, nil
}

// Start starts Sat's Vektor server and Grav discovery
func (s *Sat) Start() error {
	// start Vektor first so that the server is started up before Grav starts discovery
	go func() {
		if err := s.v.Start(); err != nil {
			if err == http.ErrServerClosed {
				// that's fine, do nothing
			} else {
				log.Fatal(errors.Wrap(err, "failed to Start server"))
			}
		}
	}()

	// configure Grav to join the mesh for its appropriate application
	// and broadcast its "interest" (i.e. the loaded function)
	s.g = grav.New(
		grav.UseBelongsTo(s.c.Identifier),
		grav.UseInterests(s.c.JobType),
		grav.UseLogger(s.c.Logger),
		grav.UseTransport(s.t),
		grav.UseDiscovery(local.New()),
		grav.UseEndpoint(fmt.Sprintf("%d", s.c.Port), "/meta/message"),
	)

	s.p = s.g.Connect()

	// set up the Executor to listen for jobs and handle them
	s.e.UseGrav(s.g)
	s.e.ListenAndRun(s.c.JobType, s.handleFnResult)

	if err := connectStaticPeers(s.c.Logger, s.g); err != nil {
		log.Fatal(err)
	}

	shutdownChan := make(chan error)

	s.setupSignals(shutdownChan)

	return <-shutdownChan
}

// execFromStdin reads stdin, passes the data through the registered module, and writes the result to stdout
func (s *Sat) ExecFromStdin() error {
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

	result, err := s.e.Do(s.j, req, ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	resp := result.(*request.CoordinatedResponse)

	fmt.Print(string(resp.Output))

	return nil
}

func (s *Sat) handler(exec *executor.Executor) vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "unknown error")
		}

		result, err := exec.Do(s.j, req, ctx, nil)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to exec"))
			return nil, vk.Wrap(http.StatusTeapot, err)
		}

		resp := result.(*request.CoordinatedResponse)

		return resp.Output, nil
	}
}

// handleFnResult this is the function mounted onto exec.ListenAndRun, and receives all
// function results received from meshed peers (i.e. Grav)
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

// setupSignals sets up clean shutdown from SIGINT and SIGTERM
func (s *Sat) setupSignals(shutdownChan chan error) {
	sigs := make(chan os.Signal, 64)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		s.l.Warn("encountered signal, beginning shutdown:", sig.String())

		// stopping Grav has a 3s delay (to allow the node to drain)
		// so s.v.Stop isn't called until all connections are ready to close (after that delay)
		// this is needed to ensure safe withdrawl from a constellation
		s.g.Withdraw()

		s.l.Info("withdraw complete")

		err := s.v.Stop()

		s.l.Warn("handled signal, shutdown proceeding", sig.String())

		shutdownChan <- err
	}()
}
