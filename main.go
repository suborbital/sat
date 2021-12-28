package main

import (
	"log"

	"github.com/suborbital/sat/sat"
)

func main() {
	config, err := sat.ConfigFromArgs()
	if err != nil {
		log.Fatal(err)
	}

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := sat.New(config)
	if err != nil {
		log.Fatal(err)
	}

	if config.UseStdin {
		if err := s.ExecFromStdin(); err != nil {
			log.Fatal(err)
		}

		return
	}

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
