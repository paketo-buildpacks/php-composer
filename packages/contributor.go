package packages

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/php-composer-cnb/composer"
	"github.com/cloudfoundry/php-web-cnb/phpweb"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
)


type Metadata struct {
	Name string
	Hash string
}

func (m Metadata) Identity() (name string, version string) {
	return m.Name, m.Hash
}

type Contributor struct {
	app                   application.Application
	composerLayer         layers.Layer
	cacheLayer            layers.Layer
	composerMetadata      Metadata
	composer              composer.Composer
	composerBuildpackYAML composer.BuildpackYAML
}

func NewContributor(context build.Build, composerPharPath string) (Contributor, bool, error) {
	buildpackYAML, err := composer.LoadComposerBuildpackYAML(context.Application.Root)
	if err != nil {
		return Contributor{}, false, err
	}

	path, err := composer.FindComposer(context.Application.Root, buildpackYAML.Composer.JsonPath)
	if err != nil {
		return Contributor{}, false, err
	}

	composerDir := filepath.Dir(path)
	lockPath := filepath.Join(composerDir, composer.ComposerLock)
	var hash [32]byte
	if exists, err := helper.FileExists(lockPath); err != nil {
		return Contributor{}, false, err
	} else if exists {
		buf, err := ioutil.ReadFile(lockPath)
		if err != nil {
			return Contributor{}, false, err
		}

		hash = sha256.Sum256(buf)
	} else {
		randBuf := make([]byte, 512)
		rand.Read(randBuf)
		hash = sha256.Sum256(randBuf)
	}


	contributor := Contributor{
		app:              context.Application,
		composerLayer:    context.Layers.Layer(composer.Dependency),
		cacheLayer:       context.Layers.Layer(composer.CacheDependency),
		composerMetadata: Metadata{"PHP Composer", hex.EncodeToString(hash[:])},
		composer:         composer.NewComposer(composerDir, composerPharPath),
	}

	if err := contributor.initializeEnv(); err != nil {
		return Contributor{}, false, err
	}

	return contributor, true, nil
}

func (c Contributor) Contribute() error {
	if err := c.composerLayer.Contribute(nil , c.contributeComposer, layers.Build); err != nil {
		return err
	}

	// This layer doesn't need to do anything; Only holds downloaded PHP packages from composer
	return c.cacheLayer.Contribute(c.composerMetadata, func(layer layers.Layer) error {
		return nil
	}, layers.Cache)
}

func (c Contributor) configureGithubOauthToken() error {
	// TODO: validate github oauth token in un-vendored case
	// TODO: check rate limiting
	// See below
	// TODO: https://github.com/cloudfoundry/php-buildpack/blob/master/extensions/composer/extension.py#L237-L298

	if c.composerBuildpackYAML.Composer.GitHubOAUTHToken != "" {
		return c.composer.Config(composer.GithubOAUTHKey, c.composerBuildpackYAML.Composer.GitHubOAUTHToken, true)
	}
	return nil
}

func (c Contributor) contributeComposer(layer layers.Layer) error {
	// TODO:
	// Run `composer global require` for all packages set in buildpack.yml
	// TODO: Need to cut a new release of php-web-cnb

	err := c.warnAboutPublicComposerFiles(layer)
	if err != nil {
		return err
	}

	return c.composer.Install(c.composerBuildpackYAML.Composer.InstallOptions...)
}

func (c Contributor) warnAboutPublicComposerFiles(layer layers.Layer) error {
	bpYAML, err := phpweb.LoadBuildpackYAML(c.app.Root)
	if err != nil {
		return err
	}
	composerLockPath := filepath.Join(c.app.Root, bpYAML.Config.WebDirectory, "composer.lock")
	composerJSONPath := filepath.Join(c.app.Root, bpYAML.Config.WebDirectory, "composer.json")

	lockExists, err := helper.FileExists(composerLockPath)
	if err != nil {
		return err
	}
	jsonExists, err := helper.FileExists(composerJSONPath)
	if err != nil {
		return err
	}

	if lockExists || jsonExists {
		layer.Logger.Info("WARNING: your composer.lock or composer.json files are located in the web directory which could publicly expose them. Please make sure this is really what you want")
	}

	return nil
}


func (c Contributor) initializeEnv() error {
	// override anything possibly set by the user
	err := os.Setenv("COMPOSER_HOME", filepath.Join(c.composerLayer.Root, ".composer"))
	if err != nil {
		return err
	}

	err = os.Setenv("COMPOSER_CACHE_DIR", filepath.Join(c.cacheLayer.Root, "cache"))
	if err != nil {
		return err
	}

	err = os.Setenv("COMPOSER_VENDOR_DIR", filepath.Join(c.app.Root, "vendor"))
	if err != nil {
		return err
	}

	err = os.Setenv("PHPRC", filepath.Join(c.composerLayer.Root, "composer-php.ini"))
	if err != nil {
		return err
	}

	return nil
}
