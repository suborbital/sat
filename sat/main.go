package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/vektor/vk"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Fatal("missing argument: module path")
	}

	modulePath := args[1]
	runnableName := strings.TrimSuffix(filepath.Base(modulePath), ".wasm")

	port, ok := os.LookupEnv("SAT_HTTP_PORT")
	if !ok {
		// choose a random port above 1000
		randPort, err := rand.Int(rand.Reader, big.NewInt(10000))
		if err != nil {
			log.Fatal(err)
		}

		port = fmt.Sprintf("%d", randPort.Int64()+1000)
	}

	portInt, _ := strconv.Atoi(port)

	r := rt.New()
	t := websocket.New()
	g := grav.New(
		grav.UseTransport(t),
		grav.UseDiscovery(local.New()),
		grav.UseEndpoint(port, "/meta/message"),
	)

	exec := r.Register(
		runnableName,
		rwasm.NewRunner(modulePath),
		rt.Autoscale(0),
		rt.MaxRetries(0),
		rt.RetrySeconds(0),
		rt.PreWarm(),
	)

	r.Listen(g.Connect(), runnableName)

	v := vk.New(
		vk.UseAppName(runnableName),
		vk.UseHTTPPort(portInt),
		vk.UseEnvPrefix("SAT"),
	)

	v.HandleHTTP(http.MethodGet, "/meta/message", t.HTTPHandlerFunc())

	v.POST("/", func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
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
	})

	if err := v.Start(); err != nil {
		log.Fatal(err)
	}
}
