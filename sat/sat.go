package sat

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/atmo/atmo/coordinator/executor"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
	wruntime "github.com/suborbital/reactr/rwasm/runtime"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/sat/sat/process"
)

const (
	MsgTypeAtmoFnResult = "atmo.fnresult"
)

// Sat is a sat server with annoyingly terse field names (because it's smol)
type Sat struct {
	jobName string // the job name / FQFN

	config    *Config
	vektor    *vk.Server
	grav      *grav.Grav
	pod       *grav.Pod
	transport *websocket.Transport
	exec      *executor.Executor
	log       *vlog.Logger
	tracer    trace.Tracer
}

type loggerScope struct {
	RequestID string `json:"request_id"`
}

var wait = false
var headless = false

// New initializes Reactr, Vektor, and Grav in a Sat instance
// if config.UseStdin is true, only Reactr will be created
func New(config *Config, traceProvider trace.TracerProvider) (*Sat, error) {
	wruntime.UseInternalLogger(config.Logger)

	exec := executor.NewWithGrav(config.Logger, nil, nil)

	var runner rt.Runnable
	if config.Runnable != nil && len(config.Runnable.ModuleRef.Data) > 0 {
		runner = rwasm.NewRunnerWithRef(config.Runnable.ModuleRef)
	} else {
		runner = rwasm.NewRunner(config.RunnableArg)
	}

	err := exec.Register(
		config.JobType,
		runner,
		&config.CapConfig,
		rt.Autoscale(24),
		rt.MaxRetries(0),
		rt.RetrySeconds(0),
		rt.PreWarm(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "exec.Register")
	}

	var transport *websocket.Transport
	if config.ControlPlaneUrl != "" {
		transport = websocket.New()
	}

	sat := &Sat{
		jobName:   config.JobType,
		config:    config,
		transport: transport,
		exec:      exec,
		log:       config.Logger,
	}

	if traceProvider != nil {
		sat.tracer = traceProvider.Tracer("sat")
	}

	// no need to continue setup if we're in stdin mode, so return here
	if config.UseStdin {
		return sat, nil
	}

	// Grav and Vektor will be started on call to s.Start()
	sat.vektor = vk.New(
		vk.UseLogger(config.Logger),
		vk.UseAppName(config.PrettyName),
		vk.UseHTTPPort(config.Port),
		vk.UseEnvPrefix("SAT"),
		vk.UseQuietRoutes("/meta/metrics"),
	)

	// if a transport is configured, enable grav and metrics endpoints, otherwise enable server mode
	if sat.transport != nil {
		sat.vektor.HandleHTTP(http.MethodGet, "/meta/message", sat.transport.HTTPHandlerFunc())
		sat.vektor.GET("/meta/metrics", sat.metricsHandler())
	} else {
		// allow any HTTP method
		sat.vektor.GET("/*any", sat.handler(exec))
		sat.vektor.POST("/*any", sat.handler(exec))
		sat.vektor.PATCH("/*any", sat.handler(exec))
		sat.vektor.DELETE("/*any", sat.handler(exec))
		sat.vektor.HEAD("/*any", sat.handler(exec))
		sat.vektor.OPTIONS("/*any", sat.handler(exec))
	}

	return sat, nil
}

// Start starts Sat's Vektor server and Grav discovery
func (s *Sat) Start() error {
	vektorError := make(chan error, 1)

	// start Vektor first so that the server is started up before Grav starts discovery
	go func() {
		if err := s.vektor.Start(); err != nil {
			vektorError <- err
		}
	}()

	if s.transport != nil {
		// configure Grav to join the mesh for its appropriate application
		// and broadcast its "interest" (i.e. the loaded function)
		s.grav = grav.New(
			grav.UseBelongsTo(s.config.Identifier),
			grav.UseInterests(s.config.JobType),
			grav.UseLogger(s.config.Logger),
			grav.UseTransport(s.transport),
			grav.UseDiscovery(local.New()),
			grav.UseEndpoint(fmt.Sprintf("%d", s.config.Port), "/meta/message"),
		)

		s.pod = s.grav.Connect()

		// set up the Executor to listen for jobs and handle them
		s.exec.UseGrav(s.grav)

		if err := s.exec.ListenAndRun(s.config.JobType, s.handleFnResult); err != nil {
			return errors.Wrap(err, "executor.ListenAndRun")
		}

		if err := connectStaticPeers(s.config.Logger, s.grav); err != nil {
			return errors.Wrap(err, "failed to connectStaticPeers")
		}
	}

	select {
	case err := <-vektorError:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "failed to start server")
		}
	}

	return nil
}

func (s *Sat) Shutdown(ctx context.Context, sig os.Signal) error {
	// stop Grav with a 3s delay between Withdraw and Stop (to allow in-flight requests to drain)
	// s.vektor.Stop isn't called until all connections are ready to close (after said delay)
	// this is needed to ensure a safe withdraw from the constellation/mesh

	if s.transport != nil {
		if err := s.grav.Withdraw(); err != nil {
			s.log.Warn("encountered error during Withdraw, will proceed:", err.Error())
		}

		time.Sleep(time.Second * 3)

		if err := s.grav.Stop(); err != nil {
			s.log.Warn("encountered error during Stop, will proceed:", err.Error())
		}
	}

	if err := process.Delete(s.config.ProcUUID); err != nil {
		s.log.Warn("encountered error during process.Delete, will proceed:", err.Error())
	}

	if err := s.vektor.StopCtx(ctx); err != nil {
		return errors.Wrap(err, "sat.vektor.StopCtx")
	}

	s.log.Warn("handled signal, continuing shutdown", sig.String())
	return nil
}

// testStart returns Sat's internal server for testing purposes
func (s *Sat) testServer() *vk.Server {
	return s.vektor
}
