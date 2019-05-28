package runner

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudfoundry/libcfbuildpack/logger"
)

type Runner interface {
	Run(bin, dir string, args ...string) error
	RunWithOutput(bin, dir string, args ...string) (string, error)
}

type ComposerRunner struct {
	Logger logger.Logger
	Out    io.Writer
	Err    io.Writer
}

func (r ComposerRunner) Run(bin, dir string, args ...string) error {
	var cmd *exec.Cmd
	if len(args) > 0 {
		r.Logger.Debug("Running `%s %s` from directory '%s'", bin, strings.Join(args, " "), dir)
		cmd = exec.Command(bin, args...)
	} else {
		r.Logger.Debug("Running `%s` from directory '%s'", bin, dir)
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

func (r ComposerRunner) RunWithOutput(bin, dir string, args ...string) (string, error) {
	var cmd *exec.Cmd
	if len(args) > 0 {
		r.Logger.Debug("Running `%s %s` from directory '%s'", bin, strings.Join(args, " "), dir)
		cmd = exec.Command(bin, args...)
	} else {
		r.Logger.Debug("Running `%s` from directory '%s'", bin, dir)
		cmd = exec.Command(bin)
	}

	cmd.Dir = dir
	buf := bytes.Buffer{}
	cmd.Stdout = &buf

	if r.Err != nil {
		cmd.Stderr = io.MultiWriter(os.Stderr, r.Err)
	} else {
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	// this is on purpose, we return whatever is in the buffer regardless of an error occurring
	//  this defers handling of the error to the caller, see CheckPlatformReqs in composer.go
	return buf.String(), err
}

type FakeRunner struct {
	Arguments []string
	Cwd       string
	Out       *bytes.Buffer
	Err       error
}

func (f *FakeRunner) Run(bin, dir string, args ...string) error {
	f.Arguments = append([]string{bin}, args...)
	f.Cwd = dir
	return f.Err
}

func (f *FakeRunner) RunWithOutput(bin, dir string, args ...string) (string, error) {
	f.Arguments = append([]string{bin}, args...)
	f.Cwd = dir
	return f.Out.String(), f.Err
}