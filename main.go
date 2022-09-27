package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/signaler"
	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/sat/sat"
	"github.com/suborbital/sat/sat/metrics"
	"github.com/suborbital/sat/sat/options"
	"github.com/suborbital/sat/sat/process"
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

	mctx, mcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer mcancel()

	mtx, err := metrics.ResolveMetrics(mctx, conf.MetricsConfig)
	if err != nil {
		return errors.Wrap(err, "metrics.ResolveMetrics")
	}

	defer traceProvider.Shutdown(context.Background())

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := sat.New(conf, traceProvider, mtx)
	if err != nil {
		return errors.Wrap(err, "sat.New")
	}

	monitor, err := createProcFile(conf.Logger, conf)
	if err != nil {
		return errors.Wrap(err, "failed to createProcFile")
	}

	signaler := signaler.Setup()

	signaler.Start(s.Start)
	signaler.Start(monitor.Start)

	return signaler.Wait(time.Second * 5)
}

// runStdIn will be called if sat is started up with conf.UseStdin set to true.
func runStdIn(conf *sat.Config) error {
	noopTracer := trace.NewNoopTracerProvider()

	mtx, err := metrics.ResolveMetrics(context.Background(), options.MetricsConfig{Type: "none"})
	if err != nil {
		return errors.Wrap(err, "metrics.ResolveMetrics with noop type")
	}

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := sat.New(conf, noopTracer, mtx)
	if err != nil {
		return errors.Wrap(err, "sat.New")
	}

	if err = s.ExecFromStdin(); err != nil {
		return errors.Wrap(err, "sat.ExecFromStdin")
	}
	return nil
}

type procFileMonitor struct {
	conf *sat.Config
}

func createProcFile(log *vlog.Logger, conf *sat.Config) (*procFileMonitor, error) {
	// write a file to disk which describes this instance
	info := process.NewInfo(conf.Port, conf.JobType)
	if err := info.Write(conf.ProcUUID); err != nil {
		return nil, errors.Wrap(err, "failed to Write process info")
	}

	log.Info("procfile created", conf.ProcUUID)

	return &procFileMonitor{conf}, nil
}

func (p *procFileMonitor) Start(ctx context.Context) error {
	// continually look for the deletion of our procfile
	for {
		if ctx.Err() != nil {
			break
		}

		if _, err := process.Find(p.conf.ProcUUID); err != nil {
			return errors.Wrap(err, "proc file deleted")
		}

		time.Sleep(time.Second)
	}

	return nil
}
