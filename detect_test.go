package gobuild_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
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
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		Expect(ioutil.WriteFile(filepath.Join(workingDir, "main.go"), nil, 0644)).To(Succeed())

		parser = &fakes.ConfigurationParser{}
		parser.ParseCall.Returns.BuildConfiguration.Targets = []string{workingDir}

		detect = gobuild.Detect(parser)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("detects", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: workingDir,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Plan).To(Equal(packit.BuildPlan{
			Requires: []packit.BuildPlanRequirement{
				{
					Name: "go",
					Metadata: map[string]interface{}{
						"build": true,
					},
				},
			},
		}))

		Expect(parser.ParseCall.Receives.Path).To(Equal(filepath.Join(workingDir, "buildpack.yml")))
	})

	context("when there are multiple targets", func() {
		it.Before(func() {
			Expect(os.Remove(filepath.Join(workingDir, "main.go"))).To(Succeed())

			Expect(os.Mkdir(filepath.Join(workingDir, "first"), os.ModePerm)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(workingDir, "second"), os.ModePerm)).To(Succeed())

			Expect(ioutil.WriteFile(filepath.Join(workingDir, "first", "main.go"), nil, 0644)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(workingDir, "second", "main.go"), nil, 0644)).To(Succeed())

			parser.ParseCall.Returns.BuildConfiguration.Targets = []string{
				filepath.Join(workingDir, "first"),
				filepath.Join(workingDir, "second"),
			}
		})

		it("detects only if all targets have go source files", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "go",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
				},
			}))

			Expect(parser.ParseCall.Receives.Path).To(Equal(filepath.Join(workingDir, "buildpack.yml")))
		})
	})

	context("when there are no *.go files in the working directory", func() {
		it.Before(func() {
			Expect(os.Remove(filepath.Join(workingDir, "main.go"))).To(Succeed())
		})

		it("fails detection", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(packit.Fail))
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
				Expect(err).To(MatchError("failed to parse configuration"))
			})
		})

		context("when file glob fails", func() {
			it.Before(func() {
				parser.ParseCall.Returns.BuildConfiguration.Targets = []string{`\`}
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("syntax error in pattern")))
			})
		})
	})
}
