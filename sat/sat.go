package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/fqfn"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

type sat struct {
	r    *rt.Reactr
	v    *vk.Server
	g    *grav.Grav
	exec rt.JobFunc
	log  *vlog.Logger
}

// initSat initializes Reactr, Vektor, and Grav instances
// if config.useStdin is true, only Reactr will be created, returning r, nil, nil
func initSat(config *config) *sat {
	r := rt.New()

	logger := vlog.Default(
		vlog.EnvPrefix("SAT"),
	)

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

	exec := r.Register(
		jobName,
		rwasm.NewRunner(config.modulePath),
		rt.Autoscale(0),
		rt.MaxRetries(0),
		rt.RetrySeconds(0),
		rt.PreWarm(),
	)

	if config.useStdin {
		s := &sat{
			r:    r,
			exec: exec,
		}

		return s
	}

	t := websocket.New()
	g := grav.New(
		grav.UseLogger(logger),
		grav.UseTransport(t),
		grav.UseDiscovery(local.New()),
		grav.UseEndpoint(config.portString, "/meta/message"),
	)

	r.Listen(g.Connect(), jobName)

	v := vk.New(
		vk.UseLogger(logger),
		vk.UseAppName(config.runnableName),
		vk.UseHTTPPort(config.port),
		vk.UseEnvPrefix("SAT"),
	)

	v.HandleHTTP(http.MethodGet, "/meta/message", t.HTTPHandlerFunc())
	v.POST("/*any", handler(exec))

	sat := &sat{r, v, g, exec, logger}

	return sat
}

// execFromStdin reads stdin, passes the data through the registered module, and writes the result to stdout
func (s *sat) execFromStdin() error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "failed to scanner.Scan")
	}

	input := scanner.Bytes()

	// construct a fake HTTP request from the input
	req := &request.CoordinatedRequest{
		Method:      http.MethodPost,
		URL:         "/",
		ID:          uuid.New().String(),
		Body:        input,
		Headers:     map[string]string{},
		RespHeaders: map[string]string{},
		Params:      map[string]string{},
		State:       map[string][]byte{},
	}

	result, err := s.exec(req).Then()
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

func handler(exec rt.JobFunc) vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "unknown error")
		}

		result, err := exec(req).Then()
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
