package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/subo/subo/util"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("missing argument: bundle")
	}

	bundlePath := os.Args[1]

	appSource, errchan := startAppSourceServer(bundlePath)

	startAtmo(bundlePath, errchan)

	startConstellation(appSource, errchan)

	// assuming nothing above throws an error, this will block forever
	if err := <-errchan; err != nil {
		log.Fatal(errors.Wrap(err, "failed to startAppSourceServer"))
	}
}

func startAtmo(bundlePath string, errchan chan error) {
	mountPath := filepath.Dir(bundlePath)

	go func() {
		if _, err := util.Run(fmt.Sprintf("docker run -p 8080:8080 -e ATMO_HTTP_PORT=8080 -e ATMO_CONTROL_PLANE=docker.for.mac.localhost:9090 -v %s:/home/atmo --network bridge suborbital/atmo-proxy:dev atmo-proxy", mountPath)); err != nil {
			errchan <- errors.Wrap(err, "failed to Run Atmo")
		}
	}()
}

func startConstellation(appSource appsource.AppSource, errchan chan error) {
	satTag := "latest"
	if tag, exists := os.LookupEnv("CONSTD_SAT_TAG"); exists {
		satTag = tag
	}

	runnables := appSource.Runnables()

	for i := range runnables {
		runnable := runnables[i]

		go func() {
			fmt.Printf("launching %s\n", runnable.FQFN)

			port, err := randPort()
			if err != nil {
				log.Fatal(errors.Wrap(err, "failed to randPort"))
			}

			for {
				_, err := util.Run(fmt.Sprintf(
					"docker run --rm -p %s:%s -e SAT_HTTP_PORT=%s -e SAT_CONTROL_PLANE=docker.for.mac.localhost:9090 --network bridge --name %s suborbital/sat:%s sat %s",
					port, port, port,
					runnable.Name,
					satTag,
					runnable.FQFN,
				))

				if err != nil {
					errchan <- errors.Wrap(err, "sat exited with error")
				}
			}
		}()
	}
}

func randPort() (string, error) {
	// choose a random port above 1000
	randPort, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", errors.Wrap(err, "failed to rand.Int")
	}

	return fmt.Sprintf("%d", randPort.Int64()+10000), nil
}
