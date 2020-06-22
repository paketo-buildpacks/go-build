package gobuild

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/fs"
)

type GoPathManager struct {
	tempDir string
}

func NewGoPathManager(tempDir string) GoPathManager {
	return GoPathManager{
		tempDir: tempDir,
	}
}

func (m GoPathManager) Setup(workspace string) (string, string, error) {
	_, err := os.Stat(filepath.Join(workspace, "go.mod"))
	if err == nil {
		return "", workspace, nil
	}

	path, err := ioutil.TempDir(m.tempDir, "gopath")
	if err != nil {
		return "", "", fmt.Errorf("failed to setup GOPATH: %w", err)
	}

	appPath := filepath.Join(path, "src", "workspace")
	err = os.MkdirAll(appPath, os.ModePerm)
	if err != nil {
		return "", "", fmt.Errorf("failed to setup GOPATH: %w", err)
	}

	err = fs.Copy(workspace, appPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to copy application source onto GOPATH: %w", err)
	}

	return path, appPath, nil
}

func (m GoPathManager) Teardown(path string) error {
	if path == "" {
		return nil
	}

	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("failed to teardown GOPATH: %w", err)
	}

	return nil
}
