package gobuild

import (
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface TargetsParser --output fakes/targets_parser.go
type TargetsParser interface {
	Parse(path string) (targets []string, err error)
}

func Detect(targetsParser TargetsParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		targets, err := targetsParser.Parse(filepath.Join(context.WorkingDir, "buildpack.yml"))
		if err != nil {
			return packit.DetectResult{}, err
		}

		for _, target := range targets {
			files, err := filepath.Glob(filepath.Join(target, "*.go"))
			if err != nil {
				return packit.DetectResult{}, err
			}

			if len(files) == 0 {
				return packit.DetectResult{}, packit.Fail
			}
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "go",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
				},
			},
		}, nil
	}
}
