package gobuild_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testGoBuildpackYMLParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		logs       *bytes.Buffer

		goBuildpackYMLParser gobuild.GoBuildpackYMLParser
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
go:
  targets:
  - first
  - ./second
  build:
    flags:
    - -first
    - value
    - -second=value
    import-path: some-import-path
`), 0644)).To(Succeed())

		logs = bytes.NewBuffer(nil)
		goBuildpackYMLParser = gobuild.NewGoBuildpackYMLParser(gobuild.NewLogEmitter(logs))
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Parse", func() {
		it("parses the buildpack and returns a build configuration", func() {
			config, err := goBuildpackYMLParser.Parse("1.2.3", workingDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(config).To(Equal(gobuild.BuildConfiguration{
				Targets: []string{"first", "./second"},
				Flags: []string{
					"-first",
					"value",
					"-second",
					"value",
				},
				ImportPath: "some-import-path",
			}))
			Expect(logs.String()).To(ContainSubstring("WARNING: Setting the Go Build configurations such as targets, build flags, and import path through buildpack.yml will be deprecated soon in Go Build Buildpack v2.0.0."))
			Expect(logs.String()).To(ContainSubstring("Please specify these configuration options through environment variables instead. See README.md or the documentation on paketo.io for more information."))
		})

		context("when the flags have an env var in them", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
go:
  build:
    flags:
    - -first=$FIRST
    - -second=${SECOND}
`), 0644)).To(Succeed())

				os.Setenv("FIRST", "first-val")
				os.Setenv("SECOND", "second-val")
			})

			it.After(func() {
				os.Unsetenv("FIRST")
				os.Unsetenv("SECOND")
			})

			it("interpolates the env vars those into the flags", func() {
				config, err := goBuildpackYMLParser.Parse("1.2.3", workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(config).To(Equal(gobuild.BuildConfiguration{
					Flags: []string{
						"-first",
						"first-val",
						"-second",
						"second-val",
					},
				}))
			})
		}, spec.Sequential())

		context("when the buildpack.yml does not contain go configuration", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
not-go:
  build:
    flags:
    - -first=value
`), 0644)).To(Succeed())
			})

			it("does not return a deprecation message", func() {
				config, err := goBuildpackYMLParser.Parse("1.2.3", workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(config).To(Equal(gobuild.BuildConfiguration{}))
				Expect(logs.String()).To(BeEmpty())
			})
		})

		context("failure cases", func() {
			context("buildpack.yml cannot be opened", func() {
				it.Before(func() {
					Expect(os.Chmod(filepath.Join(workingDir, "buildpack.yml"), 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := goBuildpackYMLParser.Parse("1.2.3", workingDir)
					Expect(err).To(MatchError(ContainSubstring("failed to read buildpack.yml")))
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("buildpack.yml fails to parse", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`%%%`), 0644)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := goBuildpackYMLParser.Parse("1.2.3", workingDir)
					Expect(err).To(MatchError(ContainSubstring("failed to decode buildpack.yml")))
					Expect(err).To(MatchError(ContainSubstring("could not find expected directive name")))
				})
			})

			context("when a the env var interpolation fails", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
go:
  build:
    flags:
    - -first=$&
`), 0644)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := goBuildpackYMLParser.Parse("1.2.3", workingDir)
					Expect(err).To(MatchError(ContainSubstring("environment variable expansion failed:")))
				})
			})
		})
	})
}
