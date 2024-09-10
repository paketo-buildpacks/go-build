package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testMod(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose().WithNoColor()
		docker = occam.NewDocker()
	})

	context("when building an app that uses modules", func() {
		var (
			image     occam.Image
			container occam.Container

			name   string
			source string

			sbomDir string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			sbomDir, err = os.MkdirTemp("", "sbom")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chmod(sbomDir, os.ModePerm)).To(Succeed())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
			Expect(os.RemoveAll(sbomDir)).To(Succeed())
		})

		it("builds successfully", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "mod"))
			Expect(err).NotTo(HaveOccurred())

			Expect(os.RemoveAll(filepath.Join(source, "vendor"))).To(Succeed())

			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.GoDist.Online,
					settings.Buildpacks.GoBuild.Online,
				).
				WithSBOMOutputDir(sbomDir).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(Serve(ContainSubstring("/workspace contents: []")).OnPort(8080))

			// check that all required SBOM files are present
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.cdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.spdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.syft.json")).To(BeARegularFile())

			// check an SBOM file to make sure it contains the expected dependency
			contents, err := os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.cdx.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(`"name": "github.com/gorilla/mux"`))
		})

		context("when there is a go.mod AND modules are vendored", func() {
			it("populates an SBOM with vendored packages", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "mod"))
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.GoDist.Online,
						settings.Buildpacks.GoBuild.Online,
					).
					WithSBOMOutputDir(sbomDir).
					Execute(name, source)
				Expect(err).ToNot(HaveOccurred(), logs.String)

				container, err = docker.Container.Run.
					WithEnv(map[string]string{"PORT": "8080"}).
					WithPublish("8080").
					WithPublishAll().
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(Serve(ContainSubstring("/workspace contents: []")).OnPort(8080))

				// check that all required SBOM files are present
				Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.cdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.spdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.syft.json")).To(BeARegularFile())

				// check an SBOM file to make sure it contains the expected dependency
				contents, err := os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.cdx.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(`"name": "github.com/gorilla/mux"`))
			})
		})
	})
}
