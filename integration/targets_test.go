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

func testTargets(t *testing.T, context spec.G, it spec.S) {
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

	context("when building an app with multiple targets", func() {
		var (
			image        occam.Image
			container    occam.Container
			containerIDs map[string]struct{}
			sbomDir      string

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			containerIDs = map[string]struct{}{}

			source, err = occam.Source(filepath.Join("testdata", "targets"))
			Expect(err).NotTo(HaveOccurred())

			sbomDir, err = os.MkdirTemp("", "sbom")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chmod(sbomDir, os.ModePerm)).To(Succeed())
		})

		it.After(func() {
			for id := range containerIDs {
				Expect(docker.Container.Remove.Execute(id)).To(Succeed())
			}
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
			Expect(os.RemoveAll(sbomDir)).To(Succeed())
		})

		it("builds successfully and includes SBOM with modules for built binaries", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithEnv(map[string]string{"BP_GO_TARGETS": "first:./second"}).
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
			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(Serve(ContainSubstring("first: go")).OnPort(8080))

			Expect(logs).To(ContainLines(
				"  Assigning launch processes:",
				fmt.Sprintf("    first (default): /layers/%s/targets/bin/first", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				fmt.Sprintf("    second:          /layers/%s/targets/bin/second", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))

			// The second launch process can be accessed using its name entrypoint
			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				WithEntrypoint("second").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())
			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(Serve(ContainSubstring("second: go")).OnPort(8080))

			// check that all required SBOM files are present
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.cdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.spdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.syft.json")).To(BeARegularFile())

			// check an SBOM file to make sure it contains entries for built binaries
			contents, err := os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.cdx.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(`"name": "github.com/gorilla/mux"`))
			Expect(string(contents)).To(ContainSubstring(`"name": "github.com/sahilm/fuzzy"`))
			// and does not contain an entry for the binary that was not compiled
			Expect(string(contents)).NotTo(ContainSubstring(`"name": "github.com/Masterminds/semver"`))
		})
	})
}
