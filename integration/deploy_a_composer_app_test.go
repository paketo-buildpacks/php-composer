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

func TestIntegrationComposerApp(t *testing.T) {
	spec.Run(t, "Deploy A Composer App", testIntegrationComposerApp, spec.Report(report.Terminal{}))
}

func testIntegrationComposerApp(t *testing.T, when spec.G, it spec.S) {
	var (
		app *dagger.App
		Expect func(interface{}, ...interface{}) Assertion
	)

	it.Before(func() {
		var err error
		Expect = NewWithT(t).Expect

		app, err = PreparePhpApp("composer_app")
		Expect(err).ToNot(HaveOccurred())
	})

	when("deploying a basic Composer app", func() {
		it("it deploys using defaults and installs a package using Composer", func() {
			err := app.Start()
			fmt.Printf("%v", err)
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

			// ensure correct version of PHP is installed
			// Expect(app.BuildLogs()).To(MatchRegexp(`----->.*PHP.*7\.2\.\d+.*Contributing.* to layer`))

			// ensure X-Powered-By header is removed so as not to leak information
			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("OK"))
		})

		it.After(func() {
			//Expect(app.Destroy()).To(Succeed()) //Only destroy app if the test passed to leave artifacts to debug
		})
	})
}
