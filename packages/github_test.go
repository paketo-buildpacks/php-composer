package packages

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitGithub(t *testing.T) {
	spec.Run(t, "Github", testGithub, spec.Report(report.Terminal{}))
}

func testGithub(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("a github oauth token is supplied", func() {
		it("validates the github token successfully when the token is correct", func(){
			response := `{
				"resources": {
					"core": {
						"limit": 5000,
						"remaining": 5000,
						"reset": 1560873755
					},
					"search": {
						"limit": 30,
						"remaining": 30,
						"reset": 1560870215
					},
					"graphql": {
						"limit": 5000,
						"remaining": 5000,
						"reset": 1560873755
					},
					"integration_manifest": {
						"limit": 5000,
						"remaining": 5000,
						"reset": 1560873755
					}
				},
				"rate": {
					"limit": 5000,
					"remaining": 5000,
					"reset": 1560873755
				}
			}`

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, response)
			}))
			defer ts.Close()

			github, err := NewGithub("FAKE", ts.URL)
			Expect(err).NotTo(HaveOccurred())

			validation, err := github.validateToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(validation).To(BeTrue())

			limitOk, err := github.checkRateLimit()
			Expect(err).NotTo(HaveOccurred())
			Expect(limitOk).To(BeTrue())
		})

		it("validates the github token successfully when the token is correct and returns false if rate limit is met", func(){
			response := `{
				"resources": {
					"core": {
						"limit": 5000,
						"remaining": 0,
						"reset": 1560873755
					},
					"search": {
						"limit": 30,
						"remaining": 30,
						"reset": 1560870215
					},
					"graphql": {
						"limit": 5000,
						"remaining": 5000,
						"reset": 1560873755
					},
					"integration_manifest": {
						"limit": 5000,
						"remaining": 5000,
						"reset": 1560873755
					}
				},
				"rate": {
					"limit": 5000,
					"remaining": 0,
					"reset": 1560873755
				}
			}`

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, response)
			}))
			defer ts.Close()

			github, err := NewGithub("FAKE", ts.URL)
			Expect(err).NotTo(HaveOccurred())

			validation, err := github.validateToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(validation).To(BeTrue())

			limitOk, err := github.checkRateLimit()
			Expect(err).NotTo(HaveOccurred())
			Expect(limitOk).To(BeFalse())
		})

		it("validates the github token successfully when the token is NOT correct", func(){
			response := `{
				"message": "Bad credentials",
				"documentation_url": "https://developer.github.com/v3"
			}`

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, response)
			}))
			defer ts.Close()

			github, err := NewGithub("FAKE", ts.URL)
			Expect(err).NotTo(HaveOccurred())

			Expect(github.validateToken()).To(BeFalse())
		})
	})
}
