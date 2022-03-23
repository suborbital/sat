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

	"github.com/suborbital/sat/sat"
	"github.com/suborbital/sat/sat/process"
)

func main() {
	config, err := sat.ConfigFromArgs()
	if err != nil {
		log.Fatal(err)
	}

	if err = run(config); err != nil {
		config.Logger.Error(errors.Wrap(err, "startup"))
		os.Exit(1)
	}
}

const shutdownTimeoutSeconds = 3

func run(conf *sat.Config) error {
	localLogger := conf.Logger.CreateScoped("main.run")

	traceProvider, err := sat.SetupTracing(conf.TracerConfig, conf.Logger)
	if err != nil {
		return errors.Wrap(err, "setup tracing")
	}
	defer traceProvider.Shutdown(context.Background())

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := sat.New(conf, traceProvider)
	if err != nil {
		return errors.Wrap(err, "sat.New")
	}

	if conf.UseStdin {
		if err = s.ExecFromStdin(); err != nil {
			return errors.Wrap(err, "sat.ExecFromStdin")
		}
		return nil
	}

	// Make a channel to listen for an interrupt or terminate signal from the OS. Use a buffered channel because the
	// signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Make a channel to listen for errors coming from the listener. Use a buffered channel so the goroutine can exit if
	// we don't collect this error.
	serverErrors := make(chan error, 1)

	go func() {
		localLogger.Info("startup", "sat with configured vektor, grav, reactr started")
		serverErrors <- s.Start()
	}()

	// start scanning for our procfile being deleted
	go func() {
		for {
			if _, err = process.Find(conf.ProcUUID); err != nil {
				localLogger.Warn("proc file deleted, sending termination signal")
				shutdown <- syscall.SIGTERM
				break
			}

			time.Sleep(time.Second)
		}
	}()

	// Blocking main and waiting for shutdown.
	select {
	case err = <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		localLogger.Info("shutdown", "status", "shutdown started", "signal", sig)
		defer localLogger.Info("shutdown", "status", "shutdown complete", "signal", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
		defer cancel()

		// Asking listener to shut down and shed load.
		if err = s.Shutdown(ctx, sig); err != nil {
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
