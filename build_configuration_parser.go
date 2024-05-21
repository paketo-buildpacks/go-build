package gobuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/paketo-buildpacks/packit/v2/fs"
)

//go:generate faux --interface TargetManager --output fakes/target_manager.go
type TargetManager interface {
	CleanAndValidate(targets []string, workingDir string) ([]string, error)
	GenerateDefaults(workingDir string) ([]string, error)
}

type BuildConfiguration struct {
	Targets             []string
	Flags               []string
	ImportPath          string
	WorkspaceUseModules []string
}

type BuildConfigurationParser struct {
	targetManager TargetManager
}

func NewBuildConfigurationParser(targetManager TargetManager) BuildConfigurationParser {
	return BuildConfigurationParser{
		targetManager: targetManager,
	}
}

func (p BuildConfigurationParser) Parse(buildpackVersion, workingDir string) (BuildConfiguration, error) {
	bpYML, err := fs.Exists(filepath.Join(workingDir, "buildpack.yml"))
	if err != nil {
		return BuildConfiguration{}, fmt.Errorf("failed to check for buildpack.yml: %w", err)
	}
	if bpYML {
		return BuildConfiguration{}, fmt.Errorf("working directory contains deprecated 'buildpack.yml'; use environment variables for configuration")
	}

	var buildConfiguration BuildConfiguration
	if val, ok := os.LookupEnv("BP_GO_TARGETS"); ok {
		buildConfiguration.Targets = filepath.SplitList(val)
	}

	if len(buildConfiguration.Targets) > 0 {
		buildConfiguration.Targets, err = p.targetManager.CleanAndValidate(buildConfiguration.Targets, workingDir)
		if err != nil {
			return BuildConfiguration{}, err
		}
	} else {
		buildConfiguration.Targets, err = p.targetManager.GenerateDefaults(workingDir)
		if err != nil {
			return BuildConfiguration{}, err
		}
	}

	buildConfiguration.Flags, err = parseFlagsFromEnvVars(buildConfiguration.Flags)
	if err != nil {
		return BuildConfiguration{}, err
	}

	if val, ok := os.LookupEnv("BP_GO_BUILD_IMPORT_PATH"); ok {
		buildConfiguration.ImportPath = val
	}

	if val, ok := os.LookupEnv("BP_GO_WORK_USE"); ok {
		buildConfiguration.WorkspaceUseModules = filepath.SplitList(val)
	}

	return buildConfiguration, nil
}

func containsFlag(flags []string, match string) bool {
	for _, flag := range flags {
		if strings.HasPrefix(flag, match) {
			return true
		}
	}
	return false
}

func parseFlagsFromEnvVars(flags []string) ([]string, error) {
	shellwordsParser := shellwords.NewParser()
	shellwordsParser.ParseEnv = true

	if buildFlags, ok := os.LookupEnv("BP_GO_BUILD_FLAGS"); ok {
		var err error
		flags, err = shellwordsParser.Parse(buildFlags)
		if err != nil {
			return nil, err
		}
	}

	if ldFlags, ok := os.LookupEnv("BP_GO_BUILD_LDFLAGS"); ok {
		parsed, err := shellwordsParser.Parse(fmt.Sprintf(`-ldflags="%s"`, ldFlags))
		if err != nil {
			return nil, err
		}
		if len(parsed) != 1 {
			return nil, fmt.Errorf("BP_GO_BUILD_LDFLAGS value (%s) could not be parsed: value contains multiple words", ldFlags)
		}

		for i, flag := range flags {
			if strings.HasPrefix(flag, "-ldflags") {
				// Replace value from BP_GO_BUILD_FLAGS or buildpack.yml with value from
				// BP_GO_BUILD_LDFLAGS because BP_GO_BUILD_LDFLAGS takes precedence
				flags[i] = parsed[0]
			}
		}

		if !containsFlag(flags, "-ldflags") {
			flags = append(flags, parsed[0])
		}
	}
	return flags, nil
}
