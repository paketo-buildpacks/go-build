package gobuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildkite/interpolate"
	"gopkg.in/yaml.v2"
)

type GoBuildpackYMLParser struct{}

func NewGoBuildpackYMLParser() GoBuildpackYMLParser {
	return GoBuildpackYMLParser{}
}

func (p GoBuildpackYMLParser) Parse(workingDir string) (BuildConfiguration, error) {
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

	return BuildConfiguration{
		Targets:    config.Go.Targets,
		Flags:      config.Go.Build.Flags,
		ImportPath: config.Go.Build.ImportPath,
	}, nil
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
