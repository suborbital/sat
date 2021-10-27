package util

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// Run runs a command, outputting to terminal and returning the full output and/or error
func Run(cmd string) (string, error) {
	return RunInDir(cmd, "")
}

// RunInDir runs a command in the specified directory and returns the full output or error
func RunInDir(cmd, dir string) (string, error) {
	// you can uncomment this below if you want to see exactly the commands being run
	// fmt.Println("▶️", cmd)

	argLRaw := strings.Split(cmd, " ")

	argL := []string{}
	for i := 1; i < len(argLRaw); i++ {
		arg := argLRaw[i]

		// if the argument ends in \ , assume we're escaping a space and join it with the next arg
		if strings.HasSuffix(arg, "\\") {
			arg = strings.TrimSuffix(arg, "\\")
			arg = arg + " " + argLRaw[i+1]
			i++
		}

		argL = append(argL, arg)
	}

	command := exec.Command(argLRaw[0], argL...)

	command.Dir = dir

	var outBuf bytes.Buffer
	command.Stdout = io.MultiWriter(os.Stdout, &outBuf)
	command.Stderr = io.MultiWriter(os.Stderr, &outBuf)

	runErr := command.Run()

	outStr := outBuf.String()

	if runErr != nil {
		return outStr, errors.Wrap(runErr, "failed to Run command")
	}

	return outStr, nil
}
