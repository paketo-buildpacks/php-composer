package packages

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// NewContributor creates a new "packages" contributor for installing Composer packages
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
		composer:         composer.NewComposer(composerDir, composerPharPath, context.Logger),
	}

	if err := contributor.initializeEnv(buildpackYAML.Composer.VendorDirectory); err != nil {
		return Contributor{}, false, err
	}

	return contributor, true, nil
}

func (c Contributor) Contribute() error {
	if err := c.composerLayer.Contribute(nil, c.contributeComposer, layers.Build); err != nil {
		return err
	}

	// This layer doesn't need to do anything; Only holds downloaded PHP packages from composer
	return c.cacheLayer.Contribute(c.composerMetadata, func(layer layers.Layer) error {
		return nil
	}, layers.Cache)
}

func (c Contributor) configureGithubOauthToken() error {
	githubOauthToken := os.Getenv("COMPOSER_GITHUB_OAUTH_TOKEN")
	if githubOauthToken != "" {
		github, err := NewDefaultGithub(githubOauthToken)
		if err != nil {
			return err
		}

		if ok, err := github.validateToken(); err != nil {
			return err
		} else if ok {
			if err := c.composer.Config("github-oauth.github.com", githubOauthToken, true); err != nil {
				return err
			}
		}

		if ok, err := github.checkRateLimit(); err != nil {
			return err
		} else if !ok {
			c.composer.Logger.Warning("The GitHub api rate limit has been exceeded. " +
				"Composer will continue by downloading from source, which might result in slower downloads. " +
				"You can increase your rate limit with a GitHub OAuth token. " +
				"Please obtain a GitHub OAuth token by registering your application at " +
				"https://github.com/settings/applications/new. " +
				"Then set COMPOSER_GITHUB_OAUTH_TOKEN in your environment to the value of this token.")
		}
	}

	return nil
}

func (c Contributor) contributeComposer(layer layers.Layer) error {
	php_extensions, err := c.composer.CheckPlatformReqs()
	if err != nil {
		return err
	}

	if err := c.enablePHPExtensions(php_extensions); err != nil {
		return err
	}

	if err := c.configureGithubOauthToken(); err != nil {
		return err
	}

	// TODO:
	// Run `composer global require` for all packages set in buildpack.yml

	err = c.warnAboutPublicComposerFiles(layer)
	if err != nil {
		return err
	}

	return c.composer.Install(c.composerBuildpackYAML.Composer.InstallOptions...)
}

func (c Contributor) enablePHPExtensions(extensions []string) error {
	buf := bytes.Buffer{}

	for _, extension := range extensions {
		buf.WriteString(fmt.Sprintf("extension = %s.so\n", extension))
	}

	return helper.WriteFile(filepath.Join(c.app.Root, ".php.ini.d", "composer-extensions.ini"), 0655, buf.String())
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

func (c Contributor) initializeEnv(vendorDirectory string) error {
	// override anything possibly set by the user
	err := os.Setenv("COMPOSER_HOME", filepath.Join(c.composerLayer.Root, ".composer"))
	if err != nil {
		return err
	}

	err = os.Setenv("COMPOSER_CACHE_DIR", filepath.Join(c.cacheLayer.Root, "cache"))
	if err != nil {
		return err
	}

	// set `--no-interaction` flag to every command, since users cannot interact
	err = os.Setenv("COMPOSER_NO_INTERACTION", "1")
	if err != nil {
		return err
	}

	err = os.Setenv("COMPOSER_VENDOR_DIR", filepath.Join(c.app.Root, vendorDirectory))
	if err != nil {
		return err
	}

	err = os.Setenv("PHPRC", filepath.Join(c.composerLayer.Root, "composer-php.ini"))
	if err != nil {
		return err
	}

	err = os.Setenv("PHP_INI_SCAN_DIR", filepath.Join(c.app.Root, ".php.ini.d"))
	if err != nil {
		return err
	}

	return nil
}
