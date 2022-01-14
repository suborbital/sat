package main

import (
	"github.com/suborbital/reactr/api/tinygo/runnable"
)

type Goodbye struct{}

func (h Goodbye) Run(input []byte) ([]byte, error) {
	return []byte("Goodbye, " + string(input)), nil
}

// initialize runnable, do not edit //
func main() {
	runnable.Use(Goodbye{})
}
