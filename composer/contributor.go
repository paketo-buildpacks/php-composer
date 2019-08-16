package composer

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/php-dist-cnb/php"
	"github.com/cloudfoundry/php-web-cnb/config"
)

type Contributor struct {
	ComposerLayer     layers.DependencyLayer
	PhpLayer          layers.Layer
	buildContribution bool
}

func NewContributor(builder build.Build) (Contributor, bool, error) {
	plan, wantDependency, err := builder.Plans.GetShallowMerged(Dependency)
	if err != nil || !wantDependency {
		return Contributor{}, false, err
	}

	deps, err := builder.Buildpack.Dependencies()
	if err != nil {
		return Contributor{}, false, err
	}

	dep, err := deps.Best(Dependency, plan.Version, builder.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	contributor := Contributor{
		ComposerLayer: builder.Layers.DependencyLayer(dep),
		PhpLayer:      builder.Layers.Layer(php.Dependency),
	}

	if _, ok := plan.Metadata["build"]; ok {
		contributor.buildContribution = true
	}

	return contributor, true, nil
}

func (n Contributor) Contribute() error {
	return n.ComposerLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.SubsequentLine("Expanding to %s", layer.Root)

		err := helper.CopyFile(artifact, filepath.Join(layer.Root, ComposerPHAR))
		if err != nil {
			return err
		}

		// generate temp php.ini for use by Composer during this buildpack
		return n.writePhpIni()
	}, n.flags()...)
}

func (n Contributor) flags() []layers.Flag {
	flags := []layers.Flag{}

	if n.buildContribution {
		flags = append(flags, layers.Build)
	}

	return flags
}

func (n Contributor) writePhpIni() error {
	phpIniCfg := config.PhpIniConfig{
		PhpHome: os.Getenv("PHP_HOME"),
		PhpAPI:  os.Getenv("PHP_API"),
		Extensions: []string{
			"openssl",
			"zlib",
		},
	}

	phpIniPath := filepath.Join(n.ComposerLayer.Root, "composer-php.ini")
	if err := config.ProcessTemplateToFile(config.PhpIniTemplate, phpIniPath, phpIniCfg); err != nil {
		return err
	}

	return nil
}
