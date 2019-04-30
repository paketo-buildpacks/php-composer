package composer

import (
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/php-composer-cnb/runner"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"path/filepath"
	"testing"
)

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
		var fakeRunner *runner.FakeRunner
		var comp Composer
		var expectedPharPath string

		it.Before(func() {
			fakeRunner = &runner.FakeRunner{}
			comp = NewComposer(factory.Build.Application.Root, "/tmp")
			comp.Runner = fakeRunner
			expectedPharPath = filepath.Join("/tmp", ComposerPHAR)
		})

		it("should run composer -V", func() {
			Expect(comp.Version()).To(Succeed())
			Expect(fakeRunner.Arguments).To(ConsistOf("php", expectedPharPath, "-V"))
		})

		it("should run composer install", func() {
			Expect(comp.Install("--foo", "--bar")).To(Succeed())
			Expect(fakeRunner.Arguments).To(ConsistOf("php", expectedPharPath, "install", "--no-progress", "--foo", "--bar"))
		})

		it("should run composer global", func() {
			Expect(comp.Global("--foo", "--bar")).To(Succeed())
			Expect(fakeRunner.Arguments).To(ConsistOf("php", expectedPharPath, "global", "require", "--no-progress", "--foo", "--bar"))
		})

		it("should run config", func() {
			Expect(comp.Config("github-oauth.github.com", "sec ret", true)).To(Succeed())
			Expect(fakeRunner.Arguments).To(ConsistOf("php", expectedPharPath, "config", "-g", "github-oauth.github.com", `"sec ret"`))

			Expect(comp.Config("key", "val", false))
			Expect(fakeRunner.Arguments).To(ConsistOf("php", expectedPharPath, "config", "key", `"val"`))
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

	when("there is a buildpack.yml", func() {
		it("loads and parses the file", func() {
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), `{"composer": {"json_path": "subdir", "github_oauth_token": "fake", "install_options": ["one", "two", "three"]}}`)

			bpYaml, err := LoadComposerBuildpackYAML(factory.Build.Application.Root)
			Expect(err).ToNot(HaveOccurred())
			Expect(bpYaml.Composer.JsonPath).To(Equal("subdir"))
			Expect(bpYaml.Composer.GitHubOAUTHToken).To(Equal("fake"))
			Expect(bpYaml.Composer.InstallOptions).To(ConsistOf("one", "two", "three"))
		})
	})
}
