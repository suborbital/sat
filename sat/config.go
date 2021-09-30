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
)

var useStdin bool

type config struct {
	modulePath   string
	runnableName string
	port         int
	portString   string
	useStdin     bool
}

func configFromArgs(args []string) (*config, error) {
	flag.Parse()

	if len(args) < 2 {
		return nil, errors.New("missing argument: module path")
	}

	modulePath := args[1]
	if strings.HasPrefix(modulePath, "-") {
		for i := 2; i < len(args); i++ {
			if !strings.HasPrefix(args[i], "-") {
				modulePath = args[i]
				break
			}
		}
	}

	if isURL(modulePath) {
		tmpFile, err := downloadFromURL(modulePath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to downloadFromURL")
		}

		modulePath = tmpFile
	}

	runnableName := strings.TrimSuffix(filepath.Base(modulePath), ".wasm")

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

	c := &config{
		modulePath:   modulePath,
		runnableName: runnableName,
		port:         portInt,
		portString:   port,
		useStdin:     useStdin,
	}

	return c, nil
}

func init() {
	flag.BoolVar(&useStdin, "stdin", false, "read stdin as input, return output to stdout and then terminate")
}
