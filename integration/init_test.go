package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/onsi/gomega/format"
	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var builder occam.Builder

var settings struct {
	Buildpacks struct {
		GoDist struct {
			Online  string
			Offline string
		}
		GoBuild struct {
			Online  string
			Offline string
		}
		Watchexec struct {
			Online  string
			Offline string
		}
	}
	Buildpack struct {
		ID   string
		Name string
	}
	Config struct {
		GoDist    string `json:"go-dist"`
		Watchexec string `json:"watchexec"`
	}
}

func TestIntegration(t *testing.T) {
	format.MaxLength = 0

	Expect := NewWithT(t).Expect
	pack := occam.NewPack()

	file, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())

	Expect(json.NewDecoder(file).Decode(&settings.Config)).To(Succeed())
	Expect(file.Close()).To(Succeed())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.NewDecoder(file).Decode(&settings)
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())

	root, err := filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	// libpakBuildpackStore := occam.NewBuildpackStore().WithPackager(packagers.NewLibpak())

	settings.Buildpacks.GoBuild.Online, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.GoBuild.Offline, err = buildpackStore.Get.
		WithVersion("1.2.3").
		WithOfflineDependencies().
		Execute(root)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.GoDist.Online, err = buildpackStore.Get.
		Execute(settings.Config.GoDist)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.GoDist.Offline, err = buildpackStore.Get.
		WithOfflineDependencies().
		Execute(settings.Config.GoDist)
	Expect(err).ToNot(HaveOccurred())

	// Currently this path way is broken while things are worked out in upstream
	// pack to allow for multi-arch buildpacks
	// settings.Buildpacks.Watchexec.Online, err = libpakBuildpackStore.Get.
	// 	Execute(settings.Config.Watchexec)
	// Expect(err).ToNot(HaveOccurred())

	// settings.Buildpacks.Watchexec.Offline, err = libpakBuildpackStore.Get.
	// 	WithOfflineDependencies().
	// 	Execute(settings.Config.Watchexec)
	// Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.Watchexec.Online = "paketo-buildpacks/watchexec"

	builder, err = pack.Builder.Inspect.Execute()
	Expect(err).NotTo(HaveOccurred())

	SetDefaultEventuallyTimeout(10 * time.Second)

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("BuildFailure", testBuildFailure)
	suite("Default", testDefault)
	suite("ImportPath", testImportPath)
	suite("KeepFiles", testKeepFiles)
	suite("Mod", testMod)
	suite("Rebuild", testRebuild)
	suite("Targets", testTargets)
	suite("Vendor", testVendor)
	suite("WorkUse", testWorkUse)
	if builder.BuilderName != "paketobuildpacks/builder-jammy-buildpackless-static" {
		suite("BuildFlags", testBuildFlags)
	}
	suite.Run(t)
}
