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

func testDefault(t *testing.T, context spec.G, it spec.S) {
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

	context("when building a simple app with no dependencies", func() {
		var (
			image     occam.Image
			container occam.Container

			name    string
			source  string
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
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.GoDist.Online,
					settings.Buildpacks.GoBuild.Online,
				).
				WithEnv(map[string]string{
					"BP_LOG_LEVEL": "DEBUG",
				}).
				WithSBOMOutputDir(sbomDir).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(
				Serve(
					SatisfyAll(
						ContainSubstring("go1.20"),
						ContainSubstring("/workspace contents: []"),
					),
				).OnPort(8080),
			)

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				"  Executing build process",
				MatchRegexp(fmt.Sprintf(`Running 'go build -o /layers/%s/targets/bin -buildmode ([^\s]+) -trimpath .'`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
			))
			Expect(logs).To(ContainLines(
				fmt.Sprintf("  Generating SBOM for /layers/%s/targets/bin", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
			))
			Expect(logs).To(ContainLines(
				"  Writing SBOM in the following format(s):",
				"    application/vnd.cyclonedx+json",
				"    application/spdx+json",
				"    application/vnd.syft+json",
			))
			Expect(logs).To(ContainLines(
				"  Assigning launch processes:",
				fmt.Sprintf("    workspace (default): /layers/%s/targets/bin/workspace", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))

			// check that all required SBOM files are present
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.cdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.spdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.syft.json")).To(BeARegularFile())

			// check an SBOM file to make sure it is generated for the right directory
			contents, err := os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "targets", "sbom.syft.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainLines(
				`  "type": "directory",`,
				fmt.Sprintf(`  "target": "/layers/%s/targets/bin"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
		})
	})

	context("when building a simple app with no dependencies in an offline environment", func() {
		var (
			image     occam.Image
			container occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds successfully", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithNetwork("none").
				WithBuildpacks(
					settings.Buildpacks.GoDist.Offline,
					settings.Buildpacks.GoBuild.Offline,
				).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(Serve(ContainSubstring("go1.20")).OnPort(8080))
		})
	})

	context("when building a simple app with no dependencies that is rooted in cmd", func() {
		var (
			image     occam.Image
			container occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds successfully", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "cmd_root"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.GoDist.Online,
					settings.Buildpacks.GoBuild.Online,
				).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(
				Serve(
					SatisfyAll(
						ContainSubstring("go1.20"),
						ContainSubstring("/workspace contents: []"),
					),
				).OnPort(8080),
			)
		})
	})

	context("when building a simple app with BP_LIVE_RELOAD_ENABLED", func() {
		var (
			image             occam.Image
			container         occam.Container
			noReloadContainer occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Container.Remove.Execute(noReloadContainer.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds successfully and makes reloadable and non-reloadable process types available", func() {
			var (
				err  error
				logs fmt.Stringer
			)
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.Watchexec.Online,
					settings.Buildpacks.GoDist.Online,
					settings.Buildpacks.GoBuild.Online,
				).
				WithEnv(map[string]string{
					"BP_LIVE_RELOAD_ENABLED": "true",
				}).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(Serve(ContainSubstring("go1.20")).OnPort(8080))

			Expect(logs).To(ContainLines(
				"  Assigning launch processes:",
				fmt.Sprintf("    workspace:                  /layers/%s/targets/bin/workspace", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				fmt.Sprintf("    reload-workspace (default): watchexec --restart --watch /workspace --watch /layers/%[1]s/targets/bin --shell none -- /layers/%[1]s/targets/bin/workspace", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))

			noReloadContainer, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				WithEntrypoint("workspace").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(noReloadContainer).Should(Serve(ContainSubstring("go1.20")).OnPort(8080))
		})
	})
}
