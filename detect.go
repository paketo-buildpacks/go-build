package gobuild

import (
	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface ConfigurationParser --output fakes/configuration_parser.go
type ConfigurationParser interface {
	Parse(workingDir string) (BuildConfiguration, error)
}

func Detect(parser ConfigurationParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		configuration, err := parser.Parse(context.WorkingDir)
		if err != nil {
			return packit.DetectResult{}, packit.Fail.WithMessage("failed to parse build configuration: %w", err)
		}

		metadata := map[string]interface{}{
			"targets": configuration.Targets,
		}

		if flags := configuration.Flags; flags != nil {
			metadata["flags"] = flags
		}

		if importPath := configuration.ImportPath; importPath != "" {
			metadata["import-path"] = importPath
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{{Name: "go-build"}},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "go",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
					{
						Name:     "go-build",
						Metadata: metadata,
					},
				},
			},
		}, nil
	}
}
