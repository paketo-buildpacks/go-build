package fakes

import "sync"

type BuildProcess struct {
	ExecuteCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Workspace string
			Output    string
			GoPath    string
			GoCache   string
			Targets   []string
		}
		Returns struct {
			Command string
			Err     error
		}
		Stub func(string, string, string, string, []string) (string, error)
	}
}

func (f *BuildProcess) Execute(param1 string, param2 string, param3 string, param4 string, param5 []string) (string, error) {
	f.ExecuteCall.Lock()
	defer f.ExecuteCall.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.Workspace = param1
	f.ExecuteCall.Receives.Output = param2
	f.ExecuteCall.Receives.GoPath = param3
	f.ExecuteCall.Receives.GoCache = param4
	f.ExecuteCall.Receives.Targets = param5
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1, param2, param3, param4, param5)
	}
	return f.ExecuteCall.Returns.Command, f.ExecuteCall.Returns.Err
}
