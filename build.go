package gobuild

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface BuildProcess --output fakes/build_process.go
type BuildProcess interface {
	Execute(config GoBuildConfiguration) (binaries []string, err error)
}

//go:generate faux --interface ChecksumCalculator --output fakes/checksum_calculator.go
type ChecksumCalculator interface {
	Sum(paths ...string) (string, error)
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
	checksumCalculator ChecksumCalculator,
	pathManager PathManager,
	clock chronos.Clock,
	logs scribe.Emitter,
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

		sum, err := checksumCalculator.Sum(targetsLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		cachedSha, _ := targetsLayer.Metadata["cache_sha"].(string)
		if cachedSha != sum {
			targetsLayer.Metadata = map[string]interface{}{
				"cache_sha": sum,
				"built_at":  clock.Now().Format(time.RFC3339Nano),
			}
		}

		err = pathManager.Teardown(goPath)
		if err != nil {
			return packit.BuildResult{}, err
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

		shouldReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.BuildResult{}, err
		}

		if shouldReload && context.Stack == TinyStackName {
			return packit.BuildResult{}, fmt.Errorf("cannot enable live reload on stack '%s': stack does not support watchexec", context.Stack)
		}

		if shouldReload {
			processes = []packit.Process{
				{
					Type:    "web",
					Command: fmt.Sprintf("watchexec --restart --watch %s --watch %s '%s'", context.WorkingDir, filepath.Dir(binaries[0]), binaries[0]),
					Direct:  context.Stack == TinyStackName,
				},
			}
		}

		for _, binary := range binaries {
			processes = append(processes, packit.Process{
				Type:    filepath.Base(binary),
				Command: binary,
				Direct:  context.Stack == TinyStackName,
			})

			if shouldReload {
				processes = append(processes, packit.Process{
					Type:    fmt.Sprintf("reload-%s", filepath.Base(binary)),
					Command: fmt.Sprintf("watchexec --restart --watch %s --watch %s '%s'", context.WorkingDir, filepath.Dir(binary), binary),
					Direct:  context.Stack == TinyStackName,
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
