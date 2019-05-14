package composer

import (
	"errors"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/php-cnb/php"
	"github.com/cloudfoundry/php-web-cnb/config"
	"os"
	"path/filepath"
	"strings"
)

type Contributor struct {
	ComposerLayer     layers.DependencyLayer
	PhpLayer					layers.Layer
	buildContribution bool
}

func NewContributor(builder build.Build) (Contributor, bool, error) {
	plan, wantDependency := builder.BuildPlan[Dependency]
	if !wantDependency {
		return Contributor{}, false, nil
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
		PhpLayer: builder.Layers.Layer(php.Dependency),
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

		// add to current path so it's accessible by the rest of this buildpack
		newPath := strings.Join([]string{ os.Getenv("PATH"), filepath.Join(layer.Root, "bin")}, string(os.PathListSeparator))
		err = os.Setenv("PATH", newPath)
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
	folders, err := filepath.Glob(filepath.Join(n.PhpLayer.Root, "php/lib/php/extensions/no-debug-non-zts*"))

	if err != nil {
		return err
	}

	if len(folders) == 0 {
		return errors.New("php extensions folder not found")
	}

	extensionFolder := strings.Split(folders[0], "-")
	apiVersion := extensionFolder[len(extensionFolder)-1]

	phpIniCfg := config.PhpIniConfig{
		PhpHome:      n.ComposerLayer.Root,
		PhpAPI:       apiVersion,
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
