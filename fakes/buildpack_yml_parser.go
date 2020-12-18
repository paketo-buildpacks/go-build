package fakes

import (
	"sync"

	gobuild "github.com/paketo-buildpacks/go-build"
)

type BuildpackYMLParser struct {
	ParseCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
		}
		Returns struct {
			BuildConfiguration gobuild.BuildConfiguration
			Error              error
		}
		Stub func(string) (gobuild.BuildConfiguration, error)
	}
}

func (f *BuildpackYMLParser) Parse(param1 string) (gobuild.BuildConfiguration, error) {
	f.ParseCall.Lock()
	defer f.ParseCall.Unlock()
	f.ParseCall.CallCount++
	f.ParseCall.Receives.WorkingDir = param1
	if f.ParseCall.Stub != nil {
		return f.ParseCall.Stub(param1)
	}
	return f.ParseCall.Returns.BuildConfiguration, f.ParseCall.Returns.Error
}
