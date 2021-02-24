package fakes

import (
	"sync"

	gobuild "github.com/paketo-buildpacks/go-build"
)

type BuildProcess struct {
	ExecuteCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Config gobuild.GoBuildConfiguration
		}
		Returns struct {
			Binaries []string
			Err      error
		}
		Stub func(gobuild.GoBuildConfiguration) ([]string, error)
	}
}

func (f *BuildProcess) Execute(param1 gobuild.GoBuildConfiguration) ([]string, error) {
	f.ExecuteCall.Lock()
	defer f.ExecuteCall.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.Config = param1
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1)
	}
	return f.ExecuteCall.Returns.Binaries, f.ExecuteCall.Returns.Err
}
