package runner

import (
	"io"
	"os"
	"os/exec"
)

type Runner interface {
	Run(bin, dir string, args ...string) error
}

type ComposerRunner struct {
	Out io.Writer
	Err io.Writer
}

func (r ComposerRunner) Run(bin, dir string, args ...string) error {
	var cmd *exec.Cmd
	if len(args) > 0 {
		cmd = exec.Command(bin, args...)
	} else {
		cmd = exec.Command(bin)
	}

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

type FakeRunner struct {
	Arguments []string
	Cwd       string
	Err       error
}

func (f *FakeRunner) Run(bin, dir string, args ...string) error {
	f.Arguments = append([]string{bin}, args...)
	f.Cwd = dir
	return f.Err
}