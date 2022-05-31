package metrics

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/metric/unit"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/suborbital/sat/sat/options"
)

type Metrics struct {
	FunctionExecutions       syncint64.Counter
	FailedFunctionExecutions syncint64.Counter
	FunctionTime             syncint64.Histogram
}

type Timer struct {
	start time.Time
}

func (t Timer) Observe() int64 {
	return time.Now().Sub(t.start).Milliseconds()
}

// NewTimer returns a Timer with the current time stored in it.
func NewTimer() Timer {
	return Timer{start: time.Now()}
}

// SetupMetricsProvider takes in an options.MetricsConfig struct and a logger to put together the structure of getting
// measured data onto the collector. It does not set up the actual meters that we need to use to actually measure
// anything, that's the job of ConfigureMetrics.
//
// This structure is configured to be in the global scope, and that's where all other meters will send their data to be
// picked up and sent off to the collector at specified intervals.
func SetupMetricsProvider(config options.MetricsConfig) error {
	grpcCtx, grpcCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer grpcCtxCancel()

	conn, err := grpc.DialContext(grpcCtx, config.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return errors.Wrap(err, "grpc.DialContext")
	}

	exporter, err := otlpmetricgrpc.New(
		context.TODO(),
		otlpmetricgrpc.WithTimeout(5*time.Second),
		otlpmetricgrpc.WithEndpoint(config.Endpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: 2 * time.Second,
			MaxInterval:     10 * time.Second,
			MaxElapsedTime:  30 * time.Second,
		}),
		otlpmetricgrpc.WithServiceConfig(config.ServiceName),
		otlpmetricgrpc.WithGRPCConn(conn),
	)
	if err != nil {
		return errors.Wrap(err, "otlpmetricgrpc.New")
	}

	cont := controller.New(
		processor.NewFactory(
			simple.NewWithInexpensiveDistribution(),
			exporter,
		),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(3*time.Second),
	)

	if err := cont.Start(context.Background()); err != nil {
		return errors.Wrap(err, "metric controller Start")
	}

	global.SetMeterProvider(cont)

	return nil
}

// ConfigureMetrics returns a struct with the meters that we want to measure in sat. It assumes that the global meter
// has already been set up (see SetupMetricsProvider). Shipping the measured values is the task of the provider, so
// from a usage point of view, nothing else is needed.
func ConfigureMetrics() (Metrics, error) {
	m := global.Meter(
		"sat",
		metric.WithInstrumentationVersion("1.0"),
	)

	si64 := m.SyncInt64()

	functionExecutions, err := si64.Counter(
		"function_executions",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription("How many function execution requests happened"),
	)
	if err != nil {
		return Metrics{}, errors.Wrap(err, "sync int 64 provider function_executions")
	}

	failedFunctionExecutions, err := si64.Counter(
		"failed_function_executions",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription("How many function execution requests failed"),
	)
	if err != nil {
		return Metrics{}, errors.Wrap(err, "sync int 64 provider failed_function_executions")
	}

	functionTime, err := si64.Histogram(
		"function_time",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("How much time was spent doing function executions"),
	)
	if err != nil {
		return Metrics{}, errors.Wrap(err, "sync int 64 provider function_time")
	}

	return Metrics{
		FunctionExecutions:       functionExecutions,
		FailedFunctionExecutions: failedFunctionExecutions,
		FunctionTime:             functionTime,
	}, nil
}
