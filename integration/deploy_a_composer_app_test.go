/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration

import (
	"fmt"
	"os"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var buildpacks []string

func TestIntegrationComposerApp(t *testing.T) {
	RegisterTestingT(t)

	var err error
	buildpacks, err = PreparePhpBps()
	Expect(err).ToNot(HaveOccurred())
	defer func() {
		for _, buildpack := range buildpacks {
			os.RemoveAll(buildpack)
		}
	}()

	spec.Run(t, "Deploy A Composer App", testIntegrationComposerApp, spec.Report(report.Terminal{}))
}

func testIntegrationComposerApp(t *testing.T, when spec.G, it spec.S) {
	var (
		app *dagger.App
		err error
	)

	when("deploying a basic Composer app", func() {
		it("it deploys using defaults and installs a package using Composer", func() {
			app, err = PreparePhpApp("composer_app", buildpacks, false)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.Info()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.Logs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			// ensure composer library is available & functions
			logs, err := app.Logs()
			Expect(err).ToNot(HaveOccurred())
			Expect(logs).To(ContainSubstring("SUCCESS"))

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("OK"))
		})

		it("deploys using custom composer setting and installs a package using Composer", func() {
			app, err = PreparePhpApp("composer_app_custom", buildpacks, false)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.Info()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.Logs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			// ensure composer library is available & functions
			logs, err := app.Logs()
			Expect(err).ToNot(HaveOccurred())
			Expect(logs).To(ContainSubstring("SUCCESS"))

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("OK"))
		})

		it("deploys an app that has PHP extensions specified in composer.json", func() {
			ExpectedExtensions := []string{
				"zip",
				"gd",
				"fileinfo",
				"mysqli",
				"mbstring",
			}

			app, err = PreparePhpApp("composer_app_extensions", buildpacks, true)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.Info()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.Logs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			// ensure composer library is available & functions
			logs, err := app.Logs()
			Expect(err).ToNot(HaveOccurred())
			Expect(logs).To(ContainSubstring("SUCCESS"))

			buildLogs := app.BuildLogs()
			Expect(buildLogs).To(ContainSubstring("Running `php /layers/org.cloudfoundry.php-composer/php-composer/composer.phar config -g github-oauth.github.com "))

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("OK"))

			// ensure C extensions are loaded at runtime & during post-install scripts
			Expect(logs).ToNot(ContainSubstring("Unable to load dynamic library"))

			body, _, err = app.HTTPGet("/extensions.php")
			Expect(err).ToNot(HaveOccurred())
			for _, extension := range ExpectedExtensions {
				Expect(body).To(ContainSubstring(extension))
				Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("PostInstall [%s]", extension)))
			}
		})

		it("deploys an app that installs global scripts using Composer and runs them as post scripts", func() {
			app, err = PreparePhpApp("composer_app_global", buildpacks, true)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.Info()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.Logs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			buildLogs := app.BuildLogs()
			Expect(buildLogs).To(ContainSubstring("Running `php /layers/org.cloudfoundry.php-composer/php-composer/composer.phar global require --no-progress friendsofphp/php-cs-fixer fxp/composer-asset-plugin:~1.3` from directory '/workspace'"))

			Expect(buildLogs).To(ContainSubstring("php-cs-fixer -h"))
			Expect(buildLogs).To(ContainSubstring("php /layers/org.cloudfoundry.php-composer/php-composer/vendor/bin/php-cs-fixer list"))

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("OK"))
		})
	})
}
