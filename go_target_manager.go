package gobuild

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GoTargetManager struct{}

func NewGoTargetManager() GoTargetManager {
	return GoTargetManager{}
}

func (tm GoTargetManager) CleanAndValidate(inputTargets []string, workingDir string) ([]string, error) {
	var targets []string
	for _, t := range inputTargets {
		if strings.HasPrefix(t, string(filepath.Separator)) {
			return nil, fmt.Errorf("failed to determine build targets: %q is an absolute path, targets must be relative to the source directory", t)
		}

		target := filepath.Clean(t)

		files, err := filepath.Glob(filepath.Join(workingDir, target, "*.go"))
		if err != nil {
			return nil, err
		}

		if len(files) == 0 {
			return nil, fmt.Errorf("there were no *.go files present in %q", filepath.Join(workingDir, target))
		}

		targets = append(targets, fmt.Sprintf(".%c%s", filepath.Separator, target))
	}

	return targets, nil
}

func (tm GoTargetManager) GenerateDefaults(workingDir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(workingDir, "*.go"))
	if err != nil {
		return nil, err
	}

	if len(files) > 0 {
		return []string{"."}, nil
	}

	var targets []string
	err = filepath.Walk(filepath.Join(workingDir, "cmd"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			match, err := filepath.Match("*.go", info.Name())
			if err != nil {
				return err
			}

			if match {
				targets = append(targets, strings.ReplaceAll(filepath.Dir(path), workingDir, "."))
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	if len(targets) == 0 {
		return nil, errors.New("no *.go files could be found")
	}

	return targets, nil
}
