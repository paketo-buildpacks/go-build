package main

import (
	"os"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

func main() {
	emitter := scribe.NewEmitter(os.Stdout)
	configParser := gobuild.NewBuildConfigurationParser(gobuild.NewGoTargetManager(), gobuild.NewGoBuildpackYMLParser(emitter))

	packit.Run(
		gobuild.Detect(
			configParser,
		),
		gobuild.Build(
			configParser,
			gobuild.NewGoBuildProcess(
				pexec.NewExecutable("go"),
				emitter,
				chronos.DefaultClock,
			),
			fs.NewChecksumCalculator(),
			gobuild.NewGoPathManager(os.TempDir()),
			chronos.DefaultClock,
			emitter,
			gobuild.NewSourceDeleter(),
		),
	)
}
