package packages

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/php-composer-cnb/composer"
	"github.com/cloudfoundry/php-web-cnb/config"
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
	composerPackagesLayer layers.Layer
	cacheLayer            layers.Layer
	composerMetadata      Metadata
	composer              composer.Composer
	composerBuildpackYAML composer.BuildpackYAML
}

func generateRandomHash() [32]byte {
	randBuf := make([]byte, 512)
	rand.Read(randBuf)
	return sha256.Sum256(randBuf)
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
		hash = generateRandomHash()
	}

	contributor := Contributor{
		app:                   context.Application,
		composerLayer:         context.Layers.Layer(composer.Dependency),
		composerPackagesLayer: context.Layers.Layer(composer.PackagesDependency),
		cacheLayer:            context.Layers.Layer(composer.CacheDependency),
		composerMetadata:      Metadata{"PHP Composer", hex.EncodeToString(hash[:])},
		composer:              composer.NewComposer(composerDir, composerPharPath, context.Logger),
		composerBuildpackYAML: buildpackYAML,
	}

	if err := contributor.initializeEnv(buildpackYAML.Composer.VendorDirectory); err != nil {
		return Contributor{}, false, err
	}

	return contributor, true, nil
}

func (c Contributor) SetupVendorDir() error {
	composerLayerVendorDir := filepath.Join(c.composerPackagesLayer.Root, c.composerBuildpackYAML.Composer.VendorDirectory)
	composerAppVendorDir := filepath.Join(c.app.Root, c.composerBuildpackYAML.Composer.VendorDirectory)

	exists, err := helper.FileExists(composerAppVendorDir)
	if err != nil {
		return err
	} else if exists {
		if err := helper.CopyDirectory(composerAppVendorDir, composerLayerVendorDir); err != nil {
			return err
		}
		if err := os.RemoveAll(composerAppVendorDir); err != nil {
			return err
		}
	}

	// symlink vendor_home to "vendor" under the app root so PHP apps can find Composer dependencies
	return helper.WriteSymlink(composerLayerVendorDir, composerAppVendorDir)
}

func (c Contributor) Contribute() error {
	randomHash := generateRandomHash()
	if err := c.cacheLayer.Contribute(Metadata{"PHP Composer Cache", hex.EncodeToString(randomHash[:])}, func(layer layers.Layer) error { return nil }, layers.Cache); err != nil {
		return err
	}

	if err := c.alwaysRunComposerInit(c.composerPackagesLayer); err != nil {
		return err
	}

	if err := c.SetupVendorDir(); err != nil {
		return err
	}

	return c.composerPackagesLayer.Contribute(c.composerMetadata, c.contributeComposerPackages, layers.Launch)
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

func (c Contributor) installGlobalPackages() error {
	if len(c.composerBuildpackYAML.Composer.InstallGlobal) > 0 {
		binPath := strings.Join([]string{os.Getenv("PATH"), filepath.Join(c.composerPackagesLayer.Root, "global/vendor/bin")}, string(os.PathListSeparator))
		err := os.Setenv("PATH", binPath)
		if err != nil {
			return err
		}

		if err := c.setGlobalVendorDir(); err != nil {
			return err
		}

		if err := c.composer.Global(c.composerBuildpackYAML.Composer.InstallGlobal...); err != nil {
			return err
		}
	}
	return nil
}

func (c Contributor) alwaysRunComposerInit(layer layers.Layer) error {
	phpExtensions, err := c.composer.CheckPlatformReqs()
	if err != nil {
		return err
	}

	if err := c.enablePHPExtensions(phpExtensions); err != nil {
		return err
	}

	if err := c.configureGithubOauthToken(); err != nil {
		return err
	}

	if err := c.installGlobalPackages(); err != nil {
		return err
	}

	err = c.warnAboutPublicComposerFiles(layer)
	if err != nil {
		return err
	}

	return c.setAppVendorDir()
}

func (c Contributor) contributeComposerPackages(layer layers.Layer) error {
	if err := os.MkdirAll(layer.Root, os.ModePerm); err != nil {
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
	bpYAML, err := config.LoadBuildpackYAML(c.app.Root)
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

	err = os.Setenv("PHPRC", filepath.Join(c.composerLayer.Root, "composer-php.ini"))
	if err != nil {
		return err
	}

	err = os.Setenv("PHP_INI_SCAN_DIR", filepath.Join(c.app.Root, ".php.ini.d"))
	if err != nil {
		return err
	}

	binPath := strings.Join([]string{os.Getenv("PATH"), filepath.Join(c.app.Root, c.composerBuildpackYAML.Composer.VendorDirectory, "bin")}, string(os.PathListSeparator))
	err = os.Setenv("PATH", binPath)
	if err != nil {
		return err
	}

	return nil
}

func (c Contributor) setGlobalVendorDir() error {
	err := os.Setenv("COMPOSER_VENDOR_DIR", filepath.Join(c.composerPackagesLayer.Root, "global", "vendor"))
	if err != nil {
		return err
	}
	return nil
}

func (c Contributor) setAppVendorDir() error {
	err := os.Setenv("COMPOSER_VENDOR_DIR", filepath.Join(c.composerPackagesLayer.Root, c.composerBuildpackYAML.Composer.VendorDirectory))
	if err != nil {
		return err
	}
	return nil
}
