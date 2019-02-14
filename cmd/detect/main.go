package main

import (
	"encoding/json"
	"fmt"
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/php-cnb/php"
	"github.com/cloudfoundry/php-composer-cnb/composer"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/php-web-cnb/phpweb"
)

const (
	COMPOSER_PATH = "COMPOSER_PATH"
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
	path, err := findComposer(context)
	if err != nil {
		return context.Fail(), err
	}

	phpVersion, err := findPHPVersion(path)
	if err != nil {
		return context.Fail(), err
	}

	buildpackYAML, err := loadComposerBuildpackYAML(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	return context.Pass(buildplan.BuildPlan{
		composer.DEPENDENCY: buildplan.Dependency{
			Version:  buildpackYAML.Composer.Version,
			Metadata: buildplan.Metadata{"build": true},
		},
		php.Dependency: buildplan.Dependency{
			Version:  phpVersion,
			Metadata: buildplan.Metadata{"build": true},
		},
	})
}

func findComposer(context detect.Detect) (string, error) {
	composerJSON := filepath.Join(context.Application.Root, composer.COMPOSER_JSON)

	if exists, err := helper.FileExists(composerJSON); err != nil {
		return "", fmt.Errorf("error checking filepath: %s", composerJSON)
	} else if exists {
		return composerJSON, nil
	}

	phpBuildpackYAML, err := phpweb.LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return "", err
	}

	composerBuildpackYAML, err := loadComposerBuildpackYAML(context.Application.Root)
	if err != nil {
		return "", err
	}

	composerJSON = filepath.Join(context.Application.Root, phpBuildpackYAML.Config.WebDirectory, composerBuildpackYAML.Composer.JsonPath, composer.COMPOSER_JSON)
	if exists, err := helper.FileExists(composerJSON); err != nil {
		return "", fmt.Errorf("error checking filepath: %s", composerJSON)
	} else if exists {
		return composerJSON, nil
	}

	return "", fmt.Errorf(`no "%s" found at: %s`, composer.COMPOSER_JSON, composerJSON)
}

type ComposerConfig struct {
	Version         string `yaml:"version"`
	InstallOptions  string `yaml:"install_options"`
	VendorDirectory string `yaml:"vendor_directory"`
	JsonPath        string `yaml:"json_path"`
}

type BuildpackYAML struct {
	Composer ComposerConfig `yaml:"composer"`
}

func loadComposerBuildpackYAML(appRoot string) (BuildpackYAML, error) {
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(appRoot, "buildpack.yml")
	if exists, err := helper.FileExists(configFile); err != nil {
		return BuildpackYAML{}, err
	} else if exists {
		file, err := os.Open(configFile)
		if err != nil {
			return BuildpackYAML{}, err
		}
		defer file.Close()

		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return BuildpackYAML{}, err
		}

		err = yaml.Unmarshal(contents, &buildpackYAML)
		if err != nil {
			return BuildpackYAML{}, err
		}
	}
	return buildpackYAML, nil
}

func findPHPVersion(path string) (string, error) {
	composerLockPath := filepath.Join(filepath.Dir(path), composer.COMPOSER_LOCK)

	if version, err := parseComposerLock(composerLockPath); err == nil && version != "" {
		return version, nil
	}

	return parseComposerJSON(path)
}

func parseComposerJSON(path string) (string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	type composerRequire struct {
		Php string `json:"php"`
	}

	composerJSON := struct {
		Require composerRequire `json:"require"`
	}{}

	if err := json.Unmarshal(buf, &composerJSON); err != nil {
		return "", err
	}

	return composerJSON.Require.Php, nil
}

func parseComposerLock(path string) (string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	type composerLockPlatform struct {
		Php string `json:"php"`
	}

	composerLock := struct {
		Platform composerLockPlatform `json:"platform"`
	}{}

	if err := json.Unmarshal(buf, &composerLock); err != nil {
		return "", err
	}

	return composerLock.Platform.Php, nil
}
