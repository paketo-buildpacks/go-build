package gobuild_test

import (
	"io/ioutil"
	"os"
	"testing"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuildConfigurationParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string

		parser gobuild.BuildConfigurationParser
	)

	it.Before(func() {
		file, err := ioutil.TempFile("", "buildpack.yml")
		Expect(err).NotTo(HaveOccurred())

		_, err = file.WriteString(`---
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
`)
		Expect(err).NotTo(HaveOccurred())

		Expect(file.Close()).To(Succeed())

		path = file.Name()

		parser = gobuild.NewBuildConfigurationParser()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("parses the targets and flags from a buildpack.yml", func() {
		targets, flags, err := parser.Parse(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(targets).To(Equal([]string{"./first", "./second"}))
		Expect(flags).To(Equal([]string{
			"-first", "value",
			"-second", "value",
			"-third", "value",
			"-fourth", "value",
		}))
	})

	context("when there is no buildpack.yml file", func() {
		it.Before(func() {
			Expect(os.Remove(path)).To(Succeed())
		})

		it("returns a list of targets with . as the only target, and empty list of flags", func() {
			targets, flags, err := parser.Parse(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(targets).To(Equal([]string{"."}))
			Expect(flags).To(BeEmpty())
		})

		context("BP_GO_TARGETS env variable is set", func() {
			it.Before(func() {
				os.Setenv("BP_GO_TARGETS", "./some/target1:./some/target2")
			})

			it.After(func() {
				os.Unsetenv("BP_GO_TARGETS")
			})

			it("uses the values in the env var", func() {
				targets, flags, err := parser.Parse(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(targets).To(Equal([]string{"./some/target1", "./some/target2"}))
				Expect(flags).To(BeEmpty())
			})
		})
	})

	context("when the targets list is empty", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(path, []byte("---\ngo:\n  targets: []\n"), 0644)).To(Succeed())
		})

		it("returns a list of targets with . as the only target", func() {
			targets, _, err := parser.Parse(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(targets).To(Equal([]string{"."}))
		})
	})

	context("BP_GO_TARGETS env variable is set", func() {
		it.Before(func() {
			os.Setenv("BP_GO_TARGETS", "./some/target1:./some/target2")
		})

		it.After(func() {
			os.Unsetenv("BP_GO_TARGETS")
		})

		it("replaces the targets list with the values in the env var", func() {
			targets, flags, err := parser.Parse(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(targets).To(Equal([]string{"./some/target1", "./some/target2"}))
			Expect(flags).To(Equal([]string{
				"-first", "value",
				"-second", "value",
				"-third", "value",
				"-fourth", "value",
			}))
		})
	})

	context("failure cases", func() {
		context("when the buildpack.yml file cannot be read", func() {
			it.Before(func() {
				Expect(os.Chmod(path, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, _, err := parser.Parse(path)
				Expect(err).To(MatchError(ContainSubstring("failed to read buildpack.yml:")))
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the buildpack.yml file cannot be read", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(path, []byte("%%%"), 0644)).To(Succeed())
			})

			it("returns an error", func() {
				_, _, err := parser.Parse(path)
				Expect(err).To(MatchError(ContainSubstring("failed to decode buildpack.yml:")))
				Expect(err).To(MatchError(ContainSubstring("could not find expected directive name")))
			})
		})

		context("when a target is an absolute path", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(path, []byte("---\ngo:\n  targets: [\"/some-target\"]\n"), 0644)).To(Succeed())
			})

			it("returns an error", func() {
				_, _, err := parser.Parse(path)
				Expect(err).To(MatchError(ContainSubstring("failed to determine build targets: \"/some-target\" is an absolute path, targets must be relative to the source directory")))
			})
		})
	})
}
