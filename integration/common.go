package integration

import (
	"path/filepath"

	"github.com/cloudfoundry/dagger"
)

func PreparePhpApp(appName string) (*dagger.App, error) {
	bpRoot, err := dagger.FindBPRoot()
	if err != nil {
		return &dagger.App{}, err
	}

	composerBp, err := dagger.PackageBuildpack(bpRoot)
	if err != nil {
		return &dagger.App{}, err
	}

	phpBp, err := dagger.GetLatestBuildpack("php-cnb")
	if err != nil {
		return &dagger.App{}, err
	}

	phpWebBp, err := dagger.GetLatestBuildpack("php-web-cnb")
	if err != nil {
		return &dagger.App{}, err
	}

	app, err := dagger.PackBuild(filepath.Join("testdata", appName), phpBp, composerBp, phpWebBp)
	if err != nil {
		return &dagger.App{}, err
	}

	app.SetHealthCheck("", "3s", "1s")
	app.Env["PORT"] = "8080"

	return app, nil
}
