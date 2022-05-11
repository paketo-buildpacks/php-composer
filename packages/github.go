package packages

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	githubTimeout = 5
	githubURL     = "https://api.github.com/rate_limit"
)

type Github struct {
	Token string
	body  []byte
}

func NewDefaultGithub(token string) (Github, error) {
	return NewGithub(token, githubURL)
}

func NewGithub(token, url string) (Github, error) {
	result := Github{
		Token: token,
	}

	err := result.makeRequest(url)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (g *Github) makeRequest(url string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.Token))
	t := &http.Transport{Proxy: http.ProxyFromEnvironment}

	client := http.Client{Transport: t, Timeout: time.Second * githubTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	g.body, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (g *Github) checkRateLimit() (bool, error) {
	type GithubRateLimitResponse struct {
		Resources struct {
			Core struct {
				Limit     int `json:"limit"`
				Remaining int `json:"remaining"`
				Reset     int `json:"reset"`
			} `json:"core"`
		} `json:"resources"`
	}
	rateLimitResp := GithubRateLimitResponse{}

	if err := json.Unmarshal(g.body, &rateLimitResp); err != nil {
		return false, err
	}

	return rateLimitResp.Resources.Core.Remaining > 0, nil
}

func (g *Github) validateToken() (bool, error) {
	respMap := map[string]interface{}{}

	if err := json.Unmarshal(g.body, &respMap); err != nil {
		return false, err
	}

	_, ok := respMap["resources"]

	return ok, nil
}
