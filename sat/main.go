package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
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

type sat struct {
	r *rt.Reactr
	v *vk.Server
	g *grav.Grav
}

func main() {
	args := os.Args
	modulePath := args[0]
	runnableName := strings.TrimSuffix(filepath.Base(modulePath), ".wasm")

	// choose a random port above 1000
	randPort, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		log.Fatal(err)
	}

	port := randPort.Int64() + 1000

	r := rt.New()
	t := websocket.New()
	g := grav.New(
		grav.UseTransport(t),
		grav.UseDiscovery(local.New()),
		grav.UseEndpoint(fmt.Sprintf("%d", int(port)), "/meta/message"),
	)

	exec := r.Register(runnableName, rwasm.NewRunner(modulePath), rt.Autoscale(0))
	r.Listen(g.Connect(), runnableName)

	v := vk.New(
		vk.UseAppName(runnableName),
		vk.UseHTTPPort(int(port)),
	)

	v.HandleHTTP(http.MethodGet, "/meta/message", t.HTTPHandlerFunc())

	v.POST("/", func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
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

		return result, nil
	})

	sat := &sat{
		r: r,
		v: v,
		g: g,
	}

	if err := sat.serve(); err != nil {
		log.Fatal(err)
	}
}

func (s *sat) serve() error {
	return s.v.Start()
}
