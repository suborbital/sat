package main

import (
	"log"
	"os"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/sat/constd/exec"
	"github.com/suborbital/vektor/vlog"
)

const (
	atmoPort            = "8080"
	defaultControlPlane = "localhost:9090"
)

type constd struct {
	logger *vlog.Logger
	config *config
	atmo   *watcher
	sats   map[string]*watcher // map of FQFNs to watchers
}

type config struct {
	bundlePath   string
	execMode     string
	satTag       string
	atmoTag      string
	controlPlane string
	envToken     string
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
		atmo:   newWatcher("atmo", l),
		sats:   map[string]*watcher{},
	}

	var appSource appsource.AppSource
	var errchan chan error

	// if an external control plane hasn't been set, act as the control plane
	// but if one has been set, use it (and launch all children with it configured)
	if c.config.controlPlane == defaultControlPlane {
		appSource, errchan = startAppSourceServer(config.bundlePath)
	} else {
		appSource = appsource.NewHTTPSource(c.config.controlPlane)

		if err := appSource.Start(*options.NewWithModifiers()); err != nil {
			log.Fatal(errors.Wrap(err, "failed to appSource.Start"))
		}

		errchan = make(chan error)
	}

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

		uuid, pid, err := exec.Run(
			atmoCommand(c.config, atmoPort),
			"ATMO_HTTP_PORT="+atmoPort,
			"ATMO_CONTROL_PLANE="+c.config.controlPlane,
		)

		if err != nil {
			errchan <- errors.Wrap(err, "failed to Run Atmo")
		}

		c.atmo.add(atmoPort, uuid, pid)
	}
}

func (c *constd) reconcileConstellation(appSource appsource.AppSource, errchan chan error) {
	apps := appSource.Applications()

	for _, app := range apps {
		runnables := appSource.Runnables(app.Identifier, app.AppVersion)

		for i := range runnables {
			runnable := runnables[i]

			if _, exists := c.sats[runnable.FQFN]; !exists {
				c.sats[runnable.FQFN] = newWatcher(runnable.FQFN, c.logger)
			}

			watcher := c.sats[runnable.FQFN]

			launch := func() {
				cmd, port := satCommand(c.config, runnable)

				// repeat forever in case the command does error out
				kill, err := exec.Run(
					cmd,
					"SAT_HTTP_PORT="+port,
					"SAT_CONTROL_PLANE=localhost:9090",
				)

				if err != nil {
					errchan <- errors.Wrap(err, "sat exited with error")
				}

				watcher.add(port, kill)
			}

			// we want to max out at 8 threads per instance
			threshold := runtime.NumCPU() / 2
			if threshold > 8 {
				threshold = 8
			}

			report := watcher.report()
			if report == nil {
				// if no instances exist, launch one
				c.logger.Warn("launching", runnable.FQFN)

				go launch()
			} else if report.totalThreads/report.instCount >= threshold {
				if report.instCount >= runtime.NumCPU() {
					c.logger.Warn("maximum instance count reached for", runnable.Name)
				} else {
					// if the current instances seem overwhelmed, add one
					c.logger.Warn("scaling up", runnable.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

					go launch()
				}
			} else if report.totalThreads/report.instCount < threshold {
				if report.instCount == 1 {
					// that's fine, do nothing
				} else {
					// if the current instances have too much spare time on their hands
					c.logger.Warn("scaling down", runnable.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

					watcher.kill()
				}
			}

			if report != nil {
				for _, p := range report.failedPorts {
					c.logger.Warn("killing instance from failed port", p)

					watcher.killPort(p)
				}
			}
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

	controlPlane := defaultControlPlane
	if cp, eExists := os.LookupEnv("CONSTD_CONTROL_PLANE"); eExists {
		controlPlane = cp
	}

	envToken := ""
	if et, eExists := os.LookupEnv("CONSTD_ENV_TOKEN"); eExists {
		envToken = et
	}

	c := &config{
		bundlePath:   bundlePath,
		execMode:     execMode,
		satTag:       satVersion,
		atmoTag:      atmoVersion,
		controlPlane: controlPlane,
		envToken:     envToken,
	}

	return c, nil
}
