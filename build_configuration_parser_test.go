package gobuild_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/paketo-buildpacks/go-build/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuildConfigurationParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir    string
		targetManager *fakes.TargetManager

		parser gobuild.BuildConfigurationParser
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		targetManager = &fakes.TargetManager{}

		parser = gobuild.NewBuildConfigurationParser(targetManager)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when there is a buildpack.yml", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
go:
  targets:
  - first
  - ./second
  build:
    flags:
    - -first
    - value
    - -second=value
    - -third="value"
    - -fourth='value'
    import-path: some-import-path
`), 0644)
			Expect(err).NotTo(HaveOccurred())

			targetManager.CleanAndValidateCall.Returns.StringSlice = []string{"./first", "./second"}
		})

		it("parses the targets and flags from a buildpack.yml", func() {
			configuration, err := parser.Parse(workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration).To(Equal(gobuild.BuildConfiguration{
				Targets: []string{"./first", "./second"},
				Flags: []string{
					"-first", "value",
					"-second", "value",
					"-third", "value",
					"-fourth", "value",
				},
				ImportPath: "some-import-path",
			}))

			Expect(targetManager.CleanAndValidateCall.Receives.Targets).To(Equal([]string{"first", "./second"}))
			Expect(targetManager.CleanAndValidateCall.Receives.WorkingDir).To(Equal(workingDir))
		})
	})

	context("when there is no buildpack.yml file", func() {
		it.Before(func() {
			targetManager.GenerateDefaultsCall.Returns.StringSlice = []string{workingDir}
		})

		it("returns a list of default targets and empty list of flags", func() {
			configuration, err := parser.Parse(workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration.Targets).To(Equal([]string{workingDir}))
			Expect(configuration.Flags).To(BeEmpty())

			Expect(targetManager.GenerateDefaultsCall.Receives.WorkingDir).To(Equal(workingDir))
		})

		context("BP_GO_TARGETS env variable is set", func() {
			it.Before(func() {
				os.Setenv("BP_GO_TARGETS", "some/target1:./some/target2")
				targetManager.CleanAndValidateCall.Returns.StringSlice = []string{"./some/target1", "./some/target2"}
			})

			it.After(func() {
				os.Unsetenv("BP_GO_TARGETS")
			})

			it("uses the values in the env var", func() {
				configuration, err := parser.Parse(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(configuration.Targets).To(Equal([]string{"./some/target1", "./some/target2"}))
				Expect(configuration.Flags).To(BeEmpty())

				Expect(targetManager.CleanAndValidateCall.Receives.Targets).To(Equal([]string{"some/target1", "./some/target2"}))
				Expect(targetManager.CleanAndValidateCall.Receives.WorkingDir).To(Equal(workingDir))
			})
		})
	})

	context("when the targets list is empty", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
go:
  targets: []
`), 0644)).To(Succeed())
			targetManager.GenerateDefaultsCall.Returns.StringSlice = []string{"./cmd/first"}
		})

		it("returns a list of default targets", func() {
			configuration, err := parser.Parse(workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration.Targets).To(Equal([]string{"./cmd/first"}))

			Expect(targetManager.GenerateDefaultsCall.Receives.WorkingDir).To(Equal(workingDir))
		})
	})

	context("when the build flags reference an env var", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
go:
  build:
    flags:
    - -first=${SOME_VALUE}
    - -second=$SOME_OTHER_VALUE
`), 0644)

			Expect(err).NotTo(HaveOccurred())

			os.Setenv("SOME_VALUE", "some-value")
			os.Setenv("SOME_OTHER_VALUE", "some-other-value")
		})

		it.After(func() {
			os.Unsetenv("SOME_VALUE")
			os.Unsetenv("SOME_OTHER_VALUE")
		})

		it("replaces the targets list with the values in the env var", func() {
			configuration, err := parser.Parse(workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration.Flags).To(Equal([]string{
				"-first", "some-value",
				"-second", "some-other-value",
			}))
		})
	})

	context("failure cases", func() {
		context("when defaults cannot be generated when there is no buildpack.yml and no targets set", func() {
			it.Before(func() {
				targetManager.GenerateDefaultsCall.Returns.Error = errors.New("no defaults could be found")
			})

			it("returns an error", func() {
				_, err := parser.Parse(workingDir)
				Expect(err).To(MatchError("no defaults could be found"))
			})
		})

		context("when targets cannot be cleaned and validated when there is no buildpack.yml but there are targets set", func() {
			it.Before(func() {
				os.Setenv("BP_GO_TARGETS", "some/target1")
				targetManager.CleanAndValidateCall.Returns.Error = errors.New("unable to validate and clean targets")
			})

			it.After(func() {
				os.Unsetenv("BP_GO_TARGETS")
			})

			it("returns an error", func() {
				_, err := parser.Parse(workingDir)
				Expect(err).To(MatchError("unable to validate and clean targets"))
			})
		})

		context("when the buildpack.yml file cannot be read", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), nil, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := parser.Parse(workingDir)
				Expect(err).To(MatchError(ContainSubstring("failed to read buildpack.yml:")))
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the buildpack.yml file cannot be parsed", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte("%%%"), 0644)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := parser.Parse(workingDir)
				Expect(err).To(MatchError(ContainSubstring("failed to decode buildpack.yml:")))
				Expect(err).To(MatchError(ContainSubstring("could not find expected directive name")))
			})
		})

		context("when targets cannot be cleaned and validated when there is buildpack.yml and the are targets set by env var", func() {
			it.Before(func() {
				os.Setenv("BP_GO_TARGETS", "some/target1")
				targetManager.CleanAndValidateCall.Returns.Error = errors.New("unable to validate and clean targets from env var")

				Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---`), 0644)).To(Succeed())
			})

			it.After(func() {
				os.Unsetenv("BP_GO_TARGETS")
			})

			it("returns an error", func() {
				_, err := parser.Parse(workingDir)
				Expect(err).To(MatchError("unable to validate and clean targets from env var"))
			})
		})

		context("when targets cannot be cleaned and validated when there is buildpack.yml and the are targets set by buildpack.yml", func() {
			it.Before(func() {
				targetManager.CleanAndValidateCall.Returns.Error = errors.New("unable to validate and clean targets from buildpack.yml")

				Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
go:
  targets:
  - ./first
`), 0644)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := parser.Parse(workingDir)
				Expect(err).To(MatchError("unable to validate and clean targets from buildpack.yml"))
			})
		})

		context("when defaults cannot be cleaned generated when there is buildpack.yml", func() {
			it.Before(func() {
				targetManager.GenerateDefaultsCall.Returns.Error = errors.New("no defaults could be found")

				Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---`), 0644)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := parser.Parse(workingDir)
				Expect(err).To(MatchError("no defaults could be found"))
			})
		})

		context("when a the env var expansion fails", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
go:
  build:
    flags:
    - -first=$&
`), 0644)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := parser.Parse(workingDir)
				Expect(err).To(MatchError(ContainSubstring("environment variable expansion failed:")))
			})
		})
	})
}
