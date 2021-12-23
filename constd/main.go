package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/subo/subo/util"
)

type config struct {
	bundlePath string
	execMode   string
	satTag     string
	atmoTag    string
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to loadConfig"))
	}

	appSource, errchan := startAppSourceServer(config.bundlePath)

	startAtmo(config, errchan)

	startConstellation(config, appSource, errchan)

	// assuming nothing above throws an error, this will block forever
	for err := range errchan {
		log.Fatal(errors.Wrap(err, "encountered error"))
	}
}

func startAtmo(config *config, errchan chan error) {
	go func() {
		for {
			// repeat forever in case the command does error out
			if _, err := util.Run(atmoCommand(config)); err != nil {
				errchan <- errors.Wrap(err, "failed to Run Atmo")
			}

			time.Sleep(time.Millisecond * 200)
		}
	}()
}

func startConstellation(config *config, appSource appsource.AppSource, errchan chan error) {
	runnables := appSource.Runnables()

	for i := range runnables {
		runnable := runnables[i]

		go func() {
			fmt.Printf("launching %s\n", runnable.FQFN)

			for {
				// repeat forever in case the command does error out
				if _, err := util.Run(satCommand(config, runnable)); err != nil {
					errchan <- errors.Wrap(err, "sat exited with error")
				}

				time.Sleep(time.Millisecond * 200)
			}
		}()
	}
}

func loadConfig() (*config, error) {
	if len(os.Args) < 2 {
		return nil, errors.New("missing required argument: bundle path")
	}

	bundlePath := os.Args[1]

	satVersion := "latest"
	if version, sExists := os.LookupEnv("CONSTD_SAT_VERSION"); sExists {
		satVersion = version
	}

	atmoVersion := "latest"
	if version, aExists := os.LookupEnv("CONSTD_ATMO_VERSION"); aExists {
		atmoVersion = version
	}

	execMode := "docker"
	if mode, eExists := os.LookupEnv("CONSTD_EXEC_MODE"); eExists {
		execMode = mode
	}

	c := &config{
		bundlePath: bundlePath,
		execMode:   execMode,
		satTag:     satVersion,
		atmoTag:    atmoVersion,
	}

	return c, nil
}
