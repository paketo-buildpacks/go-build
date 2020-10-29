# Buildpack.yml to Environment Variables

## Proposal

Migrate to using environment variables to do all buildpack configuration and
get rid of `buildpack.yml`.

## Motivation

There are several reasons for making this switch.
1. There is already an existing RFC that proposes moving away from
   `buildpack.yml` as a configuration tool.
1. Environment variables appears to be the standard for configuration in other
   buildpack ecosystems such as Google Buildpacks and Heroku as well as the
   Paketo Java buildpacks. Making this change will align the buildpack with the
   rest of the buildpack ecosystem.
1. There is native support to pass environment variables to the buildpack
   either on a per run basis or by check configuration, in the form of
   `project.toml`, into source control.

## Implementation

The proposed environment variables for Go Build are as follow:

#### BP_GO_BUILD_FLAGS

```shell
$BP_GO_BUILD_FLAGS='-buildmode=default,-tags=paketo,-ldflags="-X main.variable=some-value"'
```

This will replace the following structure in `buildpack.yml`:

```yaml
go:
  build:
    flags:
    - -buildmode=default
    - -tags=paketo
    - -ldflags="-X main.variable=some-value"
```

#### BP_GO_BUILD_IMPORT_PATH

```shell
$BP_GO_BUILD_IMPORT_PATH=example.com/some-app
```

This will replace the following structure in `buildpack.yml`:

```yaml
go:
  build:
    import-path: example.com/some-app
```

#### BP_GO_TARGETS

```shell
$BP_GO_TARGETS=./first:./second
```

This will replace the following structure in `buildpack.yml`:

```yaml
go:
  targets:
  - ./cmd/web-server
  - ./cmd/debug-server
```

### Deprecation Strategy
In order to facilitate a smooth transition from `buildpack.yml`, the buildpack
should will support both configuration options with environment variables
taking priority or `buildpack.yml` until the 1.0 release of the buildpack. The
buildpack will detect whether or not the application has a `buildpack.yml` and
print a warning message which will include links to documentation on how to
upgrade and how to run builds with environment variable configuration. After
1.0, having a `buildpack.yml` will cause a detection failure and with a link to
the same documentation. This behavior will only last until the next patch
release of the buildpack.

## Source Material
* [Google buildpack configuration](https://github.com/GoogleCloudPlatform/buildpacks#language-idiomatic-configuration-options)
* [Paketo Java configuration](https://paketo.io/docs/buildpacks/language-family-buildpacks/java)
* [Heroku configuration](https://github.com/heroku/java-buildpack#customizing)

