package main

import (
	"log"
	"os"
)

func main() {
	config, err := configFromArgs(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := initSat(config)
	if err != nil {
		log.Fatal(err)
	}

	if config.useStdin {
		if err := s.execFromStdin(); err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
	}

	if err := s.v.Start(); err != nil {
		log.Fatal(err)
	}
}
