package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/sat/constd/config"
	"github.com/suborbital/sat/constd/exec"
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

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	c := &constd{
		logger: l,
		config: conf,
		atmo:   newWatcher("atmo", l),
		sats:   map[string]*watcher{},
	}

	appSource, errChan := c.setupAppSource()

	wg := sync.WaitGroup{}

	wg.Add(2)
loop:
	for {
		select {
		case <-shutdown:
			c.logger.Info("terminating constd")
			break loop
		case err = <-errChan:
			c.logger.Error(err)
			log.Fatal(errors.Wrap(err, "encountered error"))
		default:
			break
		}

		c.reconcileAtmo(errChan)
		c.reconcileConstellation(appSource, errChan)

		time.Sleep(time.Second)
	}

	l.Info("shutting down")

	for _, s := range c.sats {
		err = s.terminate()
		if err != nil {
			log.Fatal("terminating sats failed", err)
		}
	}
	wg.Done()

	err = c.atmo.terminate()
	if err != nil {
		log.Fatal("terminating atmo failed", err)
	}
	wg.Done()
	wg.Wait()
	l.Info("shutdown complete")
}

func (c *constd) reconcileAtmo(errChan chan error) {
	report := c.atmo.report()
	if report != nil {
		return
	}

	c.logger.Info("launching atmo")

	atmoEnv := []string{
		"ATMO_HTTP_PORT=" + c.config.AtmoPort,
		"ATMO_CONTROL_PLANE=" + c.config.ControlPlane,
		"ATMO_ENV_TOKEN=" + c.config.EnvToken,
	}

	if c.config.Headless {
		atmoEnv = append(atmoEnv, "ATMO_HEADLESS=true")
	}

	uuid, pid, err := exec.Run(
		atmoCommand(c.config, c.config.AtmoPort),
		atmoEnv...,
	)

	if err != nil {
		errChan <- errors.Wrap(err, "failed to Run Atmo")
	}

	c.atmo.add("atmo-proxy", c.config.AtmoPort, uuid, pid)
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
				c.logger.Info("launching sat (", runnable.FQFN, ")")

				cmd, port := satCommand(c.config, runnable)

				// repeat forever in case the command does error out
				uuid, pid, err := exec.Run(
					cmd,
					"SAT_HTTP_PORT="+port,
					"SAT_ENV_TOKEN="+c.config.EnvToken,
					"SAT_CONTROL_PLANE="+c.config.ControlPlane,
				)

				if err != nil {
					c.logger.Error(errors.Wrapf(err, "failed to exec.Run sat ( %s )", runnable.FQFN))
					return
				}

				satWatcher.add(runnable.FQFN, port, uuid, pid)

				c.logger.Info("successfully started sat (", runnable.FQFN, ") on port", port)
			}

			// we want to max out at 8 threads per instance
			threshold := runtime.NumCPU() / 2
			if threshold > 8 {
				threshold = 8
			}

			report := satWatcher.report()
			if report == nil || report.instCount == 0 {
				// if no instances exist, launch one
				c.logger.Warn("no instances exist for", runnable.FQFN)

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

					satWatcher.scaleDown()
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

func (c *constd) setupAppSource() (appsource.AppSource, chan error) {
	// if an external control plane hasn't been set, act as the control plane
	// but if one has been set, use it (and launch all children with it configured)
	if c.config.ControlPlane == config.DefaultControlPlane {
		return startAppSourceServer(c.config.BundlePath)
	}

	appSource := appsource.NewHTTPSource(c.config.ControlPlane)

	if err := startAppSourceWithRetry(c.logger, appSource); err != nil {
		log.Fatal(errors.Wrap(err, "failed to startAppSourceHTTPClient"))
	}

	if err := registerWithControlPlane(c.config); err != nil {
		log.Fatal(errors.Wrap(err, "failed to registerWithControlPlane"))
	}

	errChan := make(chan error)

	return appSource, errChan
}
