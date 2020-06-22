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

		Expect(ioutil.WriteFile(filepath.Join(path, "some-file"), nil, 0644)).To(Succeed())

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

	context("failure cases", func() {
		context("when the path is malformed", func() {
			it("returns an error", func() {
				err := deleter.Clear(`\`)
				Expect(err).To(MatchError(ContainSubstring("failed to remove source:")))
				Expect(err).To(MatchError(ContainSubstring("syntax error in pattern")))
			})
		})
	})
}
