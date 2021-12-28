package exec

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

// Run runs a command, outputting to terminal and returning the full output and/or error
func Run(cmd string, env ...string) error {
	// you can uncomment this below if you want to see exactly the commands being run
	fmt.Println("▶️", cmd)

	command := exec.Command("sh", "-c", cmd)
	command.Env = append(os.Environ(), env...)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return errors.Wrap(err, "failed to Run")
	}

	return nil
}
