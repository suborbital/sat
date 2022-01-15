package exec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

// Run runs a command, outputting to terminal and returning the full output and/or error
// a channel is returned which, when sent on, will terminate the process that was started
func Run(cmd string, env ...string) (chan bool, error) {
	// you can uncomment this below if you want to see exactly the commands being run
	fmt.Println("▶️", cmd)

	parts := strings.Split(cmd, " ")

	command := exec.Command(parts[0], parts[1:]...)
	command.Env = append(os.Environ(), env...)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to Run")
	}

	killChan := make(chan bool)

	go func() {
		<-killChan

		command.Process.Signal(syscall.SIGTERM)

		time.Sleep(time.Second * 3)

		command.Process.Kill()
	}()

	return killChan, nil
}
