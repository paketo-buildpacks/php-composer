package composer

import (
	"bytes"
	"path/filepath"
	"testing"

	bp "github.com/buildpack/libbuildpack/logger"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/php-composer/runner"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
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
			comp = NewComposer(factory.Build.Application.Root, "/tmp", factory.Build.Logger)
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
			Expect(fakeRunner.Arguments).To(ConsistOf("php", expectedPharPath, "config", "-g", "github-oauth.github.com", `sec ret`))

			Expect(comp.Config("key", "val", false))
			Expect(fakeRunner.Arguments).To(ConsistOf("php", expectedPharPath, "config", "key", `val`))
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
		it("should find the composer.json file under webdir", func() {
			subDir := "subdir"
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), `{"php": {"webdirectory": "public"}, "composer": {"json_path": "subdir"}}`)
			compsoserPath := filepath.Join(factory.Build.Application.Root, "public", subDir, ComposerJSON)
			test.WriteFile(t, compsoserPath, "")
			path, err := FindComposer(factory.Build.Application.Root, subDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(compsoserPath))
		})

		it("should find the composer.json file under app_root", func() {
			subDir := "subdir"
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), `{"composer": {"json_path": "subdir"}}`)
			compsoserPath := filepath.Join(factory.Build.Application.Root, subDir, ComposerJSON)
			test.WriteFile(t, compsoserPath, "")
			path, err := FindComposer(factory.Build.Application.Root, subDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(compsoserPath))
		})
	})

	when("there is a buildpack.yml", func() {
		it("loads and parses with defaults", func() {
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), `{"composer": {"json_path": "subdir"}}`)

			bpYaml, err := LoadComposerBuildpackYAML(factory.Build.Application.Root)
			Expect(err).ToNot(HaveOccurred())
			Expect(bpYaml.Composer.JsonPath).To(Equal("subdir"))
			Expect(bpYaml.Composer.VendorDirectory).To(Equal("vendor"))
			Expect(bpYaml.Composer.InstallOptions).To(ConsistOf("--no-dev"))
		})

		it("loads and parses the file", func() {
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), `{"composer": {"json_path": "subdir", "vendor_directory": "somedir", "install_options": ["one", "two", "three"]}}`)

			bpYaml, err := LoadComposerBuildpackYAML(factory.Build.Application.Root)
			Expect(err).ToNot(HaveOccurred())
			Expect(bpYaml.Composer.JsonPath).To(Equal("subdir"))
			Expect(bpYaml.Composer.VendorDirectory).To(Equal("somedir"))
			Expect(bpYaml.Composer.InstallOptions).To(ConsistOf("one", "two", "three"))
		})

		it("loads and parses the file with install_global", func() {
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), `{"composer": {"install_global": ["one", "two", "three"]}}`)

			bpYaml, err := LoadComposerBuildpackYAML(factory.Build.Application.Root)
			Expect(err).ToNot(HaveOccurred())
			Expect(bpYaml.Composer.InstallGlobal).To(ConsistOf("one", "two", "three"))
		})

		when("WarnComposerBuildpackYAML", func() {
			when("there is a buildpack.yml", func() {
				it("warns about using buildpack.yml and shows equivalent env vars", func() {
					test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"),
						`{
"composer": {
	"version" : "some-version",
	"install_options" : ["--opt", "--other-opt"],
	"vendor_directory": "some-dir",
	"json_path": "subdir",
	"install_global" : ["--global", "--opt"]
	}
}`)

					buf := bytes.NewBuffer(nil)
					logger := logger.Logger{Logger: bp.NewLogger(buf, buf)}
					Expect(WarnComposerBuildpackYAML(logger, "1.2.3", factory.Build.Application.Root)).To(Succeed())
					Expect(buf.String()).To(ContainSubstring(`WARNING: Setting composer configurations through buildpack.yml will be deprecated soon in buildpack v2.0.0.`))
					Expect(buf.String()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
					Expect(buf.String()).To(ContainSubstring("composer.version -> BP_COMPOSER_VERSION"))
					Expect(buf.String()).To(ContainSubstring("composer.install_options -> BP_COMPOSER_INSTALL_OPTIONS"))
					Expect(buf.String()).To(ContainSubstring("composer.vendor_directory -> COMPOSER_VENDOR_DIR"))
					Expect(buf.String()).To(ContainSubstring("composer.json_path -> COMPOSER"))
					Expect(buf.String()).To(ContainSubstring("composer.install_global -> BP_COMPOSER_GLOBAL_INSTALL_OPTIONS"))
				})
			})
			when("there is no buildpack.yml", func() {
				it("logs no warning", func() {
					buf := bytes.NewBuffer(nil)
					logger := logger.Logger{Logger: bp.NewLogger(buf, buf)}
					Expect(WarnComposerBuildpackYAML(logger, "1.2.3", factory.Build.Application.Root)).To(Succeed())
					Expect(buf.String()).NotTo(ContainSubstring("WARNING: Setting composer configurations through buildpack.yml will be deprecated soon in buildpack v2.0.0."))
					Expect(buf.String()).NotTo(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
				})
			})
		})
	})

	when("there are PHP extensions listed in composer.json", func() {
		buf := bytes.NewBufferString(`ext-fileinfo  1.0.5                                      success   
			ext-gd        7.1.23                                     success   
			ext-kasjadf   n/a     __root__ requires ext-kasjadf (*)  missing   
			ext-mbstring  7.1.23                                     success   
			ext-mysqli    7.1.23                                     success   
			ext-zip       1.13.5                                     success   
			php           7.1.23                                     success   `)

		it("grabs a list of the extensions excluding php and already-installed extensions", func() {
			fakeRunner := &runner.FakeRunner{}
			comp := NewComposer(factory.Build.Application.Root, "/tmp", factory.Build.Logger)
			comp.Runner = fakeRunner
			fakeRunner.Out = buf

			extensions, err := comp.CheckPlatformReqs()
			Expect(err).ToNot(HaveOccurred())
			Expect(extensions).To(ConsistOf("kasjadf"))
		})

		it("grabs a list of the extensions excluding php even when extension name includes ext characters", func() {
			fakeRunner := &runner.FakeRunner{}
			comp := NewComposer(factory.Build.Application.Root, "/tmp", factory.Build.Logger)
			comp.Runner = fakeRunner
			fakeRunner.Out = bytes.NewBufferString(`ext-pdo         n/a     doctrine/orm requires ext-pdo (*)                 missing
ext-pdo_sqlite  n/a     symfony/symfony-demo requires ext-pdo_sqlite (*)  missing
php             7.3.11                                                    success
`)

			extensions, err := comp.CheckPlatformReqs()
			Expect(err).ToNot(HaveOccurred())
			Expect(extensions).To(ConsistOf("pdo", "pdo_sqlite"))
		})

	})
}
