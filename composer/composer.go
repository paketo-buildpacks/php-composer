package composer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/paketo-buildpacks/php-composer/runner"
	"github.com/paketo-buildpacks/php-web/config"
	"gopkg.in/yaml.v2"
)

const (
	Dependency         = "composer"
	PackagesDependency = "php-composer-packages"
	CacheDependency    = "php-composer-cache"
	ComposerLock       = "composer.lock"
	ComposerJSON       = "composer.json"
	ComposerPHAR       = "composer.phar"
	GithubOAUTHKey     = "github-oauth.github.com"
)

// Composer runner
type Composer struct {
	Logger     logger.Logger
	Runner     runner.Runner
	workingDir string
	pharPath   string
}

// NewComposer creates a new Composer runner
func NewComposer(composerJsonPath, composerPharPath string, logger logger.Logger) Composer {
	return Composer{
		Logger: logger,
		Runner: runner.ComposerRunner{
			Logger: logger,
		},
		workingDir: composerJsonPath,
		pharPath:   filepath.Join(composerPharPath, ComposerPHAR),
	}
}

// Install runs `composer install`
func (c Composer) Install(args ...string) error {
	args = append([]string{c.pharPath, "install", "--no-progress"}, args...)
	return c.Runner.Run("php", c.workingDir, args...)
}

// Version runs `composer version`
func (c Composer) Version() error {
	return c.Runner.Run("php", c.workingDir, c.pharPath, "-V")
}

// Global runs `composer global`
func (c Composer) Global(args ...string) error {
	args = append([]string{c.pharPath, "global", "require", "--no-progress"}, args...)
	return c.Runner.Run("php", c.workingDir, args...)
}

// Config runs `composer config`
func (c Composer) Config(key, value string, global bool) error {
	args := []string{c.pharPath, "config"}
	if global {
		args = append(args, "-g")
	}
	args = append(args, key, value)
	return c.Runner.Run("php", c.workingDir, args...)
}

// CheckPlatformReqs looks for required extension
func (c Composer) CheckPlatformReqs() ([]string, error) {

	// let Composer tell us what extensions are required
	output, err := c.Runner.RunWithOutput("php", c.workingDir, c.pharPath, "check-platform-reqs")
	if err != nil {
		exitError, ok := err.(*exec.ExitError)

		if !ok || exitError.ExitCode() != 2 {
			return []string{}, err
		}
	}

	extensions := []string{}
	for _, line := range strings.Split(output, "\n") {
		chunks := strings.Split(strings.TrimSpace(line), " ")
		extensionName := strings.TrimPrefix(strings.TrimSpace(chunks[0]), "ext-")
		extensionStatus := strings.TrimSpace(chunks[len(chunks)-1])
		if extensionName != "php" && extensionStatus == "missing" {
			extensions = append(extensions, extensionName)
		}
	}

	return extensions, nil
}

// FindComposer locates the composer JSON and composer lock files
func FindComposer(appRoot string, composerJSONPath string) (string, error) {
	phpBuildpackYAML, err := config.LoadBuildpackYAML(appRoot)
	if err != nil {
		return "", err
	}

	paths := []string{
		filepath.Join(appRoot, ComposerJSON),
		filepath.Join(appRoot, phpBuildpackYAML.Config.WebDirectory, ComposerJSON),
	}

	if composerJSONPath != "" {
		paths = append(
			paths,
			filepath.Join(appRoot, composerJSONPath, ComposerJSON),
			filepath.Join(appRoot, phpBuildpackYAML.Config.WebDirectory, composerJSONPath, ComposerJSON),
		)
	}

	for _, path := range paths {
		if exists, err := helper.FileExists(path); err != nil {
			return "", fmt.Errorf("error checking filepath: %s", path)
		} else if exists {
			return path, nil
		}
	}

	return "", fmt.Errorf(`no "%s" found in the following locations: %v`, ComposerJSON, paths)
}

type ComposerConfig struct {
	Version         string   `yaml:"version"`
	InstallOptions  []string `yaml:"install_options"`
	VendorDirectory string   `yaml:"vendor_directory"`
	JsonPath        string   `yaml:"json_path"`
	InstallGlobal   []string `yaml:"install_global"`
}

type BuildpackYAML struct {
	Composer ComposerConfig `yaml:"composer"`
}

// LoadComposerBuildpackYAML loads the buildpack YAML from disk
func LoadComposerBuildpackYAML(appRoot string) (BuildpackYAML, error) {
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(appRoot, "buildpack.yml")

	buildpackYAML.Composer.InstallOptions = []string{"--no-dev"}
	buildpackYAML.Composer.VendorDirectory = "vendor"

	if exists, err := helper.FileExists(configFile); err != nil {
		return BuildpackYAML{}, err
	} else if exists {
		file, err := os.Open(configFile)
		if err != nil {
			return BuildpackYAML{}, err
		}
		defer file.Close()

		contents, err := io.ReadAll(file)
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

func WarnComposerBuildpackYAML(logger logger.Logger, version, appRoot string) error {
	var (
		exists bool
		err    error
	)
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(appRoot, "buildpack.yml")
	if exists, err = helper.FileExists(configFile); err != nil {
		return err
	}
	if !exists {
		return nil
	}
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(contents, &buildpackYAML)
	if err != nil {
		return err
	}
	fieldMapping := map[string]string{}
	if buildpackYAML.Composer.Version != "" {
		fieldMapping["composer.version"] = "BP_COMPOSER_VERSION"
	}
	if len(buildpackYAML.Composer.InstallOptions) > 0 {
		fieldMapping["composer.install_options"] = "BP_COMPOSER_INSTALL_OPTIONS"
	}
	if buildpackYAML.Composer.VendorDirectory != "" {
		fieldMapping["composer.vendor_directory"] = "COMPOSER_VENDOR_DIR"
	}
	if buildpackYAML.Composer.JsonPath != "" {
		fieldMapping["composer.json_path"] = "COMPOSER"
	}
	if len(buildpackYAML.Composer.InstallGlobal) > 0 {
		fieldMapping["composer.install_global"] = "BP_COMPOSER_GLOBAL_INSTALL_OPTIONS"
	}

	nextMajorVersion := semver.MustParse(version).IncMajor()
	logger.BodyWarning("WARNING: Setting composer configurations through buildpack.yml will be deprecated soon in buildpack v%s.", nextMajorVersion.String())
	logger.BodyWarning("Buildpack.yml values will be replaced by environment variables in the next major version:")

	for field, envVar := range fieldMapping {
		logger.BodyWarning("  %s -> %s", field, envVar)
	}
	return nil
}
