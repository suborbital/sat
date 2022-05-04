package main

import (
	"fmt"
	"log"

	"github.com/suborbital/sat/sat"
)

func main() {
	// SAT_CONTROL_PLANE=localhost:8080
	// SAT_ENV_TOKEN={compute token}

	config, _ := sat.ConfigFromRunnableArg("com.suborbital......")

	s, _ := sat.New(config, nil)

	for {
		resp, err := s.Exec([]byte("hello"))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(resp.Output)
	}
}
