package gobuild

import (
	"fmt"
	"os"
	"path/filepath"

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
		interpolatedFlag, err := interpolate.Interpolate(interpolate.NewSliceEnv(os.Environ()), flag)
		if err != nil {
			return BuildConfiguration{}, fmt.Errorf("environment variable expansion failed: %w", err)
		}
		buildFlags = append(buildFlags, interpolatedFlag)
	}
	config.Go.Build.Flags = buildFlags

	return BuildConfiguration{
		Targets:    config.Go.Targets,
		Flags:      config.Go.Build.Flags,
		ImportPath: config.Go.Build.ImportPath,
	}, nil
}
