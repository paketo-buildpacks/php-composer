package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/php-dist-cnb/php"

	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/php-composer-cnb/composer"
)

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create default detect context: %s", err)
		os.Exit(100)
	}

	code, err := runDetect(context)
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runDetect(context detect.Detect) (int, error) {
	buildpackYAML, err := composer.LoadComposerBuildpackYAML(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	path, err := composer.FindComposer(context.Application.Root, buildpackYAML.Composer.JsonPath)
	if err != nil {
		return context.Fail(), err
	}

	phpVersion, phpVersionSrc, err := findPHPVersion(path, context.Logger)
	if err != nil {
		return context.Fail(), err
	}

	return context.Pass(buildplan.Plan{
		Requires: []buildplan.Required{
			{
				Name:    php.Dependency,
				Version: phpVersion,
				Metadata: buildplan.Metadata{
					"build":                     true,
					buildpackplan.VersionSource: phpVersionSrc,
				},
			},
			{
				Name:    composer.Dependency,
				Version: buildpackYAML.Composer.Version,
			},
		},
		Provides: []buildplan.Provided{{Name: composer.Dependency}},
	})
}

func findPHPVersion(path string, logger logger.Logger) (string, string, error) {
	composerLockPath := filepath.Join(filepath.Dir(path), composer.ComposerLock)

	composerLockExists, err := helper.FileExists(composerLockPath)
	if err != nil {
		return "", "", err
	}

	if composerLockExists {
		return parseComposerLock(composerLockPath)
	}

	logger.Info("WARNING: Include a 'composer.lock' file with your application! This will make sure the exact same version of dependencies are used when you deploy to CloudFoundry. It will also enable caching of your dependency layer.")

	return parseComposerJSON(path)
}

func parseComposerJSON(path string) (string, string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	type composerRequire struct {
		Php string `json:"php"`
	}

	composerJSON := struct {
		Require composerRequire `json:"require"`
	}{}

	if err := json.Unmarshal(buf, &composerJSON); err != nil {
		return "", "", err
	}

	return composerJSON.Require.Php, php.ComposerJSONSource, nil
}

func parseComposerLock(path string) (string, string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	// Composer.lock -> platform can be a dict or an array
	type composerLockPlatform struct {
		Php string `json:"php"`
	}

	composerLock := struct {
		Platform composerLockPlatform `json:"platform"`
	}{}

	if err := json.Unmarshal(buf, &composerLock); err != nil {
		// this happens when it's an array, which doesn't tell us the PHP version
		// return empty string to accept default PHP version & don't error
		if err.Error() == "json: cannot unmarshal array into Go struct field .platform of type main.composerLockPlatform" {
			return "", "", nil
		}
		return "", "", err
	}

	return composerLock.Platform.Php, php.ComposerLockSource, nil
}
