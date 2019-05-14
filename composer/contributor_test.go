package composer_test

import (
	"fmt"
	"github.com/cloudfoundry/php-cnb/php"
	"github.com/sclevine/spec/report"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/php-composer-cnb/composer"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func TestUnitComposer(t *testing.T) {
	spec.Run(t, "Composer", testContributor, spec.Report(report.Terminal{}))
}

func testContributor(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("NewContributor", func() {
		var stubComposerFixture = filepath.Join("testdata", "stub-composer.tar.gz")

		it("returns true if a build plan exists", func() {
			f := test.NewBuildFactory(t)
			f.AddBuildPlan(composer.Dependency, buildplan.Dependency{})
			f.AddDependency(composer.Dependency, stubComposerFixture)

			_, willContribute, err := composer.NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeTrue())
		})

		it("returns false if a build plan does not exist", func() {
			f := test.NewBuildFactory(t)

			_, willContribute, err := composer.NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeFalse())
		})

		it("contributes composer to the build layer when included in the build plan", func() {
			f := test.NewBuildFactory(t)
			f.AddBuildPlan(composer.Dependency, buildplan.Dependency{
				Metadata: buildplan.Metadata{"build": true},
			})
			f.AddDependency(composer.Dependency, stubComposerFixture)

			version := "12345"
			test.TouchFile(
				t,
				f.Build.Layers.Layer(php.Dependency).Root,
				"php/lib/php/extensions/no-debug-non-zts-"+version,
			)

			composerDep, _, err := composer.NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(composerDep.Contribute()).To(Succeed())

			layer := f.Build.Layers.Layer(composer.Dependency)
			Expect(layer).To(test.HaveLayerMetadata(true, false, false))
			Expect(filepath.Join(layer.Root, composer.ComposerPHAR)).To(BeARegularFile())
			Expect(filepath.Join(layer.Root, "composer-php.ini")).To(BeARegularFile())
			ini, err := ioutil.ReadFile(filepath.Join(layer.Root, "composer-php.ini"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(ini)).To(ContainSubstring(fmt.Sprintf("no-debug-non-zts-%s", version)))
			Expect(string(ini)).To(ContainSubstring("extension = openssl"))
			Expect(string(ini)).To(ContainSubstring("extension = zlib"))
		})
	})
}
