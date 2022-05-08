package gobuild

import (
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/reload"
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

		if watchExecReq, shouldEnableReload, err := reload.AddWatchexec(); err != nil {
			return packit.DetectResult{}, err
		} else if shouldEnableReload {
			requirements = append(requirements, watchExecReq)
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Requires: requirements,
			},
		}, nil
	}
}
