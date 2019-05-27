package packages

import (
	"bytes"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/php-composer-cnb/composer"
	"github.com/cloudfoundry/php-composer-cnb/runner"

	bplogger "github.com/buildpack/libbuildpack/logger"
	"github.com/cloudfoundry/libcfbuildpack/test"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitComposerPackage(t *testing.T) {
	spec.Run(t, "ComposerPackage", testComposerPackage, spec.Report(report.Terminal{}))
}

func testComposerPackage(t *testing.T, when spec.G, it spec.S) {
	var factory *test.BuildFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewBuildFactory(t)
	})

	when("NewContributor", func() {
		it.Before(func() {
			composerJSONString := `{something:this is a json file}`
			composerJSONPath := filepath.Join(factory.Build.Application.Root, composer.ComposerJSON)
			test.WriteFile(t, composerJSONPath, composerJSONString)
		})

		when("there is a lock file", func() {
			it("includes a hash of the lock file in the composer metadata", func() {
				composerLockString := `this is a lock file`
				composerLockPath := filepath.Join(factory.Build.Application.Root, composer.ComposerLock)
				test.WriteFile(t, composerLockPath, composerLockString)

				contributor, willContribute, err := NewContributor(factory.Build, "/tmp")
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.composerMetadata.Name).To(Equal("PHP Composer"))
				Expect(contributor.composerMetadata.Hash).To(Equal("fe2ebd62604e50ad1682fb67979fd368375c2347973c47af8b0394a5359e3e08"))
			})
		})

		when("there isn't a lock file", func() {
			it("randomly generates a hash for composer metadata", func() {
				// Caution: Not thread-safe; may cause test pollution
				rand.Seed(1)

				contributor, willContribute, err := NewContributor(factory.Build, "/tmp")
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.composerMetadata.Name).To(Equal("PHP Composer"))
				// TODO confirm the following doesn't change across different computers
				Expect(contributor.composerMetadata.Hash).To(Equal("96ebbb5c8694dd2c33b07ca6d40c10b0b670bc10176d2507d8b3b4a739d46f01"))
			})
		})
	})

	when("there is a lock file in WEBDIR", func() {
		it("should warn about the file being publicly accessible", func() {
			webdir := "htdocs"

			// write out composer.json & composer.lock
			composerJSONPath := filepath.Join(factory.Build.Application.Root, webdir, composer.ComposerJSON)
			test.WriteFile(t, composerJSONPath, "does not matter")
			composerLockPath := filepath.Join(factory.Build.Application.Root, webdir, composer.ComposerLock)
			test.WriteFile(t, composerLockPath, "does not matter")

			// write out buildpack.yml
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), `{"php": {"webdirectory": "htdocs"}}`)

			// run the contributor
			contributor, willContribute, err := NewContributor(factory.Build, "/tmp")

			Expect(err).ToNot(HaveOccurred())
			Expect(willContribute).To(BeTrue())

			debug := &bytes.Buffer{}
			info := &bytes.Buffer{}

			log := logger.Logger{Logger: bplogger.NewLogger(debug, info)}

			testLayer := factory.Build.Layers.Layer("test")
			testLayer.Logger = log

			err = contributor.warnAboutPublicComposerFiles(testLayer)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.String()).To(Equal("WARNING: your composer.lock or composer.json files are located in the web directory which could publicly expose them. Please make sure this is really what you want\n"))
		})
	})

	when("a github oauth token is supplied in buildpack.yml", func() {
		it("runs composer config to make that available to composer", func() {

			fakeRunner := &runner.FakeRunner{}
			comp := composer.NewComposer(factory.Build.Application.Root, "/tmp", factory.Build.Logger)
			comp.Runner = fakeRunner

			contributor := Contributor{
				app:              factory.Build.Application,
				composerLayer:    factory.Build.Layers.Layer("composer"),
				cacheLayer:       factory.Build.Layers.Layer("cache"),
				composerMetadata: Metadata{},
				composer:         comp,
				composerBuildpackYAML: composer.BuildpackYAML{
					Composer: composer.ComposerConfig{
						GitHubOAUTHToken: "qwerty",
					},
				},
			}

			Expect(contributor.configureGithubOauthToken()).ToNot(HaveOccurred())
			Expect(fakeRunner.Arguments).To(ConsistOf("php", filepath.Join("/tmp", composer.ComposerPHAR), "config", "-g", "github-oauth.github.com", "\"qwerty\""))
		})
	})
}
