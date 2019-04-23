package composer

import (
	"bytes"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/php-composer-cnb/runner"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"path/filepath"
	"testing"
)

type FakeRunner struct {
	arguments []string
	cwd       string
	err       error
}

func (f *FakeRunner) Run(bin, dir string, args ...string) error {
	f.arguments = append([]string{bin}, args...)
	f.cwd = dir
	return f.err
}

func TestUnitComposer(t *testing.T) {
	spec.Run(t, "ComposerRunner", testComposer, spec.Report(report.Terminal{}))
}

func testComposer(t *testing.T, when spec.G, it spec.S) {
	var factory *test.BuildFactory

	it.Before(func() {
		RegisterTestingT(t)

		factory = test.NewBuildFactory(t)
	})


	when("we are running composer", func() {
		var fakeRunner *FakeRunner
		var comp Composer

		it.Before(func() {
			fakeRunner = &FakeRunner{}
			comp = NewComposer(factory.Build.Application.Root, make(map[string]string))
			comp.Runner = fakeRunner
		})

		it("should run composer -V", func() {
			Expect(comp.Version()).To(Succeed())
			Expect(fakeRunner.arguments).To(ConsistOf("php", ComposerPHAR, "-V"))
		})

		it("should run composer install", func() {
			Expect(comp.Install("--foo", "--bar")).To(Succeed())
			Expect(fakeRunner.arguments).To(ConsistOf("php", ComposerPHAR, "install", "--no-progress", "--foo", "--bar"))
		})

		it("should run composer global", func() {
			Expect(comp.Global("--foo", "--bar")).To(Succeed())
			Expect(fakeRunner.arguments).To(ConsistOf("php", ComposerPHAR, "global", "require", "--no-progress", "--foo", "--bar"))
		})

		it("should run config", func() {
			Expect(comp.Config("sec ret")).To(Succeed())
			Expect(fakeRunner.arguments).To(ConsistOf("php", ComposerPHAR, "config", "-g", "github-oauth.github.com", `"sec ret"`))
		})
	})

	when("we are running a command with a custom environment", func() {
		it("should accept env variables", func() {
			env := make(map[string]string, 1)
			env["TEST_ENV_VAR"] = "1234"

			buf := bytes.Buffer{}

			comp := Composer {
				Runner: runner.ComposerRunner {
					Env: env,
					Out: &buf,
					Err: &buf,
				},
				appRoot: factory.Build.Application.Root,
			}

			err := comp.Runner.Run("env", "/tmp")
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(ContainSubstring("TEST_ENV_VAR=1234"))
		})
	})

	when("there is a composer.json in the app root", func() {
		var compsoserPath string
		it.Before(func() {
			compsoserPath = filepath.Join(factory.Build.Application.Root, ComposerJSON)
			test.WriteFile(t, compsoserPath, "")
		})

		it("should find the composer.json file", func() {
			path, err := FindComposer(factory.Build.Application.Root, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(compsoserPath))
		})
	})

	when("there no composer.json file", func() {
		it("should return an error", func() {
			path, err := FindComposer(factory.Build.Application.Root, "")
			Expect(path).To(BeEmpty())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no \"" + ComposerJSON + "\" found"))
		})
	})

	when("there is a composer.json in the web directory", func() {
		var compsoserPath string
		it.Before(func() {
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), `{"php": {"webdirectory": "public"}}`)
			compsoserPath = filepath.Join(factory.Build.Application.Root, "public", ComposerJSON)
			test.WriteFile(t, compsoserPath, "")
		})

		it("should find the composer.json file", func() {
			path, err := FindComposer(factory.Build.Application.Root, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(compsoserPath))
		})
	})

	when("there is a composer.json location specified in buildpack.yml", func() {
		var compsoserPath string
		var subDir string
		it.Before(func() {
			subDir = "subdir"
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), `{"php": {"webdirectory": "public"}, "composer": {"json_path": "subdir"}}`)
			compsoserPath = filepath.Join(factory.Build.Application.Root, "public", subDir, ComposerJSON)
			test.WriteFile(t, compsoserPath, "")
		})

		it("should find the composer.json file", func() {
			path, err := FindComposer(factory.Build.Application.Root, subDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(compsoserPath))
		})
	})
}
