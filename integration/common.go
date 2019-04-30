package integration

import (
	"github.com/cloudfoundry/dagger"
	"path/filepath"
)

func PreparePhpApp (appName string) (*dagger.App, error) {
	composerBp, err := dagger.PackageBuildpack()
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