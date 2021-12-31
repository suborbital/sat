package main

import (
	"log"
	"os"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/sat/constd/exec"
	"github.com/suborbital/vektor/vlog"
)

var atmoPorts = []string{"8080", "8081", "8082"}

type constd struct {
	logger *vlog.Logger
	config *config
	atmo   *watcher
	sats   map[string]*watcher // map of FQFNs to watchers
}

type config struct {
	bundlePath string
	execMode   string
	satTag     string
	atmoTag    string
	atmoCount  int
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to loadConfig"))
	}

	l := vlog.Default(
		vlog.EnvPrefix("CONSTD"),
	)

	c := &constd{
		logger: l,
		config: config,
		atmo:   newWatcher("atmo"),
		sats:   map[string]*watcher{},
	}

	appSource, errchan := startAppSourceServer(config.bundlePath)

	// main event loop
	go func() {
		for {
			c.reconcileAtmo(errchan)
			c.reconcileConstellation(appSource, errchan)

			time.Sleep(time.Second)
		}
	}()

	// assuming nothing above throws an error, this will block forever
	for err := range errchan {
		log.Fatal(errors.Wrap(err, "encountered error"))
	}
}

func (c *constd) reconcileAtmo(errchan chan error) {
	report := c.atmo.report()
	if report == nil {
		c.logger.Info("launching atmo")

		kill, err := exec.Run(
			atmoCommand(c.config, atmoPorts[0]),
			"ATMO_HTTP_PORT="+atmoPorts[0],
			"ATMO_CONTROL_PLANE=localhost:9090",
			"ATMO_LOG_LEVEL=warn",
		)
		if err != nil {
			errchan <- errors.Wrap(err, "failed to Run Atmo")
		}

		c.atmo.add(atmoPorts[0], kill)
	} else {

	}
}

func (c *constd) reconcileConstellation(appSource appsource.AppSource, errchan chan error) {
	runnables := appSource.Runnables()

	for i := range runnables {
		runnable := runnables[i]

		if _, exists := c.sats[runnable.FQFN]; !exists {
			c.sats[runnable.FQFN] = newWatcher(runnable.FQFN)
		}

		watcher := c.sats[runnable.FQFN]

		launch := func() {
			cmd, port := satCommand(c.config, runnable)

			// repeat forever in case the command does error out
			kill, err := exec.Run(
				cmd,
				"SAT_HTTP_PORT="+port,
				"SAT_CONTROL_PLANE=localhost:9090",
				"SAT_LOG_LEVEL=warn",
			)

			if err != nil {
				errchan <- errors.Wrap(err, "sat exited with error")
			}

			watcher.add(port, kill)
		}

		report := watcher.report()
		if report == nil {
			// if no instances exist, launch one
			c.logger.Warn("launching %s\n", runnable.FQFN)

			go launch()
		} else if report.totalThreads/report.instCount >= runtime.NumCPU()/2 {
			// if the current instances seem overwhelmed, add one
			c.logger.Warn("scaling up", runnable.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

			go launch()
		} else if report.totalThreads/report.instCount < 1 {
			// if the current instances have too much spare time on their hands
			c.logger.Warn("scaling down", runnable.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

			watcher.kill()
		}
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
