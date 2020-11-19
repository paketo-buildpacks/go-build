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
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir     string
		workingDir    string
		cnbDir        string
		buildProcess  *fakes.BuildProcess
		pathManager   *fakes.PathManager
		calculator    *fakes.ChecksumCalculator
		logs          *bytes.Buffer
		timestamp     time.Time
		sourceRemover *fakes.SourceRemover
		parser        *fakes.ConfigurationParser

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buildProcess = &fakes.BuildProcess{}
		buildProcess.ExecuteCall.Returns.Command = "some-start-command"

		pathManager = &fakes.PathManager{}
		pathManager.SetupCall.Returns.GoPath = "some-go-path"
		pathManager.SetupCall.Returns.Path = "some-app-path"

		timestamp = time.Now()
		clock := chronos.NewClock(func() time.Time {
			return timestamp
		})

		calculator = &fakes.ChecksumCalculator{}
		calculator.SumCall.Returns.Sha = "some-workspace-sha"

		logs = bytes.NewBuffer(nil)

		sourceRemover = &fakes.SourceRemover{}

		parser = &fakes.ConfigurationParser{}
		parser.ParseCall.Returns.BuildConfiguration = gobuild.BuildConfiguration{
			Targets:    []string{"some-target", "other-target"},
			Flags:      []string{"some-flag", "other-flag"},
			ImportPath: "some-import-path",
		}

		build = gobuild.Build(
			parser,
			buildProcess,
			pathManager,
			clock,
			calculator,
			gobuild.NewLogEmitter(logs),
			sourceRemover,
		)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that builds correctly", func() {
		result, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
			Layers: packit.Layers{Path: layersDir},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "go-build",
						Metadata: map[string]interface{}{
							"targets":     []interface{}{"some-target", "other-target"},
							"flags":       []interface{}{"some-flag", "other-flag"},
							"import-path": "some-import-path",
						},
					},
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result).To(Equal(packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "go-build",
						Metadata: map[string]interface{}{
							"targets":     []interface{}{"some-target", "other-target"},
							"flags":       []interface{}{"some-flag", "other-flag"},
							"import-path": "some-import-path",
						},
					},
				},
			},
			Layers: []packit.Layer{
				{
					Name:      "targets",
					Path:      filepath.Join(layersDir, "targets"),
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    true,
					Cache:     false,
					Metadata: map[string]interface{}{
						"built_at":      timestamp.Format(time.RFC3339Nano),
						"command":       "some-start-command",
						"workspace_sha": "some-workspace-sha",
					},
				},
				{
					Name:      "gocache",
					Path:      filepath.Join(layersDir, "gocache"),
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    false,
					Cache:     true,
				},
			},
			Launch: packit.LaunchMetadata{
				Processes: []packit.Process{
					{
						Type:    "web",
						Command: "some-start-command",
						Direct:  false,
					},
				},
			},
		}))

		Expect(calculator.SumCall.Receives.Path).To(Equal(workingDir))

		Expect(pathManager.SetupCall.Receives.Workspace).To(Equal(workingDir))
		Expect(pathManager.SetupCall.Receives.ImportPath).To(Equal("some-import-path"))

		Expect(buildProcess.ExecuteCall.Receives.Config).To(Equal(gobuild.GoBuildConfiguration{
			Workspace: "some-app-path",
			Output:    filepath.Join(layersDir, "targets", "bin"),
			GoPath:    "some-go-path",
			GoCache:   filepath.Join(layersDir, "gocache"),
			Flags:     []string{"some-flag", "other-flag"},
			Targets:   []string{"some-target", "other-target"},
		}))

		Expect(pathManager.TeardownCall.Receives.GoPath).To(Equal("some-go-path"))

		Expect(sourceRemover.ClearCall.Receives.Path).To(Equal(workingDir))

		Expect(logs.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(logs.String()).To(ContainSubstring("Assigning launch processes"))
		Expect(logs.String()).To(ContainSubstring("web: some-start-command"))
	})

	context("when the workspace contents have not changed from a previous build", func() {
		it.Before(func() {
			layerContent := fmt.Sprintf("launch = true\n[metadata]\ncommand = \"some-start-command\"\nworkspace_sha = \"some-workspace-sha\"\nbuilt_at = %q\n", timestamp.Format(time.RFC3339Nano))
			Expect(ioutil.WriteFile(filepath.Join(layersDir, "targets.toml"), []byte(layerContent), 0644)).To(Succeed())
		})

		it("skips the build process", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Layers: packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "go-build",
							Metadata: map[string]interface{}{
								"targets": []interface{}{"some-target", "other-target"},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(packit.BuildResult{
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "go-build",
							Metadata: map[string]interface{}{
								"targets": []interface{}{"some-target", "other-target"},
							},
						},
					},
				},
				Layers: []packit.Layer{
					{
						Name:      "targets",
						Path:      filepath.Join(layersDir, "targets"),
						SharedEnv: packit.Environment{},
						BuildEnv:  packit.Environment{},
						LaunchEnv: packit.Environment{},
						Build:     false,
						Launch:    true,
						Cache:     false,
						Metadata: map[string]interface{}{
							"built_at":      timestamp.Format(time.RFC3339Nano),
							"command":       "some-start-command",
							"workspace_sha": "some-workspace-sha",
						},
					},
					{
						Name:      "gocache",
						Path:      filepath.Join(layersDir, "gocache"),
						SharedEnv: packit.Environment{},
						BuildEnv:  packit.Environment{},
						LaunchEnv: packit.Environment{},
						Build:     false,
						Launch:    false,
						Cache:     true,
					},
				},
				Launch: packit.LaunchMetadata{
					Processes: []packit.Process{
						{
							Type:    "web",
							Command: "some-start-command",
							Direct:  false,
						},
					},
				},
			}))

			Expect(calculator.SumCall.Receives.Path).To(Equal(workingDir))
			Expect(pathManager.SetupCall.CallCount).To(Equal(0))
			Expect(buildProcess.ExecuteCall.CallCount).To(Equal(0))
			Expect(pathManager.TeardownCall.CallCount).To(Equal(0))

			Expect(logs.String()).To(ContainSubstring("Reusing cached layer"))
		})
	})

	context("when the stack is tiny", func() {
		it("marks the launch process as direct", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "io.paketo.stacks.tiny",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Layers: packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "go-build",
							Metadata: map[string]interface{}{
								"targets": []interface{}{"some-target", "other-target"},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(packit.BuildResult{
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "go-build",
							Metadata: map[string]interface{}{
								"targets": []interface{}{"some-target", "other-target"},
							},
						},
					},
				},
				Layers: []packit.Layer{
					{
						Name:      "targets",
						Path:      filepath.Join(layersDir, "targets"),
						SharedEnv: packit.Environment{},
						BuildEnv:  packit.Environment{},
						LaunchEnv: packit.Environment{},
						Build:     false,
						Launch:    true,
						Cache:     false,
						Metadata: map[string]interface{}{
							"built_at":      timestamp.Format(time.RFC3339Nano),
							"command":       "some-start-command",
							"workspace_sha": "some-workspace-sha",
						},
					},
					{
						Name:      "gocache",
						Path:      filepath.Join(layersDir, "gocache"),
						SharedEnv: packit.Environment{},
						BuildEnv:  packit.Environment{},
						LaunchEnv: packit.Environment{},
						Build:     false,
						Launch:    false,
						Cache:     true,
					},
				},
				Launch: packit.LaunchMetadata{
					Processes: []packit.Process{
						{
							Type:    "web",
							Command: "some-start-command",
							Direct:  true,
						},
					},
				},
			}))
		})
	})

	context("failure cases", func() {
		context("when the targets layer cannot be retrieved", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(layersDir, "targets.toml"), nil, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "go-build",
								Metadata: map[string]interface{}{
									"targets": []interface{}{"some-target", "other-target"},
								},
							},
						},
					},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the gocache layer cannot be retrieved", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(layersDir, "gocache.toml"), nil, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "go-build",
								Metadata: map[string]interface{}{
									"targets": []interface{}{"some-target", "other-target"},
								},
							},
						},
					},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the working dir cannot be checksummed", func() {
			it.Before(func() {
				calculator.SumCall.Returns.Err = errors.New("failed to checksum working dir")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "go-build",
								Metadata: map[string]interface{}{
									"targets": []interface{}{"some-target", "other-target"},
								},
							},
						},
					},
				})
				Expect(err).To(MatchError("failed to checksum working dir"))
			})
		})

		context("when the go path cannot be setup", func() {
			it.Before(func() {
				pathManager.SetupCall.Returns.Err = errors.New("failed to setup go path")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "go-build",
								Metadata: map[string]interface{}{
									"targets": []interface{}{"some-target", "other-target"},
								},
							},
						},
					},
				})
				Expect(err).To(MatchError("failed to setup go path"))
			})
		})

		context("when the build process fails", func() {
			it.Before(func() {
				buildProcess.ExecuteCall.Returns.Err = errors.New("failed to execute build process")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "go-build",
								Metadata: map[string]interface{}{
									"targets": []interface{}{"some-target", "other-target"},
								},
							},
						},
					},
				})
				Expect(err).To(MatchError("failed to execute build process"))
			})
		})

		context("when the go path cannot be torn down", func() {
			it.Before(func() {
				pathManager.TeardownCall.Returns.Error = errors.New("failed to teardown go path")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "go-build",
								Metadata: map[string]interface{}{
									"targets": []interface{}{"some-target", "other-target"},
								},
							},
						},
					},
				})
				Expect(err).To(MatchError("failed to teardown go path"))
			})
		})

		context("when cached targets layer is missing a command", func() {
			it.Before(func() {
				layerContent := fmt.Sprintf("launch = true\n[metadata]\nworkspace_sha = \"some-workspace-sha\"\nbuilt_at = %q\n", timestamp.Format(time.RFC3339Nano))
				Expect(ioutil.WriteFile(filepath.Join(layersDir, "targets.toml"), []byte(layerContent), 0644)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "go-build",
								Metadata: map[string]interface{}{
									"targets": []interface{}{"some-target", "other-target"},
								},
							},
						},
					},
				})
				Expect(err).To(MatchError("failed to identify start command from reused layer metadata"))
			})
		})

		context("when the source cannot be cleared", func() {
			it.Before(func() {
				sourceRemover.ClearCall.Returns.Error = errors.New("failed to remove source")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "go-build",
								Metadata: map[string]interface{}{
									"targets": []interface{}{"some-target", "other-target"},
								},
							},
						},
					},
				})
				Expect(err).To(MatchError("failed to remove source"))
			})
		})
	})
}
