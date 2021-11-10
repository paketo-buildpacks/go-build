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

func testBuildpackYML(t *testing.T, context spec.G, it spec.S) {
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
			image     occam.Image
			container occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = occam.Source(filepath.Join("testdata", "targets"))
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(filepath.Join(source, "buildpack.yml"), []byte(`---
go:
  targets:
  - first
  - ./second`), 0600)
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

			Eventually(container).Should(Serve(ContainSubstring("first: go1.16")).OnPort(8080))

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				"  WARNING: Setting the Go Build configurations such as targets, build flags, and import path through buildpack.yml will be deprecated soon in Go Build Buildpack v2.0.0.",
				"  Please specify these configuration options through environment variables instead. See README.md or the documentation on paketo.io for more information.",
				"",
				"  Executing build process",
				fmt.Sprintf("    Running 'go build -o /layers/%s/targets/bin -buildmode pie -trimpath ./first ./second'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				"",
				"  Assigning launch processes:",
				fmt.Sprintf("    first: /layers/%s/targets/bin/first", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
		})

		context("when building an app with target specified via BP_GO_TARGETS env", func() {
			it("builds succesfully and overrides buildpack.yml while still printing a warning", func() {
				var err error
				var logs fmt.Stringer
				image, logs, err = pack.Build.
					WithPullPolicy("never").
					WithEnv(map[string]string{"BP_GO_TARGETS": "./third"}).
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

				Eventually(container).Should(Serve(ContainSubstring("third: go1.16")).OnPort(8080))

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
					"  WARNING: Setting the Go Build configurations such as targets, build flags, and import path through buildpack.yml will be deprecated soon in Go Build Buildpack v2.0.0.",
					"  Please specify these configuration options through environment variables instead. See README.md or the documentation on paketo.io for more information.",
					"",
					"  Executing build process",
					fmt.Sprintf("    Running 'go build -o /layers/%s/targets/bin -buildmode pie -trimpath ./third'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
					MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
					"",
					"  Assigning launch processes:",
					fmt.Sprintf("    third: /layers/%s/targets/bin/third", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				))
			})
		})
	})
}
