package gobuild_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSourceDeleter(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string

		deleter gobuild.SourceDeleter
	)

	it.Before(func() {
		var err error
		path, err = ioutil.TempDir("", "source")
		Expect(err).NotTo(HaveOccurred())

		Expect(ioutil.WriteFile(filepath.Join(path, "some-file"), nil, os.ModePerm)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "some-dir", "some-other-dir", "another-dir"), os.ModePerm)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "some-dir", "some-file"), nil, os.ModePerm)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "some-dir", "some-other-dir", "some-file"), nil, os.ModePerm)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "some-dir", "some-other-dir", "another-dir", "some-file"), nil, os.ModePerm)).To(Succeed())

		deleter = gobuild.NewSourceDeleter()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("deletes the source code from the given directory path", func() {
		Expect(deleter.Clear(path)).To(Succeed())

		paths, err := filepath.Glob(filepath.Join(path, "*"))
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(BeEmpty())
	})

	context("when there are files to keep", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_KEEP_FILES", `some-dir/some-other-dir/*:some-file`)).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_KEEP_FILES")).To(Succeed())
		})

		it("returns a result that deletes the contents of the working directroy except for the file that are meant to kept", func() {
			Expect(deleter.Clear(path)).To(Succeed())

			Expect(path).To(BeADirectory())
			Expect(filepath.Join(path, "some-file")).To(BeAnExistingFile())
			Expect(filepath.Join(path, "some-dir")).To(BeADirectory())
			Expect(filepath.Join(path, "some-dir", "some-file")).NotTo(BeAnExistingFile())
			Expect(filepath.Join(path, "some-dir", "some-other-dir", "some-file")).To(BeAnExistingFile())
			Expect(filepath.Join(path, "some-dir", "some-other-dir", "another-dir", "some-file")).To(BeAnExistingFile())
		})
	})

	context("failure cases", func() {
		context("when the path is malformed", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_KEEP_FILES", `\`)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_KEEP_FILES")).To(Succeed())
			})

			it("returns an error", func() {
				err := deleter.Clear(path)

				Expect(err).To(MatchError(ContainSubstring("failed to remove source:")))
				Expect(err).To(MatchError(ContainSubstring("syntax error in pattern")))
			})
		})
	})
}
