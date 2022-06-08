package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/sat/sat"
	"github.com/suborbital/sat/sat/process"
	"github.com/suborbital/velocity/signaler"
)

func main() {
	conf, err := sat.ConfigFromArgs()
	if err != nil {
		log.Fatal(err)
	}

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

// run is called if sat is started up with StdIn mode set to false.
func run(conf *sat.Config) error {
	traceProvider, err := sat.SetupTracing(conf.TracerConfig, conf.Logger)
	if err != nil {
		return errors.Wrap(err, "setup tracing")
	}

	defer traceProvider.Shutdown(context.Background())

	s, err := sat.New(conf, traceProvider)
	if err != nil {
		return errors.Wrap(err, "sat.New")
	}

	signaler := signaler.Setup()

	signaler.Start(s.Start)

	signaler.Start(monitorProcfile(conf))

	return signaler.Wait(time.Second * 5)
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

func monitorProcfile(conf *sat.Config) func(context.Context) error {
	return func(ctx context.Context) error {
		// write a file to disk which describes this instance
		info := process.NewInfo(conf.Port, conf.JobType)

		if err := info.Write(conf.ProcUUID); err != nil {
			return errors.Wrap(err, "failed to Write process info")
		}

		for {
			if err := ctx.Err(); err != nil {
				return nil
			}

			if _, err := process.Find(conf.ProcUUID); err != nil {
				return errors.Wrap(err, "proc file deleted")
			}

			time.Sleep(time.Second)
		}

		return nil
	}
}
