package composer

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/layers"
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
	composer         Composer
}

func NewContributor(context build.Build, composer Composer) (Contributor, bool, error) {
	_, shouldUseComposer := context.BuildPlan[DEPENDENCY]
	if !shouldUseComposer {
		return Contributor{}, false, nil
	}

	buf, err := ioutil.ReadFile(filepath.Join(context.Application.Root, COMPOSER_LOCK))
	if err != nil {
		return Contributor{}, false, err
	}

	hash := sha256.Sum256(buf)

	contributor := Contributor{
		app:              context.Application,
		composerLayer:    context.Layers.Layer(DEPENDENCY),
		cacheLayer:       context.Layers.Layer(CACHE_DEPENDENCY),
		composerMetadata: Metadata{"PHP Composer", hex.EncodeToString(hash[:])},
		composer:         composer,
	}

	return contributor, true, nil
}

func (c Contributor) Contribute() error {
	if err := c.composerLayer.Contribute(c.composerMetadata, c.contributeComposer, layers.Build); err != nil {
		return err
	}
	return c.cacheLayer.Contribute(c.composerMetadata, c.contributeComposerCache, layers.Cache)
}

func (c Contributor) contributeComposer(layer layers.Layer) error {
	//nodeModules := filepath.Join(c.app.Root, ModulesDir)
	//
	//vendored, err := helper.FileExists(nodeModules)
	//if err != nil {
	//	return fmt.Errorf("unable to stat node_modules: %s", err.Error())
	//}
	//
	//if vendored {
	//	c.nodeModulesLayer.Logger.Info("Rebuilding node_modules")
	//	if err := c.pkgManager.Rebuild(c.app.Root); err != nil {
	//		return fmt.Errorf("unable to rebuild node_modules: %s", err.Error())
	//	}
	//} else {
	//	c.nodeModulesLayer.Logger.Info("Installing node_modules")
	//	if err := c.pkgManager.Install(layer.Root, c.npmCacheLayer.Root, c.app.Root); err != nil {
	//		return fmt.Errorf("unable to install node_modules: %s", err.Error())
	//	}
	//}
	//
	//if err := os.MkdirAll(layer.Root, 0777); err != nil {
	//	return fmt.Errorf("unable make node modules layer: %s", err.Error())
	//}
	//
	//nodeModulesExist, err := helper.FileExists(nodeModules)
	//if err != nil {
	//	return fmt.Errorf("unable to stat node_modules: %s", err.Error())
	//}
	//
	//if nodeModulesExist {
	//	if err := helper.CopyDirectory(nodeModules, filepath.Join(layer.Root, ModulesDir)); err != nil {
	//		return fmt.Errorf(`unable to copy "%s" to "%s": %s`, nodeModules, layer.Root, err.Error())
	//	}
	//
	//	if err := os.RemoveAll(nodeModules); err != nil {
	//		return fmt.Errorf("unable to remove node_modules from the app dir: %s", err.Error())
	//	}
	//}
	//
	//if err := layer.OverrideSharedEnv("NODE_PATH", filepath.Join(layer.Root, ModulesDir)); err != nil {
	//	return err
	//}
	return nil
}

func (c Contributor) contributeComposerCache(layer layers.Layer) error {
	return os.MkdirAll(layer.Root, 0777)
}
