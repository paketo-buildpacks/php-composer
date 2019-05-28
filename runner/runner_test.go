package runner

import (
	"bytes"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec/report"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func TestUnitRunner(t *testing.T) {
	spec.Run(t, "Runner", testRunner, spec.Report(report.Terminal{}))
}

func testRunner(t *testing.T, when spec.G, it spec.S) {
	var f *test.BuildFactory

	it.Before(func() {
		RegisterTestingT(t)
		f = test.NewBuildFactory(t)
	})

	when("Running with output to stdout/stderr", func() {
		it("should echo to stdout and stderr", func() {
			stdout := bytes.Buffer{}
			stderr := bytes.Buffer{}

			runner := ComposerRunner{
				Out: &stdout,
				Err: &stderr,
				Logger: f.Build.Logger,
			}

			err := runner.Run("echo", "", "Hello")

			Expect(err).ToNot(HaveOccurred())
			Expect(stdout.String()).To(Equal("Hello\n"))
			Expect(stderr.String()).To(BeEmpty())

			stdout.Reset()
			stderr.Reset()

			err = runner.Run("cat", "", "/does/not/exist.txt")

			Expect(err).To(HaveOccurred())
			Expect(stdout.String()).To(BeEmpty())
			Expect(stderr.String()).To(Equal("cat: /does/not/exist.txt: No such file or directory\n"))
		})
	})

	when("Running and returning output", func() {
		it("should return stdout", func() {
			stderr := bytes.Buffer{}

			runner := ComposerRunner{
				Err: &stderr,
				Logger: f.Build.Logger,
			}

			output, err := runner.RunWithOutput("echo", "", "Hello")

			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal("Hello\n"))
			Expect(stderr.String()).To(BeEmpty())

			stderr.Reset()

			output, err = runner.RunWithOutput("cat", "", "/does/not/exist.txt")

			Expect(err).To(HaveOccurred())
			Expect(output).To(BeEmpty())
			Expect(stderr.String()).To(Equal("cat: /does/not/exist.txt: No such file or directory\n"))
		})
	})
}