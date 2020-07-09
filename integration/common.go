package integration

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/dagger"
	. "github.com/onsi/gomega"
)

var (
	composerOfflineURI string
	phpDistOfflineURI  string
	phpWebOfflineURI   string
	buildpackInfo      struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}
)

func PreparePhpOfflineBps() {
	bpRoot, err := dagger.FindBPRoot()
	Expect(err).NotTo(HaveOccurred())

	composerOfflineURI, _, err = dagger.PackageCachedBuildpack(bpRoot)
	Expect(err).ToNot(HaveOccurred())

	phpDistRepo, err := dagger.GetLatestUnpackagedCommunityBuildpack("paketo-buildpacks", "php-dist")
	Expect(err).NotTo(HaveOccurred())

	phpDistOfflineURI, _, err = dagger.PackageCachedBuildpack(phpDistRepo)
	Expect(err).ToNot(HaveOccurred())

	phpWebRepo, err := dagger.GetLatestUnpackagedCommunityBuildpack("paketo-buildpacks", "php-web")
	Expect(err).NotTo(HaveOccurred())

	phpWebOfflineURI, _, err = dagger.PackageCachedBuildpack(phpWebRepo)
	Expect(err).ToNot(HaveOccurred())
}

// PreparePhpBps builds the current buildpacks.
func PreparePhpBps() ([]string, error) {
	bpRoot, err := dagger.FindBPRoot()
	Expect(err).NotTo(HaveOccurred())

	composerBp, err := dagger.PackageBuildpack(bpRoot)
	Expect(err).NotTo(HaveOccurred())

	phpDistBp, err := dagger.GetLatestBuildpack("php-dist-cnb")
	Expect(err).NotTo(HaveOccurred())

	phpWebBp, err := dagger.GetLatestBuildpack("php-web-cnb")
	Expect(err).NotTo(HaveOccurred())

	return []string{phpDistBp, composerBp, phpWebBp}, nil
}

// MakeBuildEnv creates a build environment map
func MakeBuildEnv(debug bool) map[string]string {
	env := make(map[string]string)
	if debug {
		env["BP_DEBUG"] = "true"
	}

	githubToken := os.Getenv("GIT_TOKEN")
	if githubToken != "" {
		env["COMPOSER_GITHUB_OAUTH_TOKEN"] = githubToken
	}

	return env
}

// PreparePhpApp builds the given test app
func PreparePhpApp(appName string, buildpacks []string, debug bool) (*dagger.App, error) {

	app, err := dagger.PackBuildWithEnv(filepath.Join("testdata", appName), MakeBuildEnv(debug), buildpacks...)
	if err != nil {
		return &dagger.App{}, err
	}

	app.SetHealthCheck("", "3s", "1s")
	app.Env["PORT"] = "8080"

	return app, nil
}

func DecodeBPToml() {
	file, err := os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	_, err = toml.DecodeReader(file, &buildpackInfo)
	Expect(err).NotTo(HaveOccurred())
}
