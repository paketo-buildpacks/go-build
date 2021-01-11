package gobuild

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/mattn/go-shellwords"
)

//go:generate faux --interface TargetManager --output fakes/target_manager.go
type TargetManager interface {
	CleanAndValidate(targets []string, workingDir string) ([]string, error)
	GenerateDefaults(workingDir string) ([]string, error)
}

//go:generate faux --interface BuildpackYMLParser --output fakes/buildpack_yml_parser.go
type BuildpackYMLParser interface {
	Parse(workingDir string) (BuildConfiguration, error)
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

func (p BuildConfigurationParser) Parse(workingDir string) (BuildConfiguration, error) {
	var buildConfiguration BuildConfiguration

	_, err := os.Stat(filepath.Join(workingDir, "buildpack.yml"))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return BuildConfiguration{}, err
		}
	} else {
		buildConfiguration, err = p.buildpackYMLParser.Parse(workingDir)
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

	if val, ok := os.LookupEnv("BP_GO_BUILD_FLAGS"); ok {
		shellwordsParser := shellwords.NewParser()
		shellwordsParser.ParseEnv = true
		buildConfiguration.Flags, err = shellwordsParser.Parse(val)
		if err != nil {
			return BuildConfiguration{}, err
		}
	}

	if val, ok := os.LookupEnv("BP_GO_BUILD_IMPORT_PATH"); ok {
		buildConfiguration.ImportPath = val
	}

	return buildConfiguration, nil
}
