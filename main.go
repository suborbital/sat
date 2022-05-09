package main

import (
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/suborbital/sat/sat"
)

func main() {
	config, err := sat.ConfigFromArgs()
	if err != nil {
		sat.Fatal(err)
	}

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := sat.New(config)
	if err != nil {
		sat.Fatal(err)
	}

	if config.UseStdin {
		if err := s.ExecFromStdin(); err != nil {
			os.Exit(sat.RuntimeError)
		}
		//be explicit about it and
		os.Exit(sat.Success) //this is identical to the return
	}

	if err := s.Start(); err != nil {
		if err == http.ErrServerClosed {
			config.Logger.Info("sat server shutdown complete")
		} else {
			config.Logger.Error(errors.Wrap(err, "sat error, dirty shutdown complete"))
		}
	} else {
		config.Logger.Info("sat clean shutdown complete")
	}
}
