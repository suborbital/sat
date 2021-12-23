package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/coordinator/capabilities"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/atmo/fqfn"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/vektor/vlog"
	"gopkg.in/yaml.v2"
)

var useStdin bool

type config struct {
	runnableArg     string
	runnableName    string
	runnable        *directive.Runnable
	capConfig       rcap.CapabilityConfig
	port            int
	portString      string
	useStdin        bool
	controlPlaneUrl string
}

func configFromArgs(logger *vlog.Logger) (*config, error) {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		return nil, errors.New("missing argument: runnable (path, URL or FQFN)")
	}

	runnableArg := args[0]

	var runnable *directive.Runnable

	// first, determine if we need to connect to a control plane
	controlPlane, useControlPlane := os.LookupEnv("SAT_CONTROL_PLANE")
	appClient := appsource.NewHTTPSource(controlPlane)
	caps := rcap.DefaultConfigWithLogger(logger)

	if useControlPlane {
		// configure the appSource not to wait if the controlPlane isn't available
		opts := options.Options{Logger: logger, Wait: &wait, Headless: &headless}

		if err := appClient.Start(opts); err != nil {
			return nil, errors.Wrap(err, "failed to appSource.Start")
		}

		rendered, err := capabilities.Render(caps, appClient, logger)
		if err != nil {
			return nil, errors.Wrap(err, "failed to capabilities.Render")
		}

		caps = rendered
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

			cpRunnable, err := appClient.FindRunnable(runnableArg, "")
			if err != nil {
				return nil, errors.Wrap(err, "failed to FindRunnable")
			}

			runnable = cpRunnable
		}
	} else {
		diskRunnable, err := findRunnableDotYaml(runnableArg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to findRunnable")
		}

		if diskRunnable != nil {
			ident, iExists := os.LookupEnv("SAT_RUNNABLE_IDENT")
			version, vExists := os.LookupEnv("SAT_RUNNABLE_VERSION")
			if iExists && vExists {
				FQFN := fqfn.FromParts(ident, runnable.Namespace, runnable.Name, version)
				runnable.FQFN = FQFN
			}
		}

		runnable = diskRunnable
	}

	// next, figure out the configuration of the HTTP server
	port, ok := os.LookupEnv("SAT_HTTP_PORT")
	if !ok {
		// choose a random port above 1000
		randPort, err := rand.Int(rand.Reader, big.NewInt(10000))
		if err != nil {
			return nil, errors.Wrap(err, "failed to rand.Int")
		}

		port = fmt.Sprintf("%d", randPort.Int64()+1000)
	}

	portInt, _ := strconv.Atoi(port)

	runnableName := strings.TrimSuffix(filepath.Base(runnableArg), ".wasm")

	// finally, put it all together
	c := &config{
		runnableArg:     runnableArg,
		runnableName:    runnableName,
		runnable:        runnable,
		capConfig:       caps,
		port:            portInt,
		portString:      port,
		useStdin:        useStdin,
		controlPlaneUrl: controlPlane,
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

func init() {
	flag.BoolVar(&useStdin, "stdin", false, "read stdin as input, return output to stdout and then terminate")
}
