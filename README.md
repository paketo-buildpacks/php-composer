# PHP Composer Cloud Native Buildpack

The PHP Composer CNB provides composer as a dependency. Downstream buildpacks
can require the nginx dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the PHP Composer dependency is "composer". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "composer"

  # The version of the PHP Composer dependency is not required. In the case it
  # is not specified, the buildpack will provide the default version, which can
  # be seen in the buildpack.toml file.
  # If you wish to request a specific version, the buildpack supports
  # specifying a semver constraint in the form of "1.*", "1.10.*", or even
  # "1.10.5".
  version = "1.10.5"

  # The PHP Composer buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the PHP Composer
    # dependency is available on the $PATH for subsequent buildpacks during their
    # build phase. If you are writing a buildpack that needs to run PHP Composer
    # during its build process, this flag should be set to true.
    build = true
```

## Usage

To package this buildpack for consumption:

```bash
$ ./scripts/package.sh
```

This builds the buildpack's Go source using GOOS=linux by default. You can supply another value as the first argument to package.sh.
