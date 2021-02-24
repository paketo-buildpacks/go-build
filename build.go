package gobuild

import (
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
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

func Build(
	parser ConfigurationParser,
	buildProcess BuildProcess,
	pathManager PathManager,
	clock chronos.Clock,
	logs LogEmitter,
	sourceRemover SourceRemover,
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

		binaries, err := buildProcess.Execute(GoBuildConfiguration{
			Workspace: path,
			Output:    filepath.Join(targetsLayer.Path, "bin"),
			GoPath:    goPath,
			GoCache:   goCacheLayer.Path,
			Flags:     configuration.Flags,
			Targets:   configuration.Targets,
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = pathManager.Teardown(goPath)
		if err != nil {
			return packit.BuildResult{}, err
		}

		targetsLayer.Metadata = map[string]interface{}{
			"built_at": clock.Now().Format(time.RFC3339Nano),
		}

		err = sourceRemover.Clear(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		processes := []packit.Process{
			{
				Type:    "web",
				Command: binaries[0],
				Direct:  context.Stack == TinyStackName,
			},
		}

		for _, binary := range binaries {
			processes = append(processes, packit.Process{
				Type:    filepath.Base(binary),
				Command: binary,
				Direct:  context.Stack == TinyStackName,
			})
		}

		logs.Process("Assigning launch processes")
		logs.ListProcesses(processes)

		return packit.BuildResult{
			Layers: []packit.Layer{targetsLayer, goCacheLayer},
			Launch: packit.LaunchMetadata{
				Processes: processes,
			},
		}, nil
	}
}
