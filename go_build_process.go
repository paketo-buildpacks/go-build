package gobuild

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) (err error)
}

type GoBuildConfiguration struct {
	Workspace string
	Output    string
	GoPath    string
	GoCache   string
	Targets   []string
	Flags     []string
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

func (p GoBuildProcess) Execute(config GoBuildConfiguration) (string, error) {
	p.logs.Process("Executing build process")

	err := os.MkdirAll(config.Output, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create targets output directory: %w", err)
	}

	contains := func(flags []string, match string) bool {
		for _, flag := range flags {
			if flag == match {
				return true
			}
		}

		return false
	}

	if !contains(config.Flags, "-buildmode") {
		config.Flags = append(config.Flags, "-buildmode", "pie")
	}

	if _, err = os.Stat(filepath.Join(config.Workspace, "go.mod")); err == nil {
		if _, err = os.Stat(filepath.Join(config.Workspace, "vendor")); err == nil {
			if !contains(config.Flags, "-mod") {
				config.Flags = append(config.Flags, "-mod", "vendor")
			}
		}
	}

	args := append([]string{"build", "-o", config.Output}, config.Flags...)
	args = append(args, config.Targets...)

	env := append(os.Environ(), fmt.Sprintf("GOCACHE=%s", config.GoCache))
	if config.GoPath != "" {
		env = append(env, fmt.Sprintf("GOPATH=%s", config.GoPath))
	}

	printedArgs := []string{"go"}
	for _, arg := range args {
		printedArgs = append(printedArgs, formatArg(arg))
	}
	p.logs.Subprocess("Running '%s'", strings.Join(printedArgs, " "))

	buffer := bytes.NewBuffer(nil)
	duration, err := p.clock.Measure(func() error {
		return p.executable.Execute(pexec.Execution{
			Args:   args,
			Dir:    config.Workspace,
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

	paths, err := filepath.Glob(fmt.Sprintf("%s/*", config.Output))
	if err != nil {
		return "", fmt.Errorf("failed to list targets: %w", err)
	}

	if len(paths) == 0 {
		return "", errors.New("failed to determine go executable start command")
	}

	return paths[0], nil
}

func formatArg(arg string) string {
	for _, r := range arg {
		if unicode.IsSpace(r) {
			return strconv.Quote(arg)
		}
	}

	return arg
}
