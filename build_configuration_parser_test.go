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

		workingDir string

		targetManager      *fakes.TargetManager
		buildpackYMLParser *fakes.BuildpackYMLParser

		parser gobuild.BuildConfigurationParser
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		targetManager = &fakes.TargetManager{}
		targetManager.GenerateDefaultsCall.Returns.StringSlice = []string{"."}

		buildpackYMLParser = &fakes.BuildpackYMLParser{}
		buildpackYMLParser.ParseCall.Returns.BuildConfiguration = gobuild.BuildConfiguration{
			Targets: []string{"./first", "./second"},
			Flags: []string{
				"-first=value",
			},
			ImportPath: "some-import-path",
		}

		parser = gobuild.NewBuildConfigurationParser(targetManager, buildpackYMLParser)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when BP_GO_TARGETS is set", func() {
		it.Before(func() {
			os.Setenv("BP_GO_TARGETS", "some/target1:./some/target2")
			targetManager.CleanAndValidateCall.Returns.StringSlice = []string{"./some/target1", "./some/target2"}
		})

		it.After(func() {
			os.Unsetenv("BP_GO_TARGETS")
		})

		it("uses the values in the env var", func() {
			configuration, err := parser.Parse("1.2.3", workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration).To(Equal(gobuild.BuildConfiguration{
				Targets: []string{"./some/target1", "./some/target2"},
			}))

			Expect(targetManager.CleanAndValidateCall.Receives.Targets).To(Equal([]string{"some/target1", "./some/target2"}))
			Expect(targetManager.CleanAndValidateCall.Receives.WorkingDir).To(Equal(workingDir))
		})
	})

	context("when BP_GO_BUILD_FLAGS is set", func() {
		it.Before(func() {
			os.Setenv("BP_GO_BUILD_FLAGS", `-buildmode=default -tags=paketo -ldflags="-X main.variable=some-value" -first=$FIRST -second=${SECOND}`)
			os.Setenv("FIRST", "first-flag")
			os.Setenv("SECOND", "second-flag")
		})

		it.After(func() {
			os.Unsetenv("BP_GO_BUILD_FLAGS")
			os.Unsetenv("FIRST")
			os.Unsetenv("SECOND")
		})

		it("uses the values in the env var", func() {
			configuration, err := parser.Parse("1.2.3", workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration).To(Equal(gobuild.BuildConfiguration{
				Targets: []string{"."},
				Flags: []string{
					"-buildmode=default",
					"-tags=paketo",
					`-ldflags=-X main.variable=some-value`,
					"-first=first-flag",
					"-second=second-flag",
				},
			}))

			Expect(targetManager.GenerateDefaultsCall.Receives.WorkingDir).To(Equal(workingDir))
		})
	})

	context("when BP_GO_BUILD_LDFLAGS is set", func() {
		it.Before(func() {
			os.Setenv("BP_GO_BUILD_LDFLAGS", `-X main.variable=some-value -envFlag=$ENVVAR`)
			os.Setenv("ENVVAR", "env-value")
		})

		it.After(func() {
			os.Unsetenv("BP_GO_BUILD_LDFLAGS")
			os.Unsetenv("ENVVAR")
		})

		it("uses the values in the env var", func() {
			configuration, err := parser.Parse("1.2.3", workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration).To(Equal(gobuild.BuildConfiguration{
				Targets: []string{"."},
				Flags: []string{
					`-ldflags=-X main.variable=some-value -envFlag=env-value`,
				},
			}))

			Expect(targetManager.GenerateDefaultsCall.Receives.WorkingDir).To(Equal(workingDir))
		})

		context("and BP_GO_BUILD_FLAGS is set", func() {
			it.Before(func() {
				os.Setenv("BP_GO_BUILD_FLAGS", `-buildmode=default -tags=paketo -first=$FIRST -second=${SECOND}`)
				os.Setenv("FIRST", "first-flag")
				os.Setenv("SECOND", "second-flag")
			})

			it.After(func() {
				os.Unsetenv("BP_GO_BUILD_FLAGS")
				os.Unsetenv("FIRST")
				os.Unsetenv("SECOND")
			})

			it("adds the -ldflags to the rest of the build flags", func() {

				configuration, err := parser.Parse("1.2.3", workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(configuration).To(Equal(gobuild.BuildConfiguration{
					Targets: []string{"."},
					Flags: []string{
						"-buildmode=default",
						"-tags=paketo",
						"-first=first-flag",
						"-second=second-flag",
						`-ldflags=-X main.variable=some-value -envFlag=env-value`,
					},
				}))

				Expect(targetManager.GenerateDefaultsCall.Receives.WorkingDir).To(Equal(workingDir))
			})
		})

		context("and BP_GO_BUILD_FLAGS includes -ldflags", func() {
			it.Before(func() {
				os.Setenv("BP_GO_BUILD_FLAGS", `-buildmode=default -tags=paketo -ldflags="-X buildflags.variable=some-buildflags-value"`)
			})

			it.After(func() {
				os.Unsetenv("BP_GO_BUILD_FLAGS")
			})

			it("uses the value for -ldflags that comes from BP_GO_BUILD_LDFLAGS and removes the value set in BP_GO_BUILD_FLAGS", func() {
				configuration, err := parser.Parse("1.2.3", workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(configuration).To(Equal(gobuild.BuildConfiguration{
					Targets: []string{"."},
					Flags: []string{
						"-buildmode=default",
						"-tags=paketo",
						`-ldflags=-X main.variable=some-value -envFlag=env-value`,
					},
				}))

				Expect(targetManager.GenerateDefaultsCall.Receives.WorkingDir).To(Equal(workingDir))
			})
		})
	})

	context("when BP_GO_BUILD_IMPORT_PATH is set", func() {
		it.Before(func() {
			os.Setenv("BP_GO_BUILD_IMPORT_PATH", "./some/import/path")
		})

		it.After(func() {
			os.Unsetenv("BP_GO_BUILD_IMPORT_PATH")
		})

		it("uses the values in the env var", func() {
			configuration, err := parser.Parse("1.2.3", workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration).To(Equal(gobuild.BuildConfiguration{
				Targets:    []string{"."},
				ImportPath: "./some/import/path",
			}))

			Expect(targetManager.GenerateDefaultsCall.Receives.WorkingDir).To(Equal(workingDir))
		})
	})

	context("when there is a buildpack.yml and environment variables are not set", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), nil, 0644)
			Expect(err).NotTo(HaveOccurred())

			targetManager.CleanAndValidateCall.Returns.StringSlice = []string{"./first", "./second"}
		})

		it("parses the targets and flags from a buildpack.yml", func() {
			configuration, err := parser.Parse("1.2.3", workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration).To(Equal(gobuild.BuildConfiguration{
				Targets: []string{"./first", "./second"},
				Flags: []string{
					"-first=value",
				},
				ImportPath: "some-import-path",
			}))

			Expect(buildpackYMLParser.ParseCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(targetManager.CleanAndValidateCall.Receives.Targets).To(Equal([]string{"./first", "./second"}))
			Expect(targetManager.CleanAndValidateCall.Receives.WorkingDir).To(Equal(workingDir))
		})
	})

	context("when there is a buildpack.yml and environment variables are set", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), nil, 0644)
			Expect(err).NotTo(HaveOccurred())

			os.Setenv("BP_GO_BUILD_IMPORT_PATH", "./some/import/path")
			os.Setenv("BP_GO_TARGETS", "some/target1:./some/target2")
			os.Setenv("BP_GO_BUILD_FLAGS", `-some-flag=some-value`)

			targetManager.CleanAndValidateCall.Returns.StringSlice = []string{"./some/target1", "./some/target2"}
		})

		it.After(func() {
			os.Unsetenv("BP_GO_BUILD_IMPORT_PATH")
			os.Unsetenv("BP_GO_TARGETS")
			os.Unsetenv("BP_GO_BUILD_FLAGS")
		})

		it("parses the targets and flags from a buildpack.yml but uses the values from the environment variables", func() {
			configuration, err := parser.Parse("1.2.3", workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration).To(Equal(gobuild.BuildConfiguration{
				Targets: []string{"./some/target1", "./some/target2"},
				Flags: []string{
					"-some-flag=some-value",
				},
				ImportPath: "./some/import/path",
			}))

			Expect(buildpackYMLParser.ParseCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(targetManager.CleanAndValidateCall.Receives.Targets).To(Equal([]string{"some/target1", "./some/target2"}))
			Expect(targetManager.CleanAndValidateCall.Receives.WorkingDir).To(Equal(workingDir))
		})
	})

	context("buildpack.yml specifies flags including -ldflags and BP_GO_BUILD_LDFLAGS is set", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), nil, 0644)
			Expect(err).NotTo(HaveOccurred())

			buildpackYMLParser.ParseCall.Returns.BuildConfiguration = gobuild.BuildConfiguration{
				Targets: []string{"."},
				Flags: []string{
					`-ldflags="-buildpack -yml -flags"`,
					`-otherflag`,
				},
			}

			os.Setenv("BP_GO_BUILD_LDFLAGS", `-env -value`)
		})

		it.After(func() {
			os.Unsetenv("BP_GO_BUILD_LDFLAGS")
		})

		it("uses build flags from the buildpack.yml EXCEPT -ldflags", func() {
			configuration, err := parser.Parse("1.2.3", workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(configuration).To(Equal(gobuild.BuildConfiguration{
				Flags: []string{
					`-ldflags=-env -value`,
					`-otherflag`,
				},
			}))

			Expect(buildpackYMLParser.ParseCall.Receives.WorkingDir).To(Equal(workingDir))
		})
	})

	context("failure cases", func() {
		context("when the buildpack.yml cannot be stat'd", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := parser.Parse("1.2.3", workingDir)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("buildpack.yml parsing fails", func() {
			it.Before(func() {
				err := ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), nil, 0644)
				Expect(err).NotTo(HaveOccurred())

				buildpackYMLParser.ParseCall.Returns.Error = errors.New("failed to parse buildpack.yml")
			})

			it("returns an error", func() {
				_, err := parser.Parse("1.2.3", workingDir)
				Expect(err).To(MatchError("failed to parse buildpack.yml"))
			})
		})

		context("go targets fail to be cleaned an validated", func() {
			it.Before(func() {
				os.Setenv("BP_GO_TARGETS", "./some/target")

				targetManager.CleanAndValidateCall.Returns.Error = errors.New("failed to clean and validate targets")

			})

			it.After(func() {
				os.Unsetenv("BP_GO_TARGETS")
			})

			it("returns an error", func() {
				_, err := parser.Parse("1.2.3", workingDir)
				Expect(err).To(MatchError("failed to clean and validate targets"))
			})
		})

		context("when no targets can be found", func() {
			it.Before(func() {
				targetManager.GenerateDefaultsCall.Returns.Error = errors.New("failed to default target found")
			})

			it("returns an error", func() {
				_, err := parser.Parse("1.2.3", workingDir)
				Expect(err).To(MatchError("failed to default target found"))
			})
		})

		context("when the build flags fail to parse", func() {
			it.Before(func() {
				os.Setenv("BP_GO_BUILD_FLAGS", "\"")
			})

			it.After(func() {
				os.Unsetenv("BP_GO_BUILD_FLAGS")
			})

			it("returns an error", func() {
				_, err := parser.Parse("1.2.3", workingDir)
				Expect(err).To(MatchError(ContainSubstring("invalid command line string")))
			})
		})
		context("when the ldflags fail to parse", func() {
			it.Before(func() {
				os.Setenv("BP_GO_BUILD_LDFLAGS", "\"")
			})

			it.After(func() {
				os.Unsetenv("BP_GO_BUILD_LDFLAGS")
			})

			it("returns an error", func() {
				_, err := parser.Parse("1.2.3", workingDir)
				Expect(err).To(MatchError(ContainSubstring("invalid command line string")))
			})
		})

		context("when the ldflags cannot be parsed as a single -ldflags value", func() {
			it.Before(func() {
				os.Setenv("BP_GO_BUILD_LDFLAGS", `"spaces in quotes"`)
			})

			it.After(func() {
				os.Unsetenv("BP_GO_BUILD_LDFLAGS")
			})

			it("returns an error", func() {
				_, err := parser.Parse("1.2.3", workingDir)
				Expect(err).To(MatchError(ContainSubstring(`BP_GO_BUILD_LDFLAGS value ("spaces in quotes") could not be parsed: value contains multiple words`)))
			})
		})
	})
}
