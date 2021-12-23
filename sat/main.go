package main

import (
	"log"

	"github.com/suborbital/vektor/vlog"
)

func main() {
	// logger for Sat, rwasm runtime, and Runnable output
	logger := vlog.Default(
		vlog.EnvPrefix("SAT"),
	)

	config, err := configFromArgs(logger)
	if err != nil {
		log.Fatal(err)
	}

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := initSat(logger, config)
	if err != nil {
		log.Fatal(err)
	}

	if config.useStdin {
		if err := s.execFromStdin(); err != nil {
			log.Fatal(err)
		}

		return
	}

	if err := s.v.Start(); err != nil {
		log.Fatal(err)
	}
}
