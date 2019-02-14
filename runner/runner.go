package runner

import (
	"os"
	"os/exec"
)

type ComposerRunner struct {
}

func (r ComposerRunner) Run(bin, dir string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
