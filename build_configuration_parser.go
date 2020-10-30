package gobuild

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildkite/interpolate"
	"gopkg.in/yaml.v2"
)

//go:generate faux --interface TargetManager --output fakes/target_manager.go
type TargetManager interface {
	CleanAndValidate(targets []string, workingDir string) ([]string, error)
	GenerateDefaults(workingDir string) ([]string, error)
}

type BuildConfiguration struct {
	Targets    []string
	Flags      []string
	ImportPath string
}

type BuildConfigurationParser struct {
	targetManager TargetManager
}

func NewBuildConfigurationParser(targetManager TargetManager) BuildConfigurationParser {
	return BuildConfigurationParser{
		targetManager: targetManager,
	}
}

func (p BuildConfigurationParser) Parse(workingDir string) (BuildConfiguration, error) {
	var targets []string
	if val, ok := os.LookupEnv("BP_GO_TARGETS"); ok {
		targets = filepath.SplitList(val)
	}

	file, err := os.Open(filepath.Join(workingDir, "buildpack.yml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if len(targets) == 0 {
				targets, err = p.targetManager.GenerateDefaults(workingDir)
				if err != nil {
					return BuildConfiguration{}, err
				}
			} else {
				targets, err = p.targetManager.CleanAndValidate(targets, workingDir)
				if err != nil {
					return BuildConfiguration{}, err
				}
			}
			return BuildConfiguration{Targets: targets}, nil
		}

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

	// This will use targets that it got from the env var over the targests set in buildpack.yml
	if len(targets) > 0 {
		config.Go.Targets, err = p.targetManager.CleanAndValidate(targets, workingDir)
		if err != nil {
			return BuildConfiguration{}, err
		}
	} else {
		config.Go.Targets, err = p.targetManager.CleanAndValidate(config.Go.Targets, workingDir)
		if err != nil {
			return BuildConfiguration{}, err
		}
	}

	if len(config.Go.Targets) == 0 {
		config.Go.Targets, err = p.targetManager.GenerateDefaults(workingDir)
		if err != nil {
			return BuildConfiguration{}, err
		}
	}

	env := interpolate.NewSliceEnv(os.Environ())

	var buildFlags []string
	for _, flag := range config.Go.Build.Flags {
		for _, f := range splitFlags(flag) {
			interpolatedFlag, err := interpolate.Interpolate(env, f)
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
