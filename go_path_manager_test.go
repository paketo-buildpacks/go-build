package gobuild_test

import (
	"os"
	"path/filepath"
	"testing"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testGoPathManager(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		tempDir     string
		pathManager gobuild.GoPathManager
	)

	it.Before(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		pathManager = gobuild.NewGoPathManager(tempDir)
	})

	it.After(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	context("Setup", func() {
		var workspacePath string

		it.Before(func() {
			var err error
			workspacePath, err = os.MkdirTemp("", "workspace")
			Expect(err).NotTo(HaveOccurred())

			Expect(os.WriteFile(filepath.Join(workspacePath, "some-file"), nil, 0644)).To(Succeed())
		})

		it.After(func() {
			Expect(os.RemoveAll(workspacePath)).To(Succeed())
		})

		it("sets up the GOPATH", func() {
			goPath, path, err := pathManager.Setup(workspacePath, "some/import/path")
			Expect(err).NotTo(HaveOccurred())

			Expect(goPath).NotTo(BeEmpty())
			Expect(goPath).To(HavePrefix(tempDir))

			Expect(path).To(HavePrefix(goPath))
			Expect(path).To(HaveSuffix("/src/some/import/path"))

			Expect(filepath.Join(path, "some-file")).To(BeARegularFile())
		})

		context("when the workspace contains a go.mod file", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workspacePath, "go.mod"), nil, 0644)).To(Succeed())
			})

			it("does not setup a GOPATH", func() {
				goPath, path, err := pathManager.Setup(workspacePath, "some/import/path")
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(workspacePath))

				Expect(goPath).To(BeEmpty())
			})
		})

		context("failure cases", func() {
			context("when a temporary directory cannot be created", func() {
				it.Before(func() {
					Expect(os.Chmod(tempDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(tempDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, _, err := pathManager.Setup(workspacePath, "some/import/path")
					Expect(err).To(MatchError(ContainSubstring("failed to setup GOPATH:")))
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when app source code cannot be copied", func() {
				it.Before(func() {
					Expect(os.Chmod(filepath.Join(workspacePath, "some-file"), 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, _, err := pathManager.Setup(workspacePath, "some/import/path")
					Expect(err).To(MatchError(ContainSubstring("failed to copy application source onto GOPATH:")))
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})

	context("Teardown", func() {
		var path string

		it.Before(func() {
			var err error
			path, err = os.MkdirTemp("", "gopath")
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(os.RemoveAll(path)).To(Succeed())
		})

		it("tears down the GOPATH", func() {
			Expect(pathManager.Teardown(path)).To(Succeed())

			Expect(path).NotTo(BeADirectory())
		})

		context("when the GOPATH is empty", func() {
			it("does nothing", func() {
				Expect(pathManager.Teardown("")).To(Succeed())

				Expect(path).To(BeADirectory())
			})
		})
	})
}
