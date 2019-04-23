package packages

import (
	"github.com/cloudfoundry/php-composer-cnb/composer"
	"math/rand"
	"path/filepath"
	"testing"

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

				contributor, willContribute, err := NewContributor(factory.Build, composer.Composer{})
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

				contributor, willContribute, err := NewContributor(factory.Build, composer.Composer{})
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.composerMetadata.Name).To(Equal("PHP Composer"))
				// TODO confirm the following doesn't change across different computers
				Expect(contributor.composerMetadata.Hash).To(Equal("96ebbb5c8694dd2c33b07ca6d40c10b0b670bc10176d2507d8b3b4a739d46f01"))
			})
		})
	})
}
