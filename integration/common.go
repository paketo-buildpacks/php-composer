package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/dagger"
	"github.com/cloudfoundry/packit/pexec"
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
	bpRoot, err := filepath.Abs("./..")
	if err != nil {
		return []string{}, err
	}

	composerBp, err := Package(bpRoot, "1.2.3", false)
	if err != nil {
		return []string{}, err
	}

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

func Package(root, version string, cached bool) (string, error) {
	var cmd *exec.Cmd

	bpPath := filepath.Join(root, "artifact")
	if cached {
		cmd = exec.Command(".bin/packager", "--archive", "--version", version, fmt.Sprintf("%s-cached", bpPath))
	} else {
		cmd = exec.Command(".bin/packager", "--archive", "--uncached", "--version", version, bpPath)
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("PACKAGE_DIR=%s", bpPath))
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if cached {
		return fmt.Sprintf("%s-cached.tgz", bpPath), err
	}

	return fmt.Sprintf("%s.tgz", bpPath), err
}

func GetGitVersion() (string, error) {
	gitExec := pexec.NewExecutable("git")
	revListOut := bytes.NewBuffer(nil)

	err := gitExec.Execute(pexec.Execution{
		Args:   []string{"rev-list", "--tags", "--max-count=1"},
		Stdout: revListOut,
	})
	if err != nil {
		return "", err
	}

	stdout := bytes.NewBuffer(nil)
	err = gitExec.Execute(pexec.Execution{
		Args:   []string{"describe", "--tags", strings.TrimSpace(revListOut.String())},
		Stdout: stdout,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.TrimPrefix(stdout.String(), "v")), nil
}
