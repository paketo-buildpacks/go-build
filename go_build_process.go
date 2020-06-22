package gobuild

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) (err error)
}

type GoBuildProcess struct {
	executable Executable
	logs       LogEmitter
	clock      chronos.Clock
}

func NewGoBuildProcess(executable Executable, logs LogEmitter, clock chronos.Clock) GoBuildProcess {
	return GoBuildProcess{
		executable: executable,
		logs:       logs,
		clock:      clock,
	}
}

func (p GoBuildProcess) Execute(workspace, output, goPath, goCache string, targets []string) (string, error) {
	p.logs.Process("Executing build process")

	err := os.MkdirAll(output, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create targets output directory: %w", err)
	}

	args := []string{"build", "-o", output, "-buildmode", "pie"}

	if _, err = os.Stat(filepath.Join(workspace, "go.mod")); err == nil {
		if _, err = os.Stat(filepath.Join(workspace, "vendor")); err == nil {
			args = append(args, "-mod", "vendor")
		}
	}

	args = append(args, targets...)

	env := append(os.Environ(), fmt.Sprintf("GOCACHE=%s", goCache))
	if goPath != "" {
		env = append(env, fmt.Sprintf("GOPATH=%s", goPath))
	}

	p.logs.Subprocess("Running '%s'", strings.Join(append([]string{"go"}, args...), " "))

	buffer := bytes.NewBuffer(nil)
	duration, err := p.clock.Measure(func() error {
		return p.executable.Execute(pexec.Execution{
			Args:   args,
			Dir:    workspace,
			Env:    env,
			Stdout: buffer,
			Stderr: buffer,
		})
	})
	if err != nil {
		p.logs.Action("Failed after %s", duration.Round(time.Millisecond))
		p.logs.Detail(buffer.String())

		return "", fmt.Errorf("failed to execute 'go build': %w", err)
	}

	p.logs.Action("Completed in %s", duration.Round(time.Millisecond))
	p.logs.Break()

	paths, err := filepath.Glob(fmt.Sprintf("%s/*", output))
	if err != nil {
		return "", fmt.Errorf("failed to list targets: %w", err)
	}

	if len(paths) == 0 {
		return "", errors.New("failed to determine go executable start command")
	}

	return paths[0], nil
}
