package main

import (
	"os"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

func main() {
	logEmitter := scribe.NewEmitter(os.Stdout)
	configParser := gobuild.NewBuildConfigurationParser(gobuild.NewGoTargetManager(), gobuild.NewGoBuildpackYMLParser(logEmitter))

	packit.Run(
		gobuild.Detect(
			configParser,
		),
		gobuild.Build(
			configParser,
			gobuild.NewGoBuildProcess(
				pexec.NewExecutable("go"),
				logEmitter,
				chronos.DefaultClock,
			),
			gobuild.NewGoPathManager(os.TempDir()),
			chronos.DefaultClock,
			logEmitter,
			gobuild.NewSourceDeleter(),
		),
	)
}
