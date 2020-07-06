package main

import (
	"os"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
)

func main() {
	logEmitter := gobuild.NewLogEmitter(os.Stdout)
	configParser := gobuild.NewBuildConfigurationParser()

	packit.Run(
		gobuild.Detect(
			configParser,
		),
		gobuild.Build(
			gobuild.NewGoBuildProcess(
				pexec.NewExecutable("go"),
				logEmitter,
				chronos.DefaultClock,
			),
			gobuild.NewGoPathManager(os.TempDir()),
			chronos.DefaultClock,
			fs.NewChecksumCalculator(),
			logEmitter,
			configParser,
			gobuild.NewSourceDeleter(),
		),
	)
}
