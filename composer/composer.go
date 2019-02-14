package composer

import "fmt"

const (
	DEPENDENCY       = "php-composer"
	CACHE_DEPENDENCY = "php-composer-cache"
	COMPOSER_LOCK    = "composer.lock"
	COMPOSER_JSON    = "composer.json"
	COMPOSER_PHAR    = "composer.phar"
)

type Composer struct {
	Runner  Runner
	appRoot string
}

type Runner interface {
	Run(bin, dir string, args ...string) error
}

func NewComposer(appRoot string) Composer {
	return Composer{
		appRoot: appRoot,
	}
}

func (c Composer) Install(args ...string) error {
	args = append([]string{COMPOSER_PHAR, "install", "--no-progress"}, args...)
	return c.Runner.Run("php", c.appRoot, args...)
}

func (c Composer) Version() error {
	return c.Runner.Run("php", c.appRoot, COMPOSER_PHAR, "-V")
}

func (c Composer) Global(args ...string) error {
	args = append([]string{COMPOSER_PHAR, "global", "require", "--no-progress"}, args...)
	return c.Runner.Run("php", c.appRoot, args...)
}

func (c Composer) Config(token string) error {
	return c.Runner.Run("php", c.appRoot, COMPOSER_PHAR, "config", "-g", "github-oauth.github.com", fmt.Sprintf(`"%s"`, token))
}

