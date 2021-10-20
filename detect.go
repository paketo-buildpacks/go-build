package gobuild

import (
	"fmt"
	"os"
	"strconv"

	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface ConfigurationParser --output fakes/configuration_parser.go
type ConfigurationParser interface {
	Parse(buildpackVersion, workingDir string) (BuildConfiguration, error)
}

func Detect(parser ConfigurationParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		if _, err := parser.Parse(context.BuildpackInfo.Version, context.WorkingDir); err != nil {
			return packit.DetectResult{}, packit.Fail.WithMessage("failed to parse build configuration: %w", err)
		}

		requirements := []packit.BuildPlanRequirement{
			{
				Name: "go",
				Metadata: map[string]interface{}{
					"build": true,
				},
			},
		}

		shouldEnableReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.DetectResult{}, err
		}

		if shouldEnableReload && context.Stack == TinyStackName {
			return packit.DetectResult{}, fmt.Errorf("cannot enable live reload on stack '%s': stack does not support watchexec", context.Stack)
		}

		if shouldEnableReload {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "watchexec",
				Metadata: map[string]interface{}{
					"launch": true,
				},
			})
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Requires: requirements,
			},
		}, nil
	}
}

func checkLiveReloadEnabled() (bool, error) {
	if reload, ok := os.LookupEnv("BP_LIVE_RELOAD_ENABLED"); ok {
		shouldEnableReload, err := strconv.ParseBool(reload)
		if err != nil {
			return false, fmt.Errorf("failed to parse BP_LIVE_RELOAD_ENABLED value %s: %w", reload, err)
		}
		return shouldEnableReload, nil
	}
	return false, nil
}
