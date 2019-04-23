package composer

import (
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
)

type Contributor struct {
	ComposerLayer     layers.DependencyLayer
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

	contributor := Contributor{ComposerLayer: builder.Layers.DependencyLayer(dep)}

	if _, ok := plan.Metadata["build"]; ok {
		contributor.buildContribution = true
	}

	return contributor, true, nil
}

func (n Contributor) Contribute() error {
	return n.ComposerLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.SubsequentLine("Expanding to %s", layer.Root)
		return helper.ExtractTarGz(artifact, layer.Root, 1)
	}, n.flags()...)
}

func (n Contributor) flags() []layers.Flag {
	flags := []layers.Flag{}

	if n.buildContribution {
		flags = append(flags, layers.Build)
	}

	return flags
}
