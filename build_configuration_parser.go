package gobuild

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-shellwords"
)

//go:generate faux --interface TargetManager --output fakes/target_manager.go
type TargetManager interface {
	CleanAndValidate(targets []string, workingDir string) ([]string, error)
	GenerateDefaults(workingDir string) ([]string, error)
}

//go:generate faux --interface BuildpackYMLParser --output fakes/buildpack_yml_parser.go
type BuildpackYMLParser interface {
	Parse(buildpackVersion, workingDir string) (BuildConfiguration, error)
}

type BuildConfiguration struct {
	Targets    []string
	Flags      []string
	ImportPath string
}

type BuildConfigurationParser struct {
	targetManager      TargetManager
	buildpackYMLParser BuildpackYMLParser
}

func NewBuildConfigurationParser(targetManager TargetManager, buildpackYMLParser BuildpackYMLParser) BuildConfigurationParser {
	return BuildConfigurationParser{
		targetManager:      targetManager,
		buildpackYMLParser: buildpackYMLParser,
	}
}

func (p BuildConfigurationParser) Parse(buildpackVersion, workingDir string) (BuildConfiguration, error) {
	var buildConfiguration BuildConfiguration

	_, err := os.Stat(filepath.Join(workingDir, "buildpack.yml"))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return BuildConfiguration{}, err
		}
	} else {
		buildConfiguration, err = p.buildpackYMLParser.Parse(buildpackVersion, workingDir)
		if err != nil {
			return BuildConfiguration{}, err
		}
	}

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

	ldFlags, ldFlagsSet := os.LookupEnv("BP_GO_BUILD_LDFLAGS")
	buildFlags, buildFlagsSet := os.LookupEnv("BP_GO_BUILD_FLAGS")

	if buildFlagsSet {
		shellwordsParser := shellwords.NewParser()
		shellwordsParser.ParseEnv = true
		buildConfiguration.Flags, err = shellwordsParser.Parse(buildFlags)
		if err != nil {
			return BuildConfiguration{}, err
		}
	}

	contains := func(flags []string, match string) bool {
		for _, flag := range flags {
			if strings.HasPrefix(flag, match) {
				return true
			}
		}

		return false
	}

	if ldFlagsSet {
		shellwordsParser := shellwords.NewParser()
		shellwordsParser.ParseEnv = true
		parsedLdFlags, err := shellwordsParser.Parse(fmt.Sprintf(`-ldflags="%s"`, ldFlags))
		if err != nil {
			return BuildConfiguration{}, err
		}
		if len(parsedLdFlags) != 1 {
			return BuildConfiguration{}, fmt.Errorf("BP_GO_BUILD_LDFLAGS value (%s) could not be parsed: value contains multiple words", ldFlags)
		}

		for i, flag := range buildConfiguration.Flags {
			if strings.HasPrefix(flag, "-ldflags") {
				// BP_GO_BUILD_LDFLAGS takes precedent over -ldflags in BP_GO_BUILD_FLAGS

				buildConfiguration.Flags[i] = parsedLdFlags[0]
			}
		}

		if !contains(buildConfiguration.Flags, "-ldflags") {
			buildConfiguration.Flags = append(buildConfiguration.Flags, parsedLdFlags[0])
		}
	}

	if val, ok := os.LookupEnv("BP_GO_BUILD_IMPORT_PATH"); ok {
		buildConfiguration.ImportPath = val
	}

	return buildConfiguration, nil
}
