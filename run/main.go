package main

import (
	"os"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type Generator struct{}

func (f Generator) Generate(dir string) (sbom.SBOM, error) {
	return sbom.Generate(dir)
}

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
			Generator{},
		),
	)
}
