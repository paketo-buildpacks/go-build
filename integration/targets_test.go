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
		Expect       = NewWithT(t).Expect
		Eventually   = NewWithT(t).Eventually
		Consistently = NewWithT(t).Consistently

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
		})

		it.After(func() {
			for id := range containerIDs {
				Expect(docker.Container.Remove.Execute(id)).To(Succeed())
			}
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
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
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())
			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(Serve(ContainSubstring("first: go1.16")).OnPort(8080))

			Expect(logs).To(ContainLines(
				"  Assigning launch processes:",
				fmt.Sprintf("    first (default): /layers/%s/targets/bin/first", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				fmt.Sprintf("    second:          /layers/%s/targets/bin/second", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))

			// check that all expected SBOM files are present
			container, err = docker.Container.Run.
				WithCommand(fmt.Sprintf("ls -al /layers/sbom/launch/%s/targets/",
					strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))).
				WithEntrypoint("launcher").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())
			containerIDs[container.ID] = struct{}{}

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(And(
				ContainSubstring("sbom.cdx.json"),
				ContainSubstring("sbom.spdx.json"),
				ContainSubstring("sbom.syft.json"),
			))

			// check an SBOM file to make sure it has entries for built targets
			container, err = docker.Container.Run.
				WithCommand(fmt.Sprintf("cat /layers/sbom/launch/%s/targets/sbom.cdx.json",
					strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))).
				WithEntrypoint("launcher").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())
			containerIDs[container.ID] = struct{}{}

			// a package in `first` executable
			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(ContainSubstring(`"name": "github.com/gorilla/mux"`))

			// a package in `second` executable
			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(ContainSubstring(`"name": "github.com/sahilm/fuzzy"`))

			// The SBOM shouldn't contain entries for `third` since it was not built
			Consistently(func() string {
				cLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).ShouldNot(ContainSubstring(`"name": "github.com/Masterminds/semver"`))
		})

		it("the other binary can be accessed using its name as an entrypoint", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithEnv(map[string]string{"BP_GO_TARGETS": "first:./second"}).
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
				WithEntrypoint("second").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())
			containerIDs[container.ID] = struct{}{}

			Eventually(container).Should(Serve(ContainSubstring("second: go1.16")).OnPort(8080))
		})
	})
}
