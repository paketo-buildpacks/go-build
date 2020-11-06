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

## Environment Variable Configuration
```
  # The BP_GO_TARGETS variable allows you to specify multiple programs to be
  # compiled. The first target will be used as the start command for the image.
  BP_GO_TARGETS=./cmd/web-server:./cmd/debug-server

  # The BP_INCLUDE_FILES variable allows to you to specity a path list of files
  # (including file globs) that you would like to appear in the workspace of the
  # final image. This will allow you to perserve static assests
  BP_INCLUDE_FILES=assets/*:public/*
```

## `buildpack.yml` Configurations

```yaml
go:
  # The go.targets property allows you to specify multiple programs to be
  # compiled. The first target will be used as the start command for the image.
  targets:
  - ./cmd/web-server
  - ./cmd/debug-server

  build:

    # The go.build.flags property allows you to override the default build
    # flags when compiling your program.
    flags:
    - -buildmode=default
    - -tags=paketo
    - -ldflags="-X main.variable=some-value"

    # The go.build.import-path property allows you to specify an import path
    # for your application. This is necessary if you are building a $GOPATH
    # application that imports its own sub-packages.
    import-path: example.com/some-app
```
