package sat

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"

	"github.com/suborbital/vektor/vlog"
	"github.com/suborbital/velocity/capabilities"
	"github.com/suborbital/velocity/directive"
	"github.com/suborbital/velocity/fqfn"
	"github.com/suborbital/velocity/server/appsource"
	"github.com/suborbital/velocity/server/options"

	satOptions "github.com/suborbital/sat/sat/options"
)

var useStdin bool

func init() {
	flag.BoolVar(&useStdin, "stdin", false, "read stdin as input, return output to stdout and then terminate")
}

type Config struct {
	RunnableArg     string
	JobType         string
	PrettyName      string
	Runnable        *directive.Runnable
	Identifier      string
	CapConfig       capabilities.CapabilityConfig
	Port            int
	UseStdin        bool
	ControlPlaneUrl string
	EnvToken        string
	Logger          *vlog.Logger
	ProcUUID        string
	TracerConfig    satOptions.TracerConfig
}

type satInfo struct {
	SatVersion string `json:"sat_version"`
}

type app struct {
	Name string `json:"name"`
}

func ConfigFromArgs() (*Config, error) {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		return nil, errors.New("missing argument: runnable (path, URL or FQFN)")
	}

	runnableArg := args[0]

	return ConfigFromRunnableArg(runnableArg)
}

func ConfigFromRunnableArg(runnableArg string) (*Config, error) {
	logger := vlog.Default(
		vlog.EnvPrefix("SAT"),
		vlog.AppMeta(satInfo{SatVersion: SatDotVersion}),
	)

	var runnable *directive.Runnable

	opts, err := satOptions.Resolve(envconfig.OsLookuper())
	if err != nil {
		return nil, errors.Wrap(err, "configFromRunnableArg options.Resolve")
	}

	// first, determine if we need to connect to a control plane
	controlPlane := ""
	useControlPlane := false
	if opts.ControlPlane != nil {
		controlPlane = opts.ControlPlane.Address
		useControlPlane = true
	}

	appClient := appsource.NewHTTPSource(controlPlane)
	caps := capabilities.DefaultConfigWithLogger(logger)

	if useControlPlane {
		// configure the appSource not to wait if the controlPlane isn't available
		atmoOpts := options.Options{Logger: logger, Wait: &wait, Headless: &headless}

		if err = appClient.Start(atmoOpts); err != nil {
			return nil, errors.Wrap(err, "failed to appSource.Start")
		}
	}

	// next, handle the runnable arg being a URL, an FQFN, or a path on disk
	if isURL(runnableArg) {
		logger.Debug("fetching module from URL")
		tmpFile, err := downloadFromURL(runnableArg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to downloadFromURL")
		}

		runnableArg = tmpFile
	} else if FQFN := fqfn.Parse(runnableArg); FQFN.Identifier != "" {
		if useControlPlane {
			logger.Debug("fetching module from control plane")

			cpRunnable, err := appClient.FindRunnable(runnableArg, opts.EnvToken)
			if err != nil {
				return nil, errors.Wrap(err, "failed to FindRunnable")
			}

			runnable = cpRunnable

			rendered, err := appsource.ResolveCapabilitiesFromSource(appClient, FQFN.Identifier, FQFN.Namespace, FQFN.Version, logger)
			if err != nil {
				return nil, errors.Wrap(err, "failed to capabilities.Render")
			}

			caps = rendered
		}
	} else {
		diskRunnable, err := findRunnableDotYaml(runnableArg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to findRunnable")
		}

		if diskRunnable != nil {
			if opts.Ident != nil && opts.Version != nil {
				runnable.FQFN = fqfn.FromParts(opts.Ident.Data, runnable.Namespace, runnable.Name, opts.Version.Data)
			}
		}

		runnable = diskRunnable
	}

	// set some defaults in the case we're not running in an application
	portInt, _ := strconv.Atoi(string(opts.Port))
	jobType := strings.TrimSuffix(filepath.Base(runnableArg), ".wasm")
	FQFN := fqfn.Parse(jobType)
	prettyName := jobType

	// modify configuration if we ARE running as part of an application
	if runnable != nil && runnable.FQFN != "" {
		jobType = runnable.FQFN
		FQFN = fqfn.Parse(runnable.FQFN)

		prettyName = fmt.Sprintf("%s-%s", jobType, opts.ProcUUID[:6])

		// replace the logger with something more detailed
		logger = vlog.Default(
			vlog.EnvPrefix("SAT"),
			vlog.AppMeta(app{prettyName}),
		)

		logger.Debug("configuring", jobType)
		logger.Debug("joining app", FQFN.Identifier)
	} else {
		logger.Debug("configuring", jobType)
	}

	// finally, put it all together
	c := &Config{
		RunnableArg:     runnableArg,
		JobType:         jobType,
		PrettyName:      prettyName,
		Runnable:        runnable,
		Identifier:      FQFN.Identifier,
		CapConfig:       caps,
		Port:            portInt,
		UseStdin:        useStdin,
		ControlPlaneUrl: controlPlane,
		Logger:          logger,
		TracerConfig:    opts.TracerConfig,
		ProcUUID:        string(opts.ProcUUID),
	}

	return c, nil
}

func findRunnableDotYaml(runnableArg string) (*directive.Runnable, error) {
	filename := filepath.Base(runnableArg)
	runnableFilepath := strings.Replace(runnableArg, filename, ".runnable.yml", -1)

	if _, err := os.Stat(runnableFilepath); err != nil {
		// .runnable.yaml doesn't exist, don't bother returning error
		return nil, nil
	}

	runnableBytes, err := os.ReadFile(runnableFilepath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadFile")
	}

	runnable := &directive.Runnable{}
	if err := yaml.Unmarshal(runnableBytes, runnable); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal")
	}

	return runnable, nil
}
