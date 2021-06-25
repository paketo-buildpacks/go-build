package gobuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/buildkite/interpolate"
	"github.com/paketo-buildpacks/packit/scribe"
	"gopkg.in/yaml.v2"
)

type GoBuildpackYMLParser struct {
	logger scribe.Emitter
}

func NewGoBuildpackYMLParser(logger scribe.Emitter) GoBuildpackYMLParser {
	return GoBuildpackYMLParser{
		logger: logger,
	}
}

func (p GoBuildpackYMLParser) Parse(buildpackVersion, workingDir string) (BuildConfiguration, error) {
	file, err := os.Open(filepath.Join(workingDir, "buildpack.yml"))
	if err != nil {
		return BuildConfiguration{}, fmt.Errorf("failed to read buildpack.yml: %w", err)
	}

	var config struct {
		Go struct {
			Targets []string `yaml:"targets"`
			Build   struct {
				Flags      []string `yaml:"flags"`
				ImportPath string   `yaml:"import-path"`
			} `yaml:"build"`
		} `yaml:"go"`
	}

	err = yaml.NewDecoder(file).Decode(&config)
	if err != nil {
		return BuildConfiguration{}, fmt.Errorf("failed to decode buildpack.yml: %w", err)
	}

	var buildFlags []string
	for _, flag := range config.Go.Build.Flags {
		for _, f := range splitFlags(flag) {
			interpolatedFlag, err := interpolate.Interpolate(interpolate.NewSliceEnv(os.Environ()), f)
			if err != nil {
				return BuildConfiguration{}, fmt.Errorf("environment variable expansion failed: %w", err)
			}
			buildFlags = append(buildFlags, interpolatedFlag)
		}
	}
	config.Go.Build.Flags = buildFlags

	buildConfiguration := BuildConfiguration{
		Targets:    config.Go.Targets,
		Flags:      config.Go.Build.Flags,
		ImportPath: config.Go.Build.ImportPath,
	}

	if buildConfiguration.Targets != nil || buildConfiguration.Flags != nil || buildConfiguration.ImportPath != "" {
		nextMajorVersion := semver.MustParse(buildpackVersion).IncMajor()
		p.logger.Process("WARNING: Setting the Go Build configurations such as targets, build flags, and import path through buildpack.yml will be deprecated soon in Go Build Buildpack v%s.", nextMajorVersion.String())
		p.logger.Process("Please specify these configuration options through environment variables instead. See README.md or the documentation on paketo.io for more information.")
		p.logger.Break()
	}

	return buildConfiguration, nil
}

func splitFlags(flag string) []string {
	parts := strings.SplitN(flag, "=", 2)
	if len(parts) == 2 {
		if len(parts[1]) >= 2 {
			if c := parts[1][len(parts[1])-1]; parts[1][0] == c && (c == '"' || c == '\'') {
				parts[1] = parts[1][1 : len(parts[1])-1]
			}
		}
	}

	return parts
}
