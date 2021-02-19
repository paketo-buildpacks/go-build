package gobuild

import (
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

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{{
					Name: "go",
					Metadata: map[string]interface{}{
						"build": true,
					},
				}},
			},
		}, nil
	}
}
