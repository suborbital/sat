package main

import (
	"log"
)

func main() {
	config, err := configFromArgs()
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

		return
	}

	if err := s.v.Start(); err != nil {
		log.Fatal(err)
	}
}
