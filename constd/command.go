package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/directive"
)

func atmoCommand(config *config) string {
	var cmd string

	switch config.execMode {
	case "docker":
		cmd = fmt.Sprintf(
			"docker run -p 8080:8080 -e ATMO_HTTP_PORT=8080 -e ATMO_CONTROL_PLANE=docker.for.mac.localhost:9090 --network bridge suborbital/atmo-proxy:%s atmo-proxy",
			config.atmoTag,
		)
	case "metal":
		os.Setenv("ATMO_HTTP_PORT", "8080")
		os.Setenv("ATMO_CONTROL_PLANE", "localhost:9090")

		cmd = "atmo-proxy"
	default:
		cmd = "echo 'invalid exec mode'"
	}

	return cmd
}

// satCommand returns the command and the port string
func satCommand(config *config, runnable directive.Runnable) (string, string) {
	port, err := randPort()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to randPort"))
	}

	var cmd string

	switch config.execMode {
	case "docker":
		cmd = fmt.Sprintf(
			"docker run --rm -p %s:%s -e SAT_HTTP_PORT=%s -e SAT_CONTROL_PLANE=docker.for.mac.localhost:9090 --network bridge suborbital/sat:%s sat %s",
			port, port, port,
			config.satTag,
			runnable.FQFN,
		)
	case "metal":
		cmd = fmt.Sprintf(
			"sat %s",
			runnable.FQFN,
		)
	default:
		cmd = "echo 'invalid exec mode'"
	}

	return cmd, port
}

func randPort() (string, error) {
	// choose a random port above 1000
	randPort, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", errors.Wrap(err, "failed to rand.Int")
	}

	return fmt.Sprintf("%d", randPort.Int64()+10000), nil
}
