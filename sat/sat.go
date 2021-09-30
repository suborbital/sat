package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/vektor/vk"
)

type sat struct {
	r    *rt.Reactr
	v    *vk.Server
	g    *grav.Grav
	exec rt.JobFunc
}

// initSat initializes Reactr, Vektor, and Grav instances
// if config.useStdin is true, only Reactr will be created, returning r, nil, nil
func initSat(config *config) *sat {
	r := rt.New()

	exec := r.Register(
		config.runnableName,
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
		grav.UseTransport(t),
		grav.UseDiscovery(local.New()),
		grav.UseEndpoint(config.portString, "/meta/message"),
	)

	r.Listen(g.Connect(), config.runnableName)

	v := vk.New(
		vk.UseAppName(config.runnableName),
		vk.UseHTTPPort(config.port),
		vk.UseEnvPrefix("SAT"),
	)

	v.HandleHTTP(http.MethodGet, "/meta/message", t.HTTPHandlerFunc())
	v.POST("/", handler(exec))

	sat := &sat{r, v, g, exec}

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

	result, err := s.exec(input).Then()
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	output := result.([]byte)

	fmt.Print(string(output))

	return nil
}

func handler(exec rt.JobFunc) vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "unknown error")
		}

		reqJSON, _ := req.ToJSON()

		result, err := exec(reqJSON).Then()
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
