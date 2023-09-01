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

func testBuildFlags(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		image     occam.Image
		container occam.Container

		name   string
		source string
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose().WithNoColor()
		docker = occam.NewDocker()

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

	context("when building a simple app with build flags", func() {
		it("builds successfully", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "build_flags"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithEnv(map[string]string{
					"BP_GO_BUILD_FLAGS":   `-buildmode=default -tags=paketo`,
					"BP_GO_BUILD_LDFLAGS": `-X main.variable=some-value`,
				}).
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
						ContainSubstring(`variable value: "some-value"`),
						ContainSubstring("/workspace contents: []"),
					),
				).OnPort(8080),
			)

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				"  Executing build process",
				fmt.Sprintf("    Running 'go build -o /layers/%s/targets/bin -buildmode=default -tags=paketo \"-ldflags=-X main.variable=some-value\" -trimpath .'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				"",
			))
		})
	})

	context("when building a simple app with build flags with env var interpolation", func() {
		it("builds successfully", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "build_flags"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithPullPolicy("never").
				WithEnv(map[string]string{
					"BP_GO_BUILD_FLAGS":   `-buildmode=default -tags=paketo`,
					"BP_GO_BUILD_LDFLAGS": `-X main.variable=${SOME_VALUE}`,
					"SOME_VALUE":          "env-value",
				}).
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
						ContainSubstring(`variable value: "env-value"`),
						ContainSubstring("/workspace contents: []"),
					),
				).OnPort(8080),
			)

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				"  Executing build process",
				fmt.Sprintf("    Running 'go build -o /layers/%s/targets/bin -buildmode=default -tags=paketo \"-ldflags=-X main.variable=env-value\" -trimpath .'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				"",
			))
		})
	})
}
