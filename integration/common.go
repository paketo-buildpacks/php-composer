package integration

import (
	"os"
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

	phpBp, err := dagger.GetLatestBuildpack("php-cnb")
	if err != nil {
		return []string{}, err
	}

	phpWebBp, err := dagger.GetLatestBuildpack("php-web-cnb")
	if err != nil {
		return []string{}, err
	}

	return []string{phpBp, composerBp, phpWebBp}, nil
}

// MakeBuildEnv creates a build environment map
func MakeBuildEnv(debug bool) map[string]string {
	env := make(map[string]string)
	if debug {
		env["BP_DEBUG"] = "true"
	}

	githubToken := os.Getenv("GIT_TOKEN")
	if githubToken != "" {
		env["COMPOSER_GITHUB_OAUTH_TOKEN"] = githubToken
	}

	return env
}

// PreparePhpApp builds the given test app
func PreparePhpApp(appName string, buildpacks []string, debug bool) (*dagger.App, error) {

	app, err := dagger.PackBuildWithEnv(filepath.Join("testdata", appName), MakeBuildEnv(debug), buildpacks...)
	if err != nil {
		return &dagger.App{}, err
	}

	app.SetHealthCheck("", "3s", "1s")
	app.Env["PORT"] = "8080"

	return app, nil
}
