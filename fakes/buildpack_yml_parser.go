package fakes

import (
	"sync"

	gobuild "github.com/paketo-buildpacks/go-build"
)

type BuildpackYMLParser struct {
	ParseCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			BuildpackVersion string
			WorkingDir       string
		}
		Returns struct {
			BuildConfiguration gobuild.BuildConfiguration
			Error              error
		}
		Stub func(string, string) (gobuild.BuildConfiguration, error)
	}
}

func (f *BuildpackYMLParser) Parse(param1 string, param2 string) (gobuild.BuildConfiguration, error) {
	f.ParseCall.mutex.Lock()
	defer f.ParseCall.mutex.Unlock()
	f.ParseCall.CallCount++
	f.ParseCall.Receives.BuildpackVersion = param1
	f.ParseCall.Receives.WorkingDir = param2
	if f.ParseCall.Stub != nil {
		return f.ParseCall.Stub(param1, param2)
	}
	return f.ParseCall.Returns.BuildConfiguration, f.ParseCall.Returns.Error
}
