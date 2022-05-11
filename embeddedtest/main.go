package main

import (
	"fmt"
	"log"

	"github.com/suborbital/sat/sat"
)

// Set the following environment variables before running
// 	SAT_CONTROL_PLANE={Control Plane URL}
// 	SAT_ENV_TOKEN={Environment token}
//
// Replace Fully Qualified Function Name in sat.ConfigFromRunnableArg to load a function
// See also https://docs.suborbital.dev/compute/concepts/fully-qualified-function-names
func main() {
	config, _ := sat.ConfigFromRunnableArg("com.suborbital.acmeco#default::embed@v1.0.0")

	s, _ := sat.New(config, nil)

	for i := 1; i < 100; i++ {
		resp, err := s.Exec([]byte("world!"))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%s\n", resp.Output)
	}
}
