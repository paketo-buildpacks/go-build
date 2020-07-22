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
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{"."}, nil, nil
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

	var buildFlags []string
	for _, flag := range config.Go.Build.Flags {
		buildFlags = append(buildFlags, strings.SplitN(flag, "=", 2)...)
	}
	config.Go.Build.Flags = buildFlags

	if len(config.Go.Targets) == 0 {
		return []string{"."}, config.Go.Build.Flags, nil
	}

	for index, target := range config.Go.Targets {
		if strings.HasPrefix(target, string(filepath.Separator)) {
			return nil, nil, fmt.Errorf("failed to determine build targets: %q is an absolute path, targets must be relative to the source directory", target)
		}
		config.Go.Targets[index] = fmt.Sprintf("./%s", filepath.Clean(target))
	}

	if bpGoTargets := os.Getenv("BP_GO_TARGETS"); bpGoTargets != "" {
		config.Go.Targets = strings.Split(bpGoTargets, ":")
	}

	return config.Go.Targets, config.Go.Build.Flags, nil
}
