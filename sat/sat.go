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
	j string // the job name / FQFN

	c      *Config
	v      *vk.Server
	g      *grav.Grav
	p      *grav.Pod
	t      *websocket.Transport
	e      *executor.Executor
	l      *vlog.Logger
	tracer trace.Tracer
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

	exec := executor.NewWithGrav(config.Logger, nil)

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

	sat := &Sat{
		c:      config,
		j:      config.JobType,
		e:      exec,
		t:      websocket.New(),
		l:      config.Logger,
		tracer: traceProvider.Tracer("sat"),
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
	vektorError := make(chan error, 1)

	// start Vektor first so that the server is started up before Grav starts discovery
	go func() {
		if err := s.v.Start(); err != nil {
			vektorError <- err
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

	if err := s.e.ListenAndRun(s.c.JobType, s.handleFnResult); err != nil {
		return errors.Wrap(err, "executor.ListenAndRun")
	}

	if err := connectStaticPeers(s.c.Logger, s.g); err != nil {
		return errors.Wrap(err, "failed to connectStaticPeers")
	}

	// write a file to disk which describes this instance
	info := process.NewInfo(s.c.Port, s.c.JobType)
	if err := info.Write(s.c.ProcUUID); err != nil {
		return errors.Wrap(err, "failed to Write process info")
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
	s.l.Warn("encountered signal, beginning shutdown:", sig.String())

	// stop Grav with a 3s delay between Withdraw and Stop (to allow in-flight requests to drain)
	// s.v.Stop isn't called until all connections are ready to close (after said delay)
	// this is needed to ensure a safe withdraw from the constellation/mesh
	if err := s.g.Withdraw(); err != nil {
		s.l.Warn("encountered error during Withdraw, will proceed:", err.Error())
	}

	time.Sleep(time.Second * 3)

	if err := s.g.Stop(); err != nil {
		s.l.Warn("encountered error during Stop, will proceed:", err.Error())
	}

	if err := process.Delete(s.c.ProcUUID); err != nil {
		s.l.Warn("encountered error during process.Delete, will proceed:", err.Error())
	}

	if err := s.v.StopCtx(ctx); err != nil {
		return errors.Wrap(err, "sat.vektor.StopCtx")
	}

	s.l.Warn("handled signal, continuing shutdown", sig.String())
	return nil
}

// testStart returns Sat's internal server for testing purposes
func (s *Sat) testServer() *vk.Server {
	return s.v
}
