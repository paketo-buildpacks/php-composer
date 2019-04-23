package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Runner interface {
	Run(bin, dir string, args ...string) error
}

type ComposerRunner struct {
	Env map[string]string
	Out io.Writer
	Err io.Writer
}

func (r ComposerRunner) Run(bin, dir string, args ...string) error {
	env := []string{}
	for k, v := range r.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	var cmd *exec.Cmd
	if len(args) > 0 {
		cmd = exec.Command(bin, args...)
	} else {
		cmd = exec.Command(bin)
	}

	cmd.Env = env
	cmd.Dir = dir

	if r.Out != nil {
		cmd.Stdout = io.MultiWriter(os.Stdout, r.Out)
	} else {
		cmd.Stdout = os.Stdout
	}

	if r.Err != nil {
		cmd.Stderr = io.MultiWriter(os.Stderr, r.Err)
	} else {
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}
