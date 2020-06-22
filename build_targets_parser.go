package gobuild

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type BuildTargetsParser struct{}

func NewBuildTargetsParser() BuildTargetsParser {
	return BuildTargetsParser{}
}

func (p BuildTargetsParser) Parse(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{"."}, nil
		}

		return nil, fmt.Errorf("failed to read buildpack.yml: %w", err)
	}

	var config struct {
		Go struct {
			Targets []string `yaml:"targets"`
		} `yaml:"go"`
	}

	err = yaml.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode buildpack.yml: %w", err)
	}

	if len(config.Go.Targets) == 0 {
		return []string{"."}, nil
	}

	for index, target := range config.Go.Targets {
		if strings.HasPrefix(target, string(filepath.Separator)) {
			return nil, fmt.Errorf("failed to determine build targets: %q is an absolute path, targets must be relative to the source directory", target)
		}

		config.Go.Targets[index] = fmt.Sprintf("./%s", filepath.Clean(target))
	}

	return config.Go.Targets, nil
}
