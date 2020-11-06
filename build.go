package gobuild

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
)

//go:generate faux --interface BuildProcess --output fakes/build_process.go
type BuildProcess interface {
	Execute(config GoBuildConfiguration) (command string, err error)
}

//go:generate faux --interface PathManager --output fakes/path_manager.go
type PathManager interface {
	Setup(workspace, importPath string) (goPath, path string, err error)
	Teardown(goPath string) error
}

//go:generate faux --interface ChecksumCalculator --output fakes/checksum_calculator.go
type ChecksumCalculator interface {
	Sum(path string) (sha string, err error)
}

//go:generate faux --interface SourceRemover --output fakes/source_remover.go
type SourceRemover interface {
	Clear(path string) error
}

func Build(
	buildProcess BuildProcess,
	pathManager PathManager,
	clock chronos.Clock,
	checksumCalculator ChecksumCalculator,
	logs LogEmitter,
	sourceRemover SourceRemover,
) packit.BuildFunc {

	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logs.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		targetsLayer, err := context.Layers.Get(TargetsLayerName, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		goCacheLayer, err := context.Layers.Get(GoCacheLayerName, packit.CacheLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		checksum, err := checksumCalculator.Sum(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		previousSum, _ := targetsLayer.Metadata[WorkspaceSHAKey].(string)
		if checksum != previousSum {
			var configuration BuildConfiguration

			entry := context.Plan.Entries[0]

			for _, target := range entry.Metadata["targets"].([]interface{}) {
				configuration.Targets = append(configuration.Targets, target.(string))
			}

			if flags, ok := entry.Metadata["flags"]; ok {
				for _, flag := range flags.([]interface{}) {
					configuration.Flags = append(configuration.Flags, flag.(string))
				}
			}

			if importPath, ok := entry.Metadata["import-path"]; ok {
				configuration.ImportPath = importPath.(string)
			}

			goPath, path, err := pathManager.Setup(context.WorkingDir, configuration.ImportPath)
			if err != nil {
				return packit.BuildResult{}, err
			}

			command, err := buildProcess.Execute(GoBuildConfiguration{
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
				WorkspaceSHAKey: checksum,
				"built_at":      clock.Now().Format(time.RFC3339Nano),
				"command":       command,
			}
		} else {
			logs.Process("Reusing cached layer %s", targetsLayer.Path)
			logs.Break()
		}

		command, ok := targetsLayer.Metadata["command"].(string)
		if !ok {
			return packit.BuildResult{}, errors.New("failed to identify start command from reused layer metadata")
		}

		err = sourceRemover.Clear(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logs.Process("Assigning launch processes")
		logs.Subprocess("web: %s", command)

		return packit.BuildResult{
			Plan:   context.Plan,
			Layers: []packit.Layer{targetsLayer, goCacheLayer},
			Processes: []packit.Process{
				{
					Type:    "web",
					Command: command,
					Direct:  context.Stack == TinyStackName,
				},
			},
		}, nil
	}
}
