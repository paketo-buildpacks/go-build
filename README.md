# Go Build Cloud Native Buildpack

The Go Build CNB executes the `go build` compilation process for Go programs.
The buildpack builds the source code in the application directory into an
executable and sets it as the start command for the image.

## Integration

The Go Build CNB does not provide any dependencies. However, in order to
execute the `go build` compilation process, the buildpack requires the `go`
dependency that can be provided by a buildpack like the [Go Distribution
CNB](https://github.com/paketo-buildpacks/go-dist).

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh
```

This builds the buildpack's Go source using `GOOS=linux` by default. You can
supply another value as the first argument to `package.sh`.

## Go Build Configuration
Please set the following environment
variables at build time either directly (ex. `pack build my-app --env
BP_ENVIRONMENT_VARIABLE=some-value`) or through a [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md)

### `BP_GO_BUILD_LDFLAGS`
The `BP_GO_BUILD_LDFLAGS` variable allows you to set a value for the `-ldflags` build flag
when compiling your program.

```shell
BP_GO_BUILD_LDFLAGS= -X main.variable=some-value
```

_Note: Specifying the `Go Build` configuration through `buildpack.yml` configuration
will be deprecated in Go Build Buildpack v1.0.0._

To migrate from using `buildpack.yml` please set the following environment
variables at build time either directly (ex. `pack build my-app --env
BP_ENVIRONMENT_VARIABLE=some-value`) or through a [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md)

### `BP_GO_TARGETS`
The `BP_GO_TARGETS` variable allows you to specify multiple programs to be
compiled. The first target will be used as the start command for the image.

```shell
BP_GO_TARGETS=./cmd/web-server:./cmd/debug-server
```

This will replace the following structure in `buildpack.yml`:
```yaml
go:
  targets:
  - ./cmd/web-server
  - ./cmd/debug-server
```

### `BP_GO_BUILD_FLAGS`
The `BP_GO_BUILD_FLAGS` variable allows you to override the default build flags
when compiling your program.

```shell
BP_GO_BUILD_FLAGS= -buildmode=default -tags=paketo -ldflags="-X main.variable=some-value"
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

### `BP_GO_BUILD_IMPORT_PATH`
The `BP_GO_BUILD_IMPORT_PATH` allows you to specify an import path for your
application. This is necessary if you are building a $GOPATH application that
imports its own sub-packages.

```shell
BP_GO_BUILD_IMPORT_PATH= example.com/some-app
```

This will replace the following structure in `buildpack.yml`:
```yaml
go:
  build:
    import-path: example.com/some-app
```

### `BP_KEEP_FILES`
The `BP_KEEP_FILES` variable allows to you to specity a path list of files
(including file globs) that you would like to appear in the workspace of the
final image. This will allow you to perserve static assests.

`BP_KEEP_FILES=assets/*:public/*`
