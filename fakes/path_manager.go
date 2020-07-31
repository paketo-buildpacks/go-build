package fakes

import "sync"

type PathManager struct {
	SetupCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Workspace  string
			ImportPath string
		}
		Returns struct {
			GoPath string
			Path   string
			Err    error
		}
		Stub func(string, string) (string, string, error)
	}
	TeardownCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			GoPath string
		}
		Returns struct {
			Error error
		}
		Stub func(string) error
	}
}

func (f *PathManager) Setup(param1 string, param2 string) (string, string, error) {
	f.SetupCall.Lock()
	defer f.SetupCall.Unlock()
	f.SetupCall.CallCount++
	f.SetupCall.Receives.Workspace = param1
	f.SetupCall.Receives.ImportPath = param2
	if f.SetupCall.Stub != nil {
		return f.SetupCall.Stub(param1, param2)
	}
	return f.SetupCall.Returns.GoPath, f.SetupCall.Returns.Path, f.SetupCall.Returns.Err
}
func (f *PathManager) Teardown(param1 string) error {
	f.TeardownCall.Lock()
	defer f.TeardownCall.Unlock()
	f.TeardownCall.CallCount++
	f.TeardownCall.Receives.GoPath = param1
	if f.TeardownCall.Stub != nil {
		return f.TeardownCall.Stub(param1)
	}
	return f.TeardownCall.Returns.Error
}
