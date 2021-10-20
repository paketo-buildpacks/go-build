package gobuild_test

import (
	"errors"
	"os"
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

	context("BP_LIVE_RELOAD_ENABLED=true in build environment", func() {
		it.Before(func() {
			os.Setenv("BP_LIVE_RELOAD_ENABLED", "true")
		})

		it.After(func() {
			os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
		})

		it("requires watchexec at launch time", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
				BuildpackInfo: packit.BuildpackInfo{
					Version: "some-buildpack-version",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan.Requires).To(ContainElement(packit.BuildPlanRequirement{
				Name: "watchexec",
				Metadata: map[string]interface{}{
					"launch": true,
				},
			}))
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
		context("parsing value of $BP_LIVE_RELOAD_ENABLED fails", func() {
			it.Before(func() {
				os.Setenv("BP_LIVE_RELOAD_ENABLED", "not-a-bool")
			})

			it.After(func() {
				os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
					BuildpackInfo: packit.BuildpackInfo{
						Version: "some-buildpack-version",
					},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse BP_LIVE_RELOAD_ENABLED value not-a-bool")))
			})
		})
		context("BP_LIVE_RELOAD_ENABLED=true and the stack is tiny", func() {
			it.Before(func() {
				os.Setenv("BP_LIVE_RELOAD_ENABLED", "true")
			})

			it.After(func() {
				os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
					BuildpackInfo: packit.BuildpackInfo{
						Version: "some-buildpack-version",
					},
					Stack: "io.paketo.stacks.tiny",
				})
				Expect(err).To(MatchError(ContainSubstring("cannot enable live reload on stack 'io.paketo.stacks.tiny': stack does not support watchexec")))
			})
		})
	})
}
