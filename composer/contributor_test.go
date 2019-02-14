package composer

import (
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"
)

type FakeRunner struct {
	arguments []string
	cwd string
	err       error
}

func (f *FakeRunner) Run(bin, dir string, args ...string) error {
	f.arguments = append([]string{ bin}, args...)
	f.cwd = dir
	return f.err
}

func TestUnitComposer(t *testing.T) {
	spec.Run(t, "Modules", testComposer, spec.Report(report.Terminal{}))
}

func testComposer(t *testing.T, when spec.G, it spec.S) {
	when("we are running composer", func() {
		var factory *test.BuildFactory
		var fakeRunner *FakeRunner
		var composer Composer

		it.Before(func() {
			RegisterTestingT(t)
			factory = test.NewBuildFactory(t)
			fakeRunner = &FakeRunner{}
			composer = NewComposer(factory.Build.Application.Root)
			composer.Runner = fakeRunner
		})

		it("should run composer -V", func(){
			Expect(composer.Version()).To(Succeed())
			Expect(fakeRunner.arguments).To(ConsistOf("php", COMPOSER_PHAR, "-V"))
		})

		it("should run composer install", func(){
			Expect(composer.Install("--foo", "--bar")).To(Succeed())
			Expect(fakeRunner.arguments).To(ConsistOf("php", COMPOSER_PHAR, "install", "--no-progress", "--foo", "--bar"))
		})

		it("should run composer global", func(){
			Expect(composer.Global("--foo", "--bar")).To(Succeed())
			Expect(fakeRunner.arguments).To(ConsistOf("php", COMPOSER_PHAR, "global", "require", "--no-progress", "--foo", "--bar"))
		})

		it("should run config", func(){
			Expect(composer.Config("sec ret")).To(Succeed())
			Expect(fakeRunner.arguments).To(ConsistOf("php", COMPOSER_PHAR, "config", "-g", "github-oauth.github.com", `"sec ret"`))
		})
	})
}
