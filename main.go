package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/sat/sat"
	"github.com/suborbital/sat/sat/metrics"
	"github.com/suborbital/sat/sat/process"
)

func main() {
	conf, err := sat.ConfigFromArgs()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("what are the resolved confs\n\n%#v\n", conf)

	log.Printf("uh, what\n\n%#v", os.Environ())

	if conf.UseStdin {
		if err = runStdIn(conf); err != nil {
			conf.Logger.Error(errors.Wrap(err, "startup in StdIn"))
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err = run(conf); err != nil {
		conf.Logger.Error(errors.Wrap(err, "startup"))
		os.Exit(1)
	}
}

const serverShutdownTimeoutSeconds = 4

// run is called if sat is started up with StdIn mode set to false.
func run(conf *sat.Config) error {
	logger := conf.Logger.CreateScoped("main.run")

	traceProvider, err := sat.SetupTracing(conf.TracerConfig, conf.Logger)
	if err != nil {
		return errors.Wrap(err, "setup tracing")
	}

	logger.Info("setting up metrics")
	err = metrics.SetupMetricsProvider(conf.MetricsConfig, conf.Logger)
	if err != nil {
		return errors.Wrap(err, "SetupMetricsProvider")
	}

	logger.Info("set up metrics")

	defer traceProvider.Shutdown(context.Background())

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := sat.New(conf, traceProvider)
	if err != nil {
		return errors.Wrap(err, "sat.New")
	}

	// Make a channel to listen for an interrupt or terminate signal from the OS. Use a buffered channel because the
	// signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Make a channel to listen for errors coming from the listener. Use a buffered channel so the goroutine can exit if
	// we don't collect this error.
	serverErrors := make(chan error, 1)

	// start sat and its internal reactr/vektor/grav
	go func() {
		serverErrors <- s.Start()
	}()

	// create and scan for our procfile
	go func() {
		if err := createProcFile(logger, conf); err != nil {
			serverErrors <- err
			return
		}

		if err = scanProcFile(logger, conf); err != nil {
			shutdown <- syscall.SIGTERM
		}
	}()

	// block main and wait for shutdown or errors.
	select {
	case err = <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Info("shutdown started from signal", sig.String())
		defer logger.Info("shutdown completed from signal", sig.String())

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeoutSeconds*time.Second)
		defer cancel()

		// Asking listener to shut down and shed load.
		if err = s.Shutdown(ctx, sig); err != nil {
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

// runStdIn will be called if sat is started up with conf.UseStdin set to true.
func runStdIn(conf *sat.Config) error {
	noopTracer := trace.NewNoopTracerProvider()

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := sat.New(conf, noopTracer)
	if err != nil {
		return errors.Wrap(err, "sat.New")
	}

	if err = s.ExecFromStdin(); err != nil {
		return errors.Wrap(err, "sat.ExecFromStdin")
	}
	return nil
}

func createProcFile(log *vlog.Logger, conf *sat.Config) error {
	// write a file to disk which describes this instance
	info := process.NewInfo(conf.Port, conf.JobType)
	log.Info("info to be written", info)
	if err := info.Write(conf.ProcUUID); err != nil {
		return errors.Wrap(err, "failed to Write process info")
	}

	log.Info("procfile created", conf.ProcUUID)

	return nil
}

func scanProcFile(log *vlog.Logger, conf *sat.Config) error {
	// continually look for the deletion of our procfile
	for {
		if _, err := process.Find(conf.ProcUUID); err != nil {
			return errors.Wrap(err, "proc file deleted")
		}

		time.Sleep(time.Second)
	}
}
