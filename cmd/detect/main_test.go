package main

import (
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/php-cnb/php"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
	})

	when("there is a composer.json in the app root", func() {
		var compsoserPath string
		it.Before(func() {
			compsoserPath = filepath.Join(factory.Detect.Application.Root, COMPOSER_JSON)
			test.WriteFile(t, compsoserPath, "")
		})

		it("should find the composer.json file", func() {
			path, err := findComposer(factory.Detect)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(compsoserPath))
		})
	})

	when("there no composer.json file", func() {
		it("should return an error", func() {
			path, err := findComposer(factory.Detect)
			Expect(path).To(BeEmpty())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no \"" + COMPOSER_JSON + "\" found"))
		})
	})

	when("there is a composer.json in the web directory", func() {
		var compsoserPath string
		it.Before(func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), `{"php": {"webdirectory": "public"}}`)
			compsoserPath = filepath.Join(factory.Detect.Application.Root, "public", COMPOSER_JSON)
			test.WriteFile(t, compsoserPath, "")
		})

		it("should find the composer.json file", func() {
			path, err := findComposer(factory.Detect)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(compsoserPath))
		})
	})

	when("there is a composer.json location specified in COMPOSER_PATH", func() {
		var compsoserPath string
		var subDir string
		it.Before(func() {
			subDir = "subdir"
			os.Setenv(COMPOSER_PATH, subDir)
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), `{"php": {"webdirectory": "public"}}`)
			compsoserPath = filepath.Join(factory.Detect.Application.Root, "public", subDir, COMPOSER_JSON)
			test.WriteFile(t, compsoserPath, "")
		})

		it("should find the composer.json file", func() {
			path, err := findComposer(factory.Detect)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(compsoserPath))
		})
	})

	when("there is a composer.json with a php version", func() {
		var compsoserPath string
		var phpVersion string
		it.Before(func() {
			phpVersion = ">=5.6"
			composerJSONString := `{"require": {"php": "` + phpVersion + `"}}`
			compsoserPath = filepath.Join(factory.Detect.Application.Root, COMPOSER_JSON)
			test.WriteFile(t, compsoserPath, composerJSONString)
		})

		it("should parse the correct version", func() {
			version, err := findPHPVersion(compsoserPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(phpVersion))
		})
	})

	when("there is a composer.json and a composer.lock with a php version", func() {
		var (
			compsoserPath    string
			composerLockPath string
			phpVersion       string
			phpLockVersion   string
		)

		it.Before(func() {
			phpVersion = ">=5.6"
			composerJSONString := `{"require": {"php": "` + phpVersion + `"}}`
			compsoserPath = filepath.Join(factory.Detect.Application.Root, COMPOSER_JSON)
			test.WriteFile(t, compsoserPath, composerJSONString)

			phpLockVersion = ">=7.0"
			composerLockString := `{"platform": {"php": "` + phpLockVersion + `"}}`
			composerLockPath = filepath.Join(factory.Detect.Application.Root, COMPOSER_LOCK)
			test.WriteFile(t, composerLockPath, composerLockString)
		})

		it("should parse the version from composer.lock", func() {
			version, err := findPHPVersion(compsoserPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(phpLockVersion))
		})
	})

	when("there is a composer.json", func() {
		const VERSION string = "1.2.3"
		var compsoserPath string

		it.Before(func() {
			composerJSONString := `{"require": {"php": "` + VERSION + `"}}`

			compsoserPath = filepath.Join(factory.Detect.Application.Root, COMPOSER_JSON)
			test.WriteFile(t, compsoserPath, composerJSONString)
			factory.AddBuildPlan(DEPENDENCY, buildplan.Dependency{})
			fakeVersion := "php.default.VERSION"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}
		})

		when("there is no composer version specified in buildpack.yml", func() {
			it("should contribute to the build plan with the default composer version", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).NotTo(HaveOccurred())
				Expect(code).To(Equal(detect.PassStatusCode))

				Expect(factory.Output).To(Equal(buildplan.BuildPlan{
					DEPENDENCY: buildplan.Dependency{
						Metadata: buildplan.Metadata{"build": true},
					},
					php.Dependency: buildplan.Dependency{
						Version:  VERSION,
						Metadata: buildplan.Metadata{"build": true},
					},
				}))
			})
		})

		when("there is a buildpack.yml", func() {
			it("should contribute to the build plan with the specified composer version", func() {
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), `{"composer": {"version": "1.2.3"}}`)

				code, err := runDetect(factory.Detect)
				Expect(err).NotTo(HaveOccurred())
				Expect(code).To(Equal(detect.PassStatusCode))

				Expect(factory.Output).To(Equal(buildplan.BuildPlan{
					DEPENDENCY: buildplan.Dependency{
						Version: "1.2.3",
						Metadata: buildplan.Metadata{"build": true},
					},
					php.Dependency: buildplan.Dependency{
						Version:  VERSION,
						Metadata: buildplan.Metadata{"build": true},
					},
				}))
			})
		})
	})

	when("there is no composer.json", func() {
		it("should NOT contribute to the build plan", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).To(HaveOccurred())
			Expect(code).To(Equal(detect.FailStatusCode))
		})
	})
}
