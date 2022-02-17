package gobuild_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/paketo-buildpacks/go-build/fakes"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testGoBuildProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerPath     string
		workspacePath string
		goPath        string
		goCache       string
		executions    []pexec.Execution

		executable *fakes.Executable
		logs       *bytes.Buffer

		buildProcess gobuild.GoBuildProcess
	)

	it.Before(func() {
		var err error
		layerPath, err = os.MkdirTemp("", "layer")
		Expect(err).NotTo(HaveOccurred())

		workspacePath, err = os.MkdirTemp("", "workspace")
		Expect(err).NotTo(HaveOccurred())

		goPath, err = os.MkdirTemp("", "go-path")
		Expect(err).NotTo(HaveOccurred())

		goCache, err = os.MkdirTemp("", "gocache")
		Expect(err).NotTo(HaveOccurred())

		logs = bytes.NewBuffer(nil)

		executable = &fakes.Executable{}
		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			executions = append(executions, execution)

			if execution.Args[0] == "list" {
				fmt.Fprintf(execution.Stdout, `{
					"ImportPath": "%s"
				}`, filepath.Join("some-dir", execution.Args[len(execution.Args)-1]))
			}
			return nil
		}

		now := time.Now()
		times := []time.Time{now, now.Add(1 * time.Second)}

		clock := chronos.NewClock(func() time.Time {
			if len(times) == 0 {
				return time.Now()
			}

			t := times[0]
			times = times[1:]
			return t
		})

		buildProcess = gobuild.NewGoBuildProcess(executable, scribe.NewEmitter(logs), clock)
	})

	it.After(func() {
		Expect(os.RemoveAll(layerPath)).To(Succeed())
		Expect(os.RemoveAll(workspacePath)).To(Succeed())
		Expect(os.RemoveAll(goPath)).To(Succeed())
		Expect(os.RemoveAll(goCache)).To(Succeed())
	})

	it("executes the go build process", func() {
		binaries, err := buildProcess.Execute(gobuild.GoBuildConfiguration{
			Workspace: workspacePath,
			Output:    filepath.Join(layerPath, "bin"),
			GoPath:    goPath,
			GoCache:   goCache,
			Targets:   []string{"./some-target", "./other-target"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(binaries).To(Equal([]string{
			filepath.Join(layerPath, "bin", "some-target"),
			filepath.Join(layerPath, "bin", "other-target"),
		}))

		Expect(filepath.Join(layerPath, "bin")).To(BeADirectory())

		Expect(executions[0].Args).To(Equal([]string{
			"build",
			"-o", filepath.Join(layerPath, "bin"),
			"-buildmode", "pie",
			"-trimpath",
			"./some-target", "./other-target",
		}))

		Expect(executions[1].Args).To(Equal([]string{
			"list",
			"--json",
			"./some-target",
		}))

		Expect(executions[2].Args).To(Equal([]string{
			"list",
			"--json",
			"./other-target",
		}))

		Expect(executable.ExecuteCall.Receives.Execution.Dir).To(Equal(workspacePath))
		Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf("GOPATH=%s", goPath)))
		Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf("GOCACHE=%s", goCache)))
		Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement("GO111MODULE=auto"))

		Expect(logs.String()).To(ContainSubstring("  Executing build process"))
		Expect(logs.String()).To(ContainSubstring(fmt.Sprintf("    Running 'go build -o %s -buildmode pie -trimpath ./some-target ./other-target'", filepath.Join(layerPath, "bin"))))
		Expect(logs.String()).To(ContainSubstring("      Completed in 1s"))
	})

	context("when there are build flags", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(workspacePath, "go.mod"), nil, 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(workspacePath, "vendor"), os.ModePerm)).To(Succeed())
		})

		it("executes the go build process with those flags", func() {
			binaries, err := buildProcess.Execute(gobuild.GoBuildConfiguration{
				Workspace: workspacePath,
				Output:    filepath.Join(layerPath, "bin"),
				GoCache:   goCache,
				Targets:   []string{"."},
				Flags:     []string{"-buildmode", "default", "-ldflags", "-X main.variable=some-value", "-mod", "mod", "-trimpath"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(binaries).To(Equal([]string{
				filepath.Join(layerPath, "bin", "some-dir"),
			}))

			Expect(filepath.Join(layerPath, "bin")).To(BeADirectory())

			Expect(executions[0].Args).To(Equal([]string{
				"build",
				"-o", filepath.Join(layerPath, "bin"),
				"-buildmode", "default",
				"-ldflags", "-X main.variable=some-value",
				"-mod", "mod",
				"-trimpath",
				".",
			}))

			Expect(executions[1].Args).To(Equal([]string{
				"list",
				"--json",
				".",
			}))

			Expect(executable.ExecuteCall.Receives.Execution.Dir).To(Equal(workspacePath))
			Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf("GOCACHE=%s", goCache)))

			Expect(logs).To(ContainLines(
				"  Executing build process",
				fmt.Sprintf(`    Running 'go build -o %s -buildmode default -ldflags "-X main.variable=some-value" -mod mod -trimpath .'`, filepath.Join(layerPath, "bin")),
				"      Completed in 1s",
			))
		})
	})

	context("when the GOPATH is empty", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(workspacePath, "go.mod"), nil, 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(workspacePath, "vendor"), os.ModePerm)).To(Succeed())
		})

		it("executes the go build process without setting GOPATH", func() {
			binaries, err := buildProcess.Execute(gobuild.GoBuildConfiguration{
				Workspace: workspacePath,
				Output:    filepath.Join(layerPath, "bin"),
				GoPath:    "",
				GoCache:   goCache,
				Targets:   []string{"./other-target", "./some-target"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(binaries).To(Equal([]string{
				filepath.Join(layerPath, "bin", "other-target"),
				filepath.Join(layerPath, "bin", "some-target"),
			}))

			Expect(filepath.Join(layerPath, "bin")).To(BeADirectory())

			Expect(executable.ExecuteCall.Receives.Execution.Env).NotTo(ContainElement("GOPATH="))
		})
	})

	context("failure cases", func() {
		context("when the output directory cannot be created", func() {
			it.Before(func() {
				Expect(os.Chmod(layerPath, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := buildProcess.Execute(gobuild.GoBuildConfiguration{
					Workspace: workspacePath,
					Output:    filepath.Join(layerPath, "bin"),
					GoPath:    goPath,
					GoCache:   goCache,
					Targets:   []string{"./some-target", "./other-target"},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to create targets output directory:")))
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the executable fails go build", func() {
			it.Before(func() {
				executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
					fmt.Fprintln(execution.Stdout, "build error stdout")
					fmt.Fprintln(execution.Stderr, "build error stderr")

					return errors.New("command failed")
				}
			})

			it("returns an error", func() {
				_, err := buildProcess.Execute(gobuild.GoBuildConfiguration{
					Workspace: workspacePath,
					Output:    filepath.Join(layerPath, "bin"),
					GoPath:    goPath,
					GoCache:   goCache,
					Targets:   []string{"./some-target", "./other-target"},
				})
				Expect(err).To(MatchError("failed to execute 'go build': command failed"))

				Expect(logs).To(ContainLines(
					"      Failed after 1s",
					"        build error stdout",
					"        build error stderr",
				))
			})
		})

		context("when the executable fails go list", func() {
			it.Before(func() {
				executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
					if execution.Args[0] == "list" {
						fmt.Fprintln(execution.Stdout, "build error stdout")
						fmt.Fprintln(execution.Stderr, "build error stderr")
						return errors.New("command failed")
					}

					return nil
				}
			})

			it("returns an error", func() {
				_, err := buildProcess.Execute(gobuild.GoBuildConfiguration{
					Workspace: workspacePath,
					Output:    filepath.Join(layerPath, "bin"),
					GoPath:    goPath,
					GoCache:   goCache,
					Targets:   []string{"./some-target", "./other-target"},
				})
				Expect(err).To(MatchError("failed to execute 'go list': command failed"))

				Expect(logs).To(ContainLines(
					"        build error stdout",
					"        build error stderr",
				))
			})
		})

		context("when the json parse of go list fails", func() {
			it.Before(func() {
				executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
					if execution.Args[0] == "list" {
						fmt.Fprintln(execution.Stdout, "%%%")
					}

					return nil
				}
			})

			it("returns an error", func() {
				_, err := buildProcess.Execute(gobuild.GoBuildConfiguration{
					Workspace: workspacePath,
					Output:    filepath.Join(layerPath, "bin"),
					GoPath:    goPath,
					GoCache:   goCache,
					Targets:   []string{"./some-target", "./other-target"},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse 'go list' output:")))
			})
		})
	})
}
