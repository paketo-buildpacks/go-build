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
	Execute(workspace, output, goPath, goCache string, targets []string) (command string, err error)
}

//go:generate faux --interface PathManager --output fakes/path_manager.go
type PathManager interface {
	Setup(workspace string) (goPath, path string, err error)
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
	parser TargetsParser,
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
			targets, err := parser.Parse(filepath.Join(context.WorkingDir, "buildpack.yml"))
			if err != nil {
				return packit.BuildResult{}, err
			}

			goPath, path, err := pathManager.Setup(context.WorkingDir)
			if err != nil {
				return packit.BuildResult{}, err
			}

			command, err := buildProcess.Execute(path, filepath.Join(targetsLayer.Path, "bin"), goPath, goCacheLayer.Path, targets)
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
