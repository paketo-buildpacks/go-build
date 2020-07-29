package gobuild

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type BuildConfigurationParser struct{}

func NewBuildConfigurationParser() BuildConfigurationParser {
	return BuildConfigurationParser{}
}

func (p BuildConfigurationParser) Parse(path string) ([]string, []string, error) {
	var targets []string
	if len(os.Getenv("BP_GO_TARGETS")) > 0 {
		targets = strings.Split(os.Getenv("BP_GO_TARGETS"), ":")
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if len(targets) == 0 {
				targets = []string{"."}
			}

			return targets, nil, nil
		}

		return nil, nil, fmt.Errorf("failed to read buildpack.yml: %w", err)
	}

	var config struct {
		Go struct {
			Targets []string `yaml:"targets"`
			Build   struct {
				Flags []string `yaml:"flags"`
			} `yaml:"build"`
		} `yaml:"go"`
	}

	err = yaml.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode buildpack.yml: %w", err)
	}

	if len(targets) > 0 {
		config.Go.Targets = targets
	}

	var buildFlags []string
	for _, flag := range config.Go.Build.Flags {
		buildFlags = append(buildFlags, splitFlags(flag)...)
	}
	config.Go.Build.Flags = buildFlags

	for index, target := range config.Go.Targets {
		if strings.HasPrefix(target, string(filepath.Separator)) {
			return nil, nil, fmt.Errorf("failed to determine build targets: %q is an absolute path, targets must be relative to the source directory", target)
		}
		config.Go.Targets[index] = fmt.Sprintf("./%s", filepath.Clean(target))
	}

	if len(config.Go.Targets) == 0 {
		config.Go.Targets = []string{"."}
	}

	return config.Go.Targets, config.Go.Build.Flags, nil
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
