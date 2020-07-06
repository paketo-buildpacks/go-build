package gobuild_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitGoBuild(t *testing.T) {
	suite := spec.New("go-build", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("BuildConfigurationParser", testBuildConfigurationParser)
	suite("Detect", testDetect)
	suite("GoBuildProcess", testGoBuildProcess)
	suite("GoPathManager", testGoPathManager)
	suite("SourceDeleter", testSourceDeleter)
	suite.Run(t)
}
