package gobuild_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/paketo-buildpacks/go-build/fakes"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testGoBuildProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerPath     string
		workspacePath string
		goPath        string
		goCache       string
		executable    *fakes.Executable
		logs          *bytes.Buffer

		buildProcess gobuild.GoBuildProcess
	)

	it.Before(func() {
		var err error
		layerPath, err = ioutil.TempDir("", "layer")
		Expect(err).NotTo(HaveOccurred())

		workspacePath, err = ioutil.TempDir("", "workspace")
		Expect(err).NotTo(HaveOccurred())

		goPath, err = ioutil.TempDir("", "go-path")
		Expect(err).NotTo(HaveOccurred())

		goCache, err = ioutil.TempDir("", "gocache")
		Expect(err).NotTo(HaveOccurred())

		executable = &fakes.Executable{}
		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			path := execution.Args[2]

			if err := ioutil.WriteFile(filepath.Join(path, "c_command"), nil, 0755); err != nil {
				return err
			}

			if err := ioutil.WriteFile(filepath.Join(path, "b_command"), nil, 0755); err != nil {
				return err
			}

			if err := ioutil.WriteFile(filepath.Join(path, "a_command"), nil, 0755); err != nil {
				return err
			}

			return nil
		}

		logs = bytes.NewBuffer(nil)

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

		buildProcess = gobuild.NewGoBuildProcess(executable, gobuild.NewLogEmitter(logs), clock)
	})

	it.After(func() {
		Expect(os.RemoveAll(layerPath)).To(Succeed())
		Expect(os.RemoveAll(workspacePath)).To(Succeed())
		Expect(os.RemoveAll(goPath)).To(Succeed())
		Expect(os.RemoveAll(goCache)).To(Succeed())
	})

	it("executes the go build process", func() {
		command, err := buildProcess.Execute(workspacePath, filepath.Join(layerPath, "bin"), goPath, goCache, []string{"./some-target", "./other-target"})
		Expect(err).NotTo(HaveOccurred())
		Expect(command).To(Equal(filepath.Join(layerPath, "bin", "a_command")))

		Expect(filepath.Join(layerPath, "bin")).To(BeADirectory())

		Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
			"build",
			"-o", filepath.Join(layerPath, "bin"),
			"-buildmode", "pie",
			"./some-target", "./other-target",
		}))
		Expect(executable.ExecuteCall.Receives.Execution.Dir).To(Equal(workspacePath))
		Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf("GOPATH=%s", goPath)))
		Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf("GOCACHE=%s", goCache)))

		Expect(logs.String()).To(ContainSubstring("  Executing build process"))
		Expect(logs.String()).To(ContainSubstring(fmt.Sprintf("    Running 'go build -o %s -buildmode pie ./some-target ./other-target'", filepath.Join(layerPath, "bin"))))
		Expect(logs.String()).To(ContainSubstring("      Completed in 1s"))
	})

	context("when the workspace contains a go.mod and a vendor directory", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(workspacePath, "go.mod"), nil, 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(workspacePath, "vendor"), os.ModePerm)).To(Succeed())
		})

		it("executes the go build process with -mod=vendor", func() {
			command, err := buildProcess.Execute(workspacePath, filepath.Join(layerPath, "bin"), goPath, goCache, []string{"./some-target", "./other-target"})
			Expect(err).NotTo(HaveOccurred())
			Expect(command).To(Equal(filepath.Join(layerPath, "bin", "a_command")))

			Expect(filepath.Join(layerPath, "bin")).To(BeADirectory())

			Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
				"build",
				"-o", filepath.Join(layerPath, "bin"),
				"-buildmode", "pie",
				"-mod", "vendor",
				"./some-target", "./other-target",
			}))
			Expect(executable.ExecuteCall.Receives.Execution.Dir).To(Equal(workspacePath))
		})
	})

	context("when the GOPATH is empty", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(workspacePath, "go.mod"), nil, 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(workspacePath, "vendor"), os.ModePerm)).To(Succeed())
		})

		it("executes the go build process without setting GOPATH", func() {
			command, err := buildProcess.Execute(workspacePath, filepath.Join(layerPath, "bin"), "", goCache, []string{"./some-target", "./other-target"})
			Expect(err).NotTo(HaveOccurred())
			Expect(command).To(Equal(filepath.Join(layerPath, "bin", "a_command")))

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
				_, err := buildProcess.Execute(workspacePath, filepath.Join(layerPath, "bin"), goPath, goCache, []string{"./some-target", "./other-target"})
				Expect(err).To(MatchError(ContainSubstring("failed to create targets output directory:")))
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the executable fails", func() {
			it.Before(func() {
				executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
					fmt.Fprintln(execution.Stdout, "build error stdout")
					fmt.Fprintln(execution.Stderr, "build error stderr")

					return errors.New("command failed")
				}
			})

			it("returns an error", func() {
				_, err := buildProcess.Execute(workspacePath, filepath.Join(layerPath, "bin"), goPath, goCache, []string{"./some-target", "./other-target"})
				Expect(err).To(MatchError("failed to execute 'go build': command failed"))

				Expect(logs.String()).To(ContainSubstring("      Failed after 1s"))
				Expect(logs.String()).To(ContainSubstring("        build error stdout"))
				Expect(logs.String()).To(ContainSubstring("        build error stderr"))
			})
		})

		context("when 'go build' doesn't create any executables", func() {
			it.Before(func() {
				executable.ExecuteCall.Stub = nil
			})

			it("returns an error", func() {
				_, err := buildProcess.Execute(workspacePath, filepath.Join(layerPath, "bin"), goPath, goCache, []string{"./some-target", "./other-target"})
				Expect(err).To(MatchError("failed to determine go executable start command"))
			})
		})
	})
}
