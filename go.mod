module github.com/paketo-buildpacks/go-build

go 1.16

replace github.com/paketo-buildpacks/packit/v2 => /Users/caseyj/git/paketo-buildpacks/packit

require (
	github.com/BurntSushi/toml v1.1.0
	github.com/mattn/go-shellwords v1.0.11-0.20201201010856-2c8720de5e83
	github.com/onsi/gomega v1.19.0
	github.com/paketo-buildpacks/occam v0.8.0
	github.com/paketo-buildpacks/packit/v2 v2.3.0
	github.com/sclevine/spec v1.4.0
)
