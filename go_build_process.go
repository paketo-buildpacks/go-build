package gobuild

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) (err error)
}

type GoBuildConfiguration struct {
	Workspace           string
	Output              string
	GoPath              string
	GoCache             string
	Targets             []string
	Flags               []string
	DisableCGO          bool
	WorkspaceUseModules []string
}

type GoBuildProcess struct {
	executable Executable
	logs       scribe.Emitter
	clock      chronos.Clock
}

func NewGoBuildProcess(executable Executable, logs scribe.Emitter, clock chronos.Clock) GoBuildProcess {
	return GoBuildProcess{
		executable: executable,
		logs:       logs,
		clock:      clock,
	}
}

func (p GoBuildProcess) Execute(config GoBuildConfiguration) ([]string, error) {
	p.logs.Process("Executing build process")

	err := os.MkdirAll(config.Output, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create targets output directory: %w", err)
	}

	if !containsFlag(config.Flags, "-buildmode") {
		config.Flags = append(config.Flags, "-buildmode", "pie")
	}

	if !containsFlag(config.Flags, "-trimpath") {
		config.Flags = append(config.Flags, "-trimpath")
	}

	args := append([]string{"build", "-o", config.Output}, config.Flags...)
	args = append(args, config.Targets...)

	env := append(os.Environ(), fmt.Sprintf("GOCACHE=%s", config.GoCache))
	if config.GoPath != "" {
		env = append(env, fmt.Sprintf("GOPATH=%s", config.GoPath))
	}
	env = append(env, "GO111MODULE=auto")

	if config.DisableCGO {
		env = append(env, "CGO_ENABLED=0")
	}

	if len(config.WorkspaceUseModules) > 0 {
		// go work init
		workInitArgs := []string{"work", "init"}
		p.logs.Subprocess("Running '%s'", strings.Join(append([]string{"go"}, workInitArgs...), " "))

		duration, err := p.clock.Measure(func() error {
			return p.executable.Execute(pexec.Execution{
				Args:   workInitArgs,
				Dir:    config.Workspace,
				Env:    env,
				Stdout: p.logs.ActionWriter,
				Stderr: p.logs.ActionWriter,
			})
		})
		if err != nil {
			p.logs.Action("Failed after %s", duration.Round(time.Millisecond))
			return nil, fmt.Errorf("failed to execute '%s': %w", workInitArgs, err)
		}

		// go work use <modules...>
		workUseArgs := append([]string{"work", "use"}, config.WorkspaceUseModules...)
		p.logs.Subprocess("Running '%s'", strings.Join(append([]string{"go"}, workUseArgs...), " "))

		duration, err = p.clock.Measure(func() error {
			return p.executable.Execute(pexec.Execution{
				Args:   workUseArgs,
				Dir:    config.Workspace,
				Env:    env,
				Stdout: p.logs.ActionWriter,
				Stderr: p.logs.ActionWriter,
			})
		})
		if err != nil {
			p.logs.Action("Failed after %s", duration.Round(time.Millisecond))
			return nil, fmt.Errorf("failed to execute '%s': %w", workUseArgs, err)
		}
	}

	printedArgs := []string{"go"}
	for _, arg := range args {
		printedArgs = append(printedArgs, formatArg(arg))
	}
	p.logs.Subprocess("Running '%s'", strings.Join(printedArgs, " "))

	duration, err := p.clock.Measure(func() error {
		return p.executable.Execute(pexec.Execution{
			Args:   args,
			Dir:    config.Workspace,
			Env:    env,
			Stdout: p.logs.ActionWriter,
			Stderr: p.logs.ActionWriter,
		})
	})
	if err != nil {
		p.logs.Action("Failed after %s", duration.Round(time.Millisecond))
		return nil, fmt.Errorf("failed to execute 'go build': %w", err)
	}

	p.logs.Action("Completed in %s", duration.Round(time.Millisecond))
	p.logs.Break()

	var paths []string
	for _, target := range config.Targets {
		buffer := bytes.NewBuffer(nil)
		err := p.executable.Execute(pexec.Execution{
			Args:   []string{"list", "--json", target},
			Dir:    config.Workspace,
			Env:    env,
			Stdout: buffer,
			Stderr: buffer,
		})
		if err != nil {
			p.logs.Detail(buffer.String())
			return nil, fmt.Errorf("failed to execute 'go list': %w", err)
		}

		var list struct {
			ImportPath string `json:"ImportPath"`
		}
		err = json.Unmarshal(buffer.Bytes(), &list)
		if err != nil {
			return nil, fmt.Errorf("failed to parse 'go list' output: %w", err)
		}

		paths = append(paths, filepath.Join(config.Output, filepath.Base(list.ImportPath)))
	}

	if len(paths) == 0 {
		return nil, errors.New("failed to determine go executable start command")
	}

	return paths, nil
}

func formatArg(arg string) string {
	for _, r := range arg {
		if unicode.IsSpace(r) {
			return strconv.Quote(arg)
		}
	}

	return arg
}
