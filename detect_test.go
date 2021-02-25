package gobuild_test

import (
	"errors"
	"testing"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/paketo-buildpacks/go-build/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		parser     *fakes.ConfigurationParser

		detect packit.DetectFunc
	)

	it.Before(func() {
		workingDir = "working-dir"

		parser = &fakes.ConfigurationParser{}
		parser.ParseCall.Returns.BuildConfiguration.Targets = []string{workingDir}

		detect = gobuild.Detect(parser)
	})

	it("detects", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: workingDir,
			BuildpackInfo: packit.BuildpackInfo{
				Version: "some-buildpack-version",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Plan).To(Equal(packit.BuildPlan{
			Requires: []packit.BuildPlanRequirement{{
				Name: "go",
				Metadata: map[string]interface{}{
					"build": true,
				},
			}},
		}))

		Expect(parser.ParseCall.Receives.BuildpackVersion).To(Equal("some-buildpack-version"))
		Expect(parser.ParseCall.Receives.WorkingDir).To(Equal(workingDir))
	})

	context("when there are no *.go files in the working directory", func() {
		it.Before(func() {
			parser.ParseCall.Returns.Error = errors.New("no *.go files found")
		})

		it("fails detection", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(ContainSubstring("failed to parse build configuration: no *.go files found")))
		})
	})

	context("failure cases", func() {
		context("when the configuration parser fails", func() {
			it.Before(func() {
				parser.ParseCall.Returns.Error = errors.New("failed to parse configuration")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse configuration")))
			})
		})
	})
}
