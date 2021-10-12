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

func testBuildFailure(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose().WithNoColor()
		docker = occam.NewDocker()
	})

	context("when building an app with compilation errors", func() {
		var (
			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("shows those errors in the output", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			file, err := os.OpenFile(filepath.Join(source, "main.go"), os.O_RDWR|os.O_APPEND, 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = file.WriteString("func SomeFunc(i int, t SomeType) error { return nil }")
			Expect(err).NotTo(HaveOccurred())

			Expect(file.Close()).To(Succeed())

			_, logs, err := pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.GoDist.Online,
					settings.Buildpacks.GoBuild.Online,
				).
				Execute(name, source)
			Expect(err).To(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				"  Executing build process",
				fmt.Sprintf("    Running 'go build -o /layers/%s/targets/bin -buildmode pie -trimpath .'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
			Expect(logs).To(ContainLines(
				MatchRegexp(`      Failed after ([0-9]*(\.[0-9]*)?[a-z]+)+`),
			))
			Expect(logs).To(ContainLines(
				MatchRegexp(`undefined: SomeType`),
			))
		})
	})
}
