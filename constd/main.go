package main

import (
	"log"
	"os"
	"runtime"
	"time"

	"github.com/pkg/errors"

	// company packages.
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/sat/constd/config"
	"github.com/suborbital/sat/constd/exec"
)

const (
	atmoPort = "8080"
)

type constd struct {
	logger *vlog.Logger
	config config.Config
	atmo   *watcher
	sats   map[string]*watcher // map of FQFNs to watchers
}

func main() {
	conf, err := config.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to loadConfig"))
	}

	l := vlog.Default(
		vlog.EnvPrefix("CONSTD"),
	)

	c := &constd{
		logger: l,
		config: conf,
		atmo:   newWatcher("atmo", l),
		sats:   map[string]*watcher{},
	}

	var appSource appsource.AppSource
	var errChan chan error

	// if an external control plane hasn't been set, act as the control plane
	// but if one has been set, use it (and launch all children with it configured)
	if c.config.ControlPlane == config.DefaultControlPlane {
		appSource, errChan = startAppSourceServer(c.config.BundlePath)
	} else {
		appSource = appsource.NewHTTPSource(c.config.ControlPlane)

		if err := startAppSourceWithRetry(l, appSource); err != nil {
			log.Fatal(errors.Wrap(err, "failed to startAppSourceHTTPClient"))
		}

		if err := registerWithControlPlane(c.config); err != nil {
			log.Fatal(errors.Wrap(err, "failed to registerWithControlPlane"))
		}

		errChan = make(chan error)
	}

	// main event loop
	go func() {
		for {
			c.reconcileAtmo(errChan)
			c.reconcileConstellation(appSource, errChan)

			time.Sleep(time.Second)
		}
	}()

	// assuming nothing above throws an error, this will block forever
	for err := range errChan {
		log.Fatal(errors.Wrap(err, "encountered error"))
	}
}

func (c *constd) reconcileAtmo(errChan chan error) {
	report := c.atmo.report()
	if report == nil {
		c.logger.Info("launching atmo")

		uuid, pid, err := exec.Run(
			atmoCommand(c.config, atmoPort),
			"ATMO_HTTP_PORT="+atmoPort,
			"ATMO_CONTROL_PLANE="+c.config.controlPlane,
			"ATMO_ENV_TOKEN="+c.config.envToken,
			"ATMO_HEADLESS=true",
		)

		if err != nil {
			errChan <- errors.Wrap(err, "failed to Run Atmo")
		}

		c.atmo.add(atmoPort, uuid, pid)
	}
}

func (c *constd) reconcileConstellation(appSource appsource.AppSource, errChan chan error) {
	apps := appSource.Applications()

	for _, app := range apps {
		runnables := appSource.Runnables(app.Identifier, app.AppVersion)

		for i := range runnables {
			runnable := runnables[i]

			c.logger.Debug("reconciling", runnable.FQFN)

			if _, exists := c.sats[runnable.FQFN]; !exists {
				c.sats[runnable.FQFN] = newWatcher(runnable.FQFN, c.logger)
			}

			satWatcher := c.sats[runnable.FQFN]

			launch := func() {
				cmd, port := satCommand(c.config, runnable)

				// repeat forever in case the command does error out
				uuid, pid, err := exec.Run(
					cmd,
					"SAT_HTTP_PORT="+port,
					"SAT_ENV_TOKEN="+c.config.envToken,
					"SAT_CONTROL_PLANE="+c.config.controlPlane,
				)

				if err != nil {
					errChan <- errors.Wrap(err, "sat exited with error")
				}

				satWatcher.add(port, uuid, pid)
			}

			// we want to max out at 8 threads per instance
			threshold := runtime.NumCPU() / 2
			if threshold > 8 {
				threshold = 8
			}

			report := satWatcher.report()
			if report == nil {
				// if no instances exist, launch one
				c.logger.Warn("launching", runnable.FQFN)

				go launch()
			} else if report.instCount > 0 && report.totalThreads/report.instCount >= threshold {
				if report.instCount >= runtime.NumCPU() {
					c.logger.Warn("maximum instance count reached for", runnable.Name)
				} else {
					// if the current instances seem overwhelmed, add one
					c.logger.Warn("scaling up", runnable.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

					go launch()
				}
			} else if report.instCount > 0 && report.totalThreads/report.instCount < threshold {
				if report.instCount == 1 {
					// that's fine, do nothing
				} else {
					// if the current instances have too much spare time on their hands
					c.logger.Warn("scaling down", runnable.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

					satWatcher.terminate()
				}
			}

			if report != nil {
				for _, p := range report.failedPorts {
					c.logger.Warn("killing instance from failed port", p)

					satWatcher.terminateInstance(p)
				}
			}
		}
	}
}
