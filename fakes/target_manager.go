package fakes

import "sync"

type TargetManager struct {
	CleanAndValidateCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Targets    []string
			WorkingDir string
		}
		Returns struct {
			StringSlice []string
			Error       error
		}
		Stub func([]string, string) ([]string, error)
	}
	GenerateDefaultsCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
		}
		Returns struct {
			StringSlice []string
			Error       error
		}
		Stub func(string) ([]string, error)
	}
}

func (f *TargetManager) CleanAndValidate(param1 []string, param2 string) ([]string, error) {
	f.CleanAndValidateCall.Lock()
	defer f.CleanAndValidateCall.Unlock()
	f.CleanAndValidateCall.CallCount++
	f.CleanAndValidateCall.Receives.Targets = param1
	f.CleanAndValidateCall.Receives.WorkingDir = param2
	if f.CleanAndValidateCall.Stub != nil {
		return f.CleanAndValidateCall.Stub(param1, param2)
	}
	return f.CleanAndValidateCall.Returns.StringSlice, f.CleanAndValidateCall.Returns.Error
}
func (f *TargetManager) GenerateDefaults(param1 string) ([]string, error) {
	f.GenerateDefaultsCall.Lock()
	defer f.GenerateDefaultsCall.Unlock()
	f.GenerateDefaultsCall.CallCount++
	f.GenerateDefaultsCall.Receives.WorkingDir = param1
	if f.GenerateDefaultsCall.Stub != nil {
		return f.GenerateDefaultsCall.Stub(param1)
	}
	return f.GenerateDefaultsCall.Returns.StringSlice, f.GenerateDefaultsCall.Returns.Error
}
