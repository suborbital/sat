package sat

import (
	"log"
	"os"
)

const (
	Success int = iota
	RuntimeError
	RunnableError
)

func Fatal(err error) {
	// still adds timestamp, might not want that for parsing reasons from SCN's perspective
	log.Println(err)

	os.Exit(int(RuntimeError))
}
