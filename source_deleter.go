package gobuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SourceDeleter struct{}

func NewSourceDeleter() SourceDeleter {
	return SourceDeleter{}
}

func (d SourceDeleter) Clear(path string) error {
	var envGlobs []string
	if val, ok := os.LookupEnv("BP_KEEP_FILES"); ok {
		envGlobs = append(envGlobs, filepath.SplitList(val)...)
	}

	// This is logic taken from github.com/ForestEckhardt/source-removal/build.go
	//
	// The following constructs a set of all the file paths that are required
	// from a globed file to exist and prepends the working directory onto all of
	// those permutation
	//
	// Example:
	// Input: "public/data/*"
	// Output: ["path/public", "path/public/data", "path/public/data/*"]
	var globs = []string{path}
	for _, glob := range envGlobs {
		dirs := strings.Split(glob, string(os.PathSeparator))
		for i := range dirs {
			globs = append(globs, filepath.Join(path, filepath.Join(dirs[:i+1]...)))
		}
	}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		match, err := matchingGlob(path, globs)
		if err != nil {
			return err
		}

		if !match {
			err := os.RemoveAll(path)
			if err != nil {
				return err
			}

			if info.IsDir() {
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to remove source: %w", err)
	}

	return nil
}

func matchingGlob(path string, globs []string) (bool, error) {
	for _, glob := range globs {
		match, err := filepath.Match(glob, path)
		if err != nil {
			return false, err
		}

		if match {
			// filepath.SkipDir is returned here because this is a glob that
			// specifies everything in a directroy. If we get a match on such
			// a glob we want to ignore all other files in that directory because
			// they are files we want to keep and the glob will not work
			// if it enters that directory
			if strings.HasSuffix(glob, fmt.Sprintf("%c*", os.PathSeparator)) {
				return true, filepath.SkipDir
			}
			return true, nil
		}
	}

	return false, nil
}
