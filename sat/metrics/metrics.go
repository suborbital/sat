package metrics

import (
	"context"
	"log"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"

	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/sat/sat/options"
)

type Metrics struct {
	FunctionExecutions metrics.Counter
	FunctionTime       metrics.Histogram
}

func SetupMetrics(config options.MetricsConfig, logger *vlog.Logger) error {
	exporter, err := otlpmetricgrpc.New(context.TODO())
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
		log.Fatalln("failed to start the metric controller:", err)
	}

	global.SetMeterProvider(cont)

}
