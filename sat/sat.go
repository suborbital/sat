package sat

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pkg/errors"

	// company packages.
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

var wait = false
var headless = false

// New initializes Reactr, Vektor, and Grav in a Sat instance
// if config.UseStdin is true, only Reactr will be created
func New(config *Config) (*Sat, error) {
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
		log.Fatal(errors.Wrap(err, "failed to connectStaticPeers"))
	}

	// write a file to disk which describes this instance
	info := process.NewInfo(s.c.Port, s.c.JobType)
	if err := info.Write(s.c.ProcUUID); err != nil {
		log.Fatal(errors.Wrap(err, "failed to Write process info"))
	}

	shutdownChan := make(chan error)

	s.setupSignals(shutdownChan)

	return <-shutdownChan
}

// testStart returns Sat's internal server for testing purposes
func (s *Sat) testServer() *vk.Server {
	return s.v
}
