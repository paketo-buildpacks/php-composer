package integration

import (
	"path/filepath"

	"github.com/cloudfoundry/dagger"
)

// PreparePhpBps builds the current buildpacks
func PreparePhpBps() ([]string, error) {
	bpRoot, err := dagger.FindBPRoot()
	if err != nil {
		return []string{}, err
	}

	composerBp, err := dagger.PackageBuildpack(bpRoot)
	if err != nil {
		return []string{}, err
	}

	// TODO: Need to cut a new release of php-web-cnb
	phpBp, err := dagger.PackageBuildpack("/Users/dmikusa/Downloads/buildpacks/php-cnb")
	// phpBp, err := dagger.GetLatestBuildpack("php-cnb")
	if err != nil {
		return []string{}, err
	}

	phpWebBp, err := dagger.PackageBuildpack("/Users/dmikusa/Downloads/buildpacks/php-web-cnb")
	// phpWebBp, err := dagger.GetLatestBuildpack("php-web-cnb")
	if err != nil {
		return []string{}, err
	}

	return []string{phpBp, composerBp, phpWebBp}, nil
}

// PreparePhpApp builds the given test app
func PreparePhpApp(appName string, buildpacks []string) (*dagger.App, error) {
	app, err := dagger.PackBuild(filepath.Join("testdata", appName), buildpacks...)
	if err != nil {
		return &dagger.App{}, err
	}

	app.SetHealthCheck("", "3s", "1s")
	app.Env["PORT"] = "8080"

	return app, nil
}
