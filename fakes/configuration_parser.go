package fakes

import (
	"sync"

	gobuild "github.com/paketo-buildpacks/go-build"
)

type ConfigurationParser struct {
	ParseCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			BuildConfiguration gobuild.BuildConfiguration
			Error              error
		}
		Stub func(string) (gobuild.BuildConfiguration, error)
	}
}

func (f *ConfigurationParser) Parse(param1 string) (gobuild.BuildConfiguration, error) {
	f.ParseCall.Lock()
	defer f.ParseCall.Unlock()
	f.ParseCall.CallCount++
	f.ParseCall.Receives.Path = param1
	if f.ParseCall.Stub != nil {
		return f.ParseCall.Stub(param1)
	}
	return f.ParseCall.Returns.BuildConfiguration, f.ParseCall.Returns.Error
}
