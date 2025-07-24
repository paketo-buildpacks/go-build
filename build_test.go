package gobuild_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	gobuild "github.com/paketo-buildpacks/go-build"
	"github.com/paketo-buildpacks/go-build/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string
		logs       *bytes.Buffer

		buildProcess  *fakes.BuildProcess
		pathManager   *fakes.PathManager
		sourceRemover *fakes.SourceRemover
		parser        *fakes.ConfigurationParser
		sbomGenerator *fakes.SBOMGenerator

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buildProcess = &fakes.BuildProcess{}
		buildProcess.ExecuteCall.Returns.Binaries = []string{"path/some-start-command", "path/another-start-command"}

		pathManager = &fakes.PathManager{}
		pathManager.SetupCall.Returns.GoPath = "some-go-path"
		pathManager.SetupCall.Returns.Path = "some-app-path"

		logs = bytes.NewBuffer(nil)

		sourceRemover = &fakes.SourceRemover{}

		parser = &fakes.ConfigurationParser{}
		parser.ParseCall.Returns.BuildConfiguration = gobuild.BuildConfiguration{
			Targets:    []string{"some-target", "other-target"},
			Flags:      []string{"some-flag", "other-flag"},
			ImportPath: "some-import-path",
		}

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateCall.Returns.SBOM = sbom.SBOM{}

		build = gobuild.Build(
			parser,
			buildProcess,
			pathManager,
			chronos.DefaultClock,
			scribe.NewEmitter(logs),
			sourceRemover,
			sbomGenerator,
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
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			Layers: packit.Layers{Path: layersDir},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))

		targets := result.Layers[0]
		Expect(targets.Name).To(Equal("targets"))
		Expect(targets.Path).To(Equal(filepath.Join(layersDir, "targets")))
		Expect(targets.Build).To(BeFalse())
		Expect(targets.Cache).To(BeFalse())
		Expect(targets.Launch).To(BeTrue())

		Expect(targets.SBOM.Formats()).To(HaveLen(2))
		cdx := targets.SBOM.Formats()[0]
		spdx := targets.SBOM.Formats()[1]

		Expect(cdx.Extension).To(Equal("cdx.json"))
		content, err := io.ReadAll(cdx.Content)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(MatchJSON(`{
			"$schema": "http://cyclonedx.org/schema/bom-1.3.schema.json",
			"bomFormat": "CycloneDX",
			"metadata": {
				"tools": [
					{
						"name": "",
						"vendor": "anchore"
					}
				]
			},
			"specVersion": "1.3",
			"version": 1
		}`))

		Expect(spdx.Extension).To(Equal("spdx.json"))
		content, err = io.ReadAll(spdx.Content)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(MatchJSON(`{
			"SPDXID": "SPDXRef-DOCUMENT",
			"creationInfo": {
				"created": "0001-01-01T00:00:00Z",
				"creators": [
					"Organization: Anchore, Inc",
					"Tool: -"
				],
				"licenseListVersion": "3.25"
			},
			"dataLicense": "CC0-1.0",
			"documentNamespace": "https://paketo.io/unknown-source-type/unknown-9ecf240a-d971-5a3c-8e7b-6d3f3ea4d9c2",
			"name": "unknown",
			"packages": [
				{
					"SPDXID": "SPDXRef-DocumentRoot-Unknown-",
					"copyrightText": "NOASSERTION",
					"downloadLocation": "NOASSERTION",
					"filesAnalyzed": false,
					"licenseConcluded": "NOASSERTION",
					"licenseDeclared": "NOASSERTION",
					"name": "",
					"supplier": "NOASSERTION"
				}
			],
			"relationships": [
				{
					"relatedSpdxElement": "SPDXRef-DocumentRoot-Unknown-",
					"relationshipType": "DESCRIBES",
					"spdxElementId": "SPDXRef-DOCUMENT"
				}
			],
			"spdxVersion": "SPDX-2.2"
		}`))

		gocache := result.Layers[1]
		Expect(gocache.Name).To(Equal("gocache"))
		Expect(gocache.Path).To(Equal(filepath.Join(layersDir, "gocache")))
		Expect(gocache.Build).To(BeFalse())
		Expect(gocache.Cache).To(BeTrue())
		Expect(gocache.Launch).To(BeFalse())

		Expect(result.Launch.Processes).To(Equal([]packit.Process{
			{
				Type:    "some-start-command",
				Command: "path/some-start-command",
				Direct:  true,
				Default: true,
			},
			{
				Type:    "another-start-command",
				Command: "path/another-start-command",
				Direct:  true,
			},
		}))

		Expect(parser.ParseCall.Receives.BuildpackVersion).To(Equal("some-version"))
		Expect(parser.ParseCall.Receives.WorkingDir).To(Equal(workingDir))

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
		Expect(sbomGenerator.GenerateCall.Receives.Dir).To(Equal(filepath.Join(targets.Path, "bin")))

		Expect(logs.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(logs.String()).To(ContainSubstring("Assigning launch processes"))
		Expect(logs.String()).To(ContainSubstring("some-start-command (default): path/some-start-command"))
		Expect(logs.String()).To(ContainSubstring("another-start-command:        path/another-start-command"))
	})

	context("BP_LIVE_RELOAD_ENABLED=true in the build environment", func() {
		it.Before(func() {
			os.Setenv("BP_LIVE_RELOAD_ENABLED", "true")
		})

		it.After(func() {
			os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
		})

		it("wraps the target process(es) in watchexec", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Launch).To(Equal(packit.LaunchMetadata{
				Processes: []packit.Process{
					{
						Type:    "some-start-command",
						Command: "path/some-start-command",
						Direct:  true,
					},
					{
						Type:    "reload-some-start-command",
						Command: "watchexec",
						Args: []string{
							"--restart",
							"--watch", workingDir,
							"--watch", "path",
							"--shell", "none",
							"--",
							"path/some-start-command"},
						Direct:  true,
						Default: true,
					},
					{
						Type:    "another-start-command",
						Command: "path/another-start-command",
						Direct:  true,
					},
					{
						Type:    "reload-another-start-command",
						Command: "watchexec",
						Args: []string{
							"--restart",
							"--watch", workingDir,
							"--watch", "path",
							"--shell", "none",
							"--",
							"path/another-start-command"},
						Direct: true,
					},
				},
			}))
		})
	})

	context("when the stack is static", func() {
		it("sets CGO_ENABLED=0 and -buildmode=default", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "io.buildpacks.stacks.jammy.static",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Launch.Processes).To(Equal([]packit.Process{
				{
					Type:    "some-start-command",
					Command: "path/some-start-command",
					Direct:  true,
					Default: true,
				},
				{
					Type:    "another-start-command",
					Command: "path/another-start-command",
					Direct:  true,
				},
			}))

			receivedConfig := buildProcess.ExecuteCall.Receives.Config

			Expect(receivedConfig.DisableCGO).To(BeTrue())
			Expect(receivedConfig.Flags).To(Equal([]string{"some-flag", "other-flag", "-buildmode", "default"}))
		})

		context("there is a pre-existing -buildmode flag", func() {
			it.Before(func() {
				parser.ParseCall.Returns.BuildConfiguration = gobuild.BuildConfiguration{
					Flags: []string{"-buildmode", "some-provided-buildmode"},
				}
			})

			it("does not set CGO_ENABLED=0 or -buildmode=default", func() {
				result, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "io.buildpacks.stacks.jammy.static",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(result.Launch.Processes).To(Equal([]packit.Process{
					{
						Type:    "some-start-command",
						Command: "path/some-start-command",
						Direct:  true,
						Default: true,
					},
					{
						Type:    "another-start-command",
						Command: "path/another-start-command",
						Direct:  true,
					},
				}))

				receivedConfig := buildProcess.ExecuteCall.Receives.Config

				Expect(receivedConfig.DisableCGO).To(BeFalse())
				Expect(receivedConfig.Flags).To(Equal([]string{"-buildmode", "some-provided-buildmode"}))
			})
		})
	})

	context("failure cases", func() {
		context("when the targets layer cannot be retrieved", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(layersDir, "targets.toml"), nil, 0000)).To(Succeed())
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
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the gocache layer cannot be retrieved", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(layersDir, "gocache.toml"), nil, 0000)).To(Succeed())
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
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
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
				})
				Expect(err).To(MatchError("failed to teardown go path"))
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
				})
				Expect(err).To(MatchError("failed to remove source"))
			})
		})

		context("when an SBOM cannot be generated", func() {
			it.Before(func() {
				sbomGenerator.GenerateCall.Returns.Error = errors.New("sbom generation error")
			})
			it("fails the build and returns the error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "io.paketo.stacks.tiny",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError("sbom generation error"))
			})
		})

		context("when a requested SBOM format is invalid", func() {
			it("fails the build and returns the error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "io.paketo.stacks.tiny",
					BuildpackInfo: packit.BuildpackInfo{
						Name:        "Some Buildpack",
						Version:     "some-version",
						SBOMFormats: []string{"invalid-format"},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(`unsupported SBOM format: 'invalid-format'`))
			})
		})

		context("when BP_LIVE_RELOAD_ENABLED value is invalid", func() {
			it.Before(func() {
				os.Setenv("BP_LIVE_RELOAD_ENABLED", "not-a-bool")
			})

			it.After(func() {
				os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
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
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse BP_LIVE_RELOAD_ENABLED value not-a-bool")))
			})
		})
	})
}
