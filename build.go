package gobuild

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

const (
	// JammyStaticStackID is the ID for the Cloud Native Buildpacks jammy static stack.
	JammyStaticStackID = "io.buildpacks.stacks.jammy.static"
)

//go:generate faux --interface BuildProcess --output fakes/build_process.go
type BuildProcess interface {
	Execute(config GoBuildConfiguration) (binaries []string, err error)
}

//go:generate faux --interface PathManager --output fakes/path_manager.go
type PathManager interface {
	Setup(workspace, importPath string) (goPath, path string, err error)
	Teardown(goPath string) error
}

//go:generate faux --interface SourceRemover --output fakes/source_remover.go
type SourceRemover interface {
	Clear(path string) error
}

//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go
type SBOMGenerator interface {
	Generate(dir string) (sbom.SBOM, error)
}

func Build(
	parser ConfigurationParser,
	buildProcess BuildProcess,
	pathManager PathManager,
	clock chronos.Clock,
	logs scribe.Emitter,
	sourceRemover SourceRemover,
	sbomGenerator SBOMGenerator,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logs.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		targetsLayer, err := context.Layers.Get(TargetsLayerName)
		if err != nil {
			return packit.BuildResult{}, err
		}

		targetsLayer.Launch = true

		goCacheLayer, err := context.Layers.Get(GoCacheLayerName)
		if err != nil {
			return packit.BuildResult{}, err
		}

		goCacheLayer.Cache = true

		// Parse the BuildConfiguration from the environment again since a prior
		// step may have augmented the configuration.
		configuration, err := parser.Parse(context.BuildpackInfo.Version, context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, packit.Fail.WithMessage("failed to parse build configuration: %w", err)
		}

		goPath, path, err := pathManager.Setup(context.WorkingDir, configuration.ImportPath)
		if err != nil {
			return packit.BuildResult{}, err
		}

		config := GoBuildConfiguration{
			Workspace:           path,
			Output:              filepath.Join(targetsLayer.Path, "bin"),
			GoPath:              goPath,
			GoCache:             goCacheLayer.Path,
			Flags:               configuration.Flags,
			Targets:             configuration.Targets,
			WorkspaceUseModules: configuration.WorkspaceUseModules,
		}

		if isStaticStack(context.Stack) && !containsFlag(config.Flags, "-buildmode") {
			config.DisableCGO = true
			config.Flags = append(config.Flags, "-buildmode", "default")
		}

		binaries, err := buildProcess.Execute(config)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = pathManager.Teardown(goPath)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = sourceRemover.Clear(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logs.GeneratingSBOM(filepath.Join(targetsLayer.Path, "bin"))

		var sbomContent sbom.SBOM
		duration, err := clock.Measure(func() error {
			sbomContent, err = sbomGenerator.Generate(filepath.Join(targetsLayer.Path, "bin"))
			return err
		})
		if err != nil {
			return packit.BuildResult{}, err
		}
		logs.Action("Completed in %s", duration.Round(time.Millisecond))
		logs.Break()

		logs.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)
		targetsLayer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
		if err != nil {
			return packit.BuildResult{}, err
		}

		shouldReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.BuildResult{}, err
		}

		var processes []packit.Process
		for index, binary := range binaries {
			processes = append(processes, packit.Process{
				Type:    filepath.Base(binary),
				Command: binary,
				Direct:  true,
				Default: index == 0 && !shouldReload,
			})

			if shouldReload {
				processes = append(processes, packit.Process{
					Type:    fmt.Sprintf("reload-%s", filepath.Base(binary)),
					Command: "watchexec",
					Args: []string{
						"--restart",
						"--watch", context.WorkingDir,
						"--watch", filepath.Dir(binary),
						"--shell", "none",
						"--",
						binary},
					Direct:  true,
					Default: index == 0,
				})
			}
		}

		logs.LaunchProcesses(processes)

		return packit.BuildResult{
			Layers: []packit.Layer{targetsLayer, goCacheLayer},
			Launch: packit.LaunchMetadata{
				Processes: processes,
			},
		}, nil
	}
}

func isStaticStack(stack string) bool {
	return stack == JammyStaticStackID
}
