package packages

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"io/ioutil"
	"math/rand"
	"path/filepath"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/php-composer-cnb/composer"
)

type Metadata struct {
	Name string
	Hash string
}

func (m Metadata) Identity() (name string, version string) {
	return m.Name, m.Hash
}

type Contributor struct {
	app              application.Application
	composerLayer    layers.Layer
	cacheLayer       layers.Layer
	composerMetadata Metadata
	composer         composer.Composer
}

func NewContributor(context build.Build, comp composer.Composer) (Contributor, bool, error) {
	buildpackYAML, err := composer.LoadComposerBuildpackYAML(context.Application.Root)
	if err != nil {
		return Contributor{}, false, err
	}

	path, err := composer.FindComposer(context.Application.Root, buildpackYAML.Composer.JsonPath)
	if err != nil {
		return Contributor{}, false, err
	}

	lockPath := filepath.Join(filepath.Dir(path), composer.ComposerLock)
	var hash [32]byte
	if exists, err := helper.FileExists(lockPath); err != nil  {
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
		composer:         comp,
	}

	return contributor, true, nil
}

func (c Contributor) Contribute() error {
	if err := c.composerLayer.Contribute(c.composerMetadata, c.contributeComposer, layers.Build); err != nil {
		return err
	}

	// This layer doesn't need to do anything; Only holds downloaded PHP packages from composer
	return c.cacheLayer.Contribute(c.composerMetadata, func(layer layers.Layer) error {
		return nil
	}, layers.Cache)
}

func (c Contributor) contributeComposer(layer layers.Layer) error {
	// TODO:
	 // if json and lock files are in web dir, warn that they may be publicly accessible
	 // debug? nah
	 // configure github oauth token if set in buildpack.yml
	 // Run `composer global require` for all packages set in buildpack.yml
	 // Run `composer install` with options as set in buildpack.yml
	 // Create NewComposer & feed in env variables
	//[]string {
	//	"COMPOSER_HOME=",
	//	"COMPOSER_CACHE_DIR=",
	//	"COMPOSER_BIN_DIR=",
	//	"COMPOSER_VENDOR_DIR=",
	//	"HTTP_PROXY=",
	//	"HTTPS_PROXY=",
	//	"NO_PROXY=",
	//	"LD_LIBRARY_PATH=",
	//	"PHPRC=",
	//	"PATH=",
	//}

	return nil
}
