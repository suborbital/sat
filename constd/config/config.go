package config

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
)

const (
	DefaultControlPlane = "localhost:9090"
)

type Config struct {
	BundlePath   string `env:"bundle_path"`
	ExecMode     string `env:"CONSTD_EXEC_MODE,default=docker"`
	SatTag       string `env:"CONSTD_SAT_VERSION,default=latest"`
	AtmoTag      string `env:"CONSTD_ATMO_VERSION,default=latest"`
	ControlPlane string `env:"CONSTD_CONTROL_PLANE,overwrite"`
	EnvToken     string `env:"CONSTD_ENV_TOKEN"`
	UpstreamHost string `env:"CONSTD_UPSTREAM_HOST"`
	Headless     bool   `env:"CONSTD_HEADLESS,default=false"`
}

// Parse will return a resolved config struct configured by a combination of environment variables and command line
// arguments.
func Parse(args []string) (Config, error) {
	c := Config{
		ControlPlane: DefaultControlPlane,
	}

	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

	if err := envconfig.Process(ctx, &c); err != nil {
		return Config{}, errors.Wrap(err, "resolving config: envconfig.Process")
	}

	if c.ControlPlane == DefaultControlPlane && len(args) < 1 {
		return Config{}, errors.New("missing required argument: bundle path")
	} else if len(args) == 1 {
		c.BundlePath = args[0]
	}

	return c, nil
}
