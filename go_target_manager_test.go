package gobuild_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testGoTargetManager(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string

		targetManager gobuild.GoTargetManager
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		targetManager = gobuild.NewGoTargetManager()

	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("CleanAndValidate", func() {
		context("when the targets contain a *.go", func() {
			it.Before(func() {
				targetDir := filepath.Join(workingDir, "first")
				Expect(os.MkdirAll(targetDir, os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(targetDir, "main.go"), nil, 0644)).To(Succeed())

				targetDir = filepath.Join(workingDir, "second")
				Expect(os.MkdirAll(targetDir, os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(targetDir, "main.go"), nil, 0644)).To(Succeed())
			})

			it("returns a slice of targets that have been cleaned", func() {
				targets, err := targetManager.CleanAndValidate([]string{"first", "./second"}, workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(targets).To(Equal([]string{"./first", "./second"}))
			})
		})

		context("when one of the targets in an absolute path", func() {
			it("returns an error", func() {
				_, err := targetManager.CleanAndValidate([]string{"/first"}, workingDir)
				Expect(err).To(MatchError(ContainSubstring(`failed to determine build targets: "/first" is an absolute path, targets must be relative to the source directory`)))
			})
		})

		context("when one of the targets does not contain a *.go", func() {
			it.Before(func() {
				targetDir := filepath.Join(workingDir, "first")
				Expect(os.MkdirAll(targetDir, os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(targetDir, "main.go"), nil, 0644)).To(Succeed())

			})

			it("returns a slice of targets that have been cleaned", func() {
				_, err := targetManager.CleanAndValidate([]string{"first", "./second"}, workingDir)
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("there were no *.go files present in %q", filepath.Join(workingDir, "second")))))
			})
		})

		context("failure cases", func() {
			context("when file glob failes", func() {
				it("returns an error", func() {
					_, err := targetManager.CleanAndValidate([]string{`\`}, `\`)
					Expect(err).To(MatchError(ContainSubstring("syntax error in pattern")))

				})
			})
		})
	})

	context("GenerateDefaults", func() {
		context("when there is a *.go file in the workingDir", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(workingDir, "main.go"), nil, 0644)).To(Succeed())
			})

			it("returns . as the target", func() {
				targets, err := targetManager.GenerateDefaults(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(targets).To(Equal([]string{"."}))
			})
		})

		context("when there are go files nested inside of a ./cmd folder", func() {
			it.Before(func() {
				targetDir := filepath.Join(workingDir, "cmd", "first")
				Expect(os.MkdirAll(targetDir, os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(targetDir, "main.go"), nil, 0644)).To(Succeed())

				targetDir = filepath.Join(workingDir, "cmd", "something", "second")
				Expect(os.MkdirAll(targetDir, os.ModePerm)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(targetDir, "main.go"), nil, 0644)).To(Succeed())
			})

			it("returns a target list of all top level directories in ./cmd that contain *.go files", func() {
				targets, err := targetManager.GenerateDefaults(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(targets).To(Equal([]string{"./cmd/first", "./cmd/something/second"}))
			})
		})

		context("when there is no *.go in the app root in ./cmd", func() {
			it("returns a target list of all top level directories in ./cmd that contain *.go files", func() {
				_, err := targetManager.GenerateDefaults(workingDir)
				Expect(err).To(MatchError(ContainSubstring("no *.go files could be found")))
			})
		})

		context("failure cases", func() {
			context("when file glob failes", func() {
				it("returns an error", func() {
					_, err := targetManager.GenerateDefaults(`\`)
					Expect(err).To(MatchError(ContainSubstring("syntax error in pattern")))

				})
			})

			context("when the workingDir is unstatable", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := targetManager.GenerateDefaults(workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
