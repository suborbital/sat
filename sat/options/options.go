package options

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"

	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/vektor/vlog"
)

type Options struct {
	EnvToken string   `env:"SAT_ENV_TOKEN"`
	Port     port     `env:"SAT_HTTP_PORT"`
	ProcUUID procUUID `env:"SAT_UUID"`

	ControlPlane *ControlPlane `env:",noinit"`
	Ident        *Ident        `env:",noinit"`
	Version      *Version      `env:",noinit"`
	TracerConfig TracerConfig  `env:",prefix=SAT_TRACER_"`
}

type ControlPlane struct {
	Address string `env:"SAT_CONTROL_PLANE"`
}

type Ident struct {
	Data string `env:"SAT_RUNNABLE_IDENT"`
}

type Version struct {
	Data string `env:"SAT_RUNNABLE_VERSION"`
}

// TracerConfig holds values specific to setting up the tracer. It's only used in proxy mode. All configuration options
// have a prefix of SAT_TRACER_ specified in the parent Options struct.
type TracerConfig struct {
	TracerType      string           `env:"TYPE,default=none"`
	ServiceName     string           `env:"SERVICENAME,default=sat"`
	Probability     float64          `env:"PROBABILITY,default=0.5"`
	Collector       *CollectorConfig `env:",prefix=COLLECTOR_,noinit"`
	HoneycombConfig *HoneycombConfig `env:",prefix=HONEYCOMB_,noinit"`
}

// CollectorConfig holds config values specific to the collector tracer exporter running locally / within your cluster.
// All the configuration values here have a prefix of SAT_TRACER_COLLECTOR_, specified in the top level Options struct,
// and the parent TracerConfig struct.
type CollectorConfig struct {
	Endpoint string `env:"ENDPOINT"`
}

// HoneycombConfig holds config values specific to the honeycomb tracer exporter. All the configuration values here have
// a prefix of SAT_TRACER_HONEYCOMB_, specified in the top level Options struct, and the parent TracerConfig struct.
type HoneycombConfig struct {
	Endpoint string `env:"ENDPOINT"`
	APIKey   string `env:"APIKEY"`
	Dataset  string `env:"DATASET"`
}

type Config struct {
	RunnableArg     string
	JobType         string
	PrettyName      string
	Runnable        *directive.Runnable
	Identifier      string
	CapConfig       rcap.CapabilityConfig
	UseStdin        bool
	ControlPlaneUrl string
	Logger          *vlog.Logger
	ProcUUID        string
}

func Resolve(lookuper envconfig.Lookuper) (Options, error) {
	if lookuper == nil {
		lookuper = envconfig.OsLookuper()
	}

	var opts Options

	if err := envconfig.ProcessWith(context.Background(), &opts, lookuper); err != nil {
		return Options{}, errors.Wrap(err, "sat options parsing")
	}

	return opts, nil
}
