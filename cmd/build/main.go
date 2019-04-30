package main

import (
	"fmt"
	"github.com/cloudfoundry/php-composer-cnb/composer"
	"github.com/cloudfoundry/php-composer-cnb/packages"
	"os"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/build"
)

func main() {
	context, err := build.DefaultBuild()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create default build context: %s", err)
		os.Exit(100)
	}

	code, err := runBuild(context)
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runBuild(context build.Build) (int, error) {
	context.Logger.FirstLine(context.Logger.PrettyIdentity(context.Buildpack))

	composerContributor, willContributeComposer, err := composer.NewContributor(context)
	if err != nil {
		return context.Failure(102), err
	}

	if willContributeComposer {
		err := composerContributor.Contribute()
		if err != nil {
			return context.Failure(103), err
		}

		packageContributor, willContributePackages, err := packages.NewContributor(context, composerContributor.ComposerLayer.Root)
		if err != nil {
			return context.Failure(104), err
		}

		if ! willContributePackages {
			// should always run if composer is being installed
			return context.Failure(105), err
		}

		err = packageContributor.Contribute()
		if err != nil {
			return context.Failure(106), err
		}
	}

	return context.Success(buildplan.BuildPlan{})
}
