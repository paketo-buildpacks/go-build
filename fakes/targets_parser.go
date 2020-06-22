package fakes

import "sync"

type TargetsParser struct {
	ParseCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			Targets []string
			Err     error
		}
		Stub func(string) ([]string, error)
	}
}

func (f *TargetsParser) Parse(param1 string) ([]string, error) {
	f.ParseCall.Lock()
	defer f.ParseCall.Unlock()
	f.ParseCall.CallCount++
	f.ParseCall.Receives.Path = param1
	if f.ParseCall.Stub != nil {
		return f.ParseCall.Stub(param1)
	}
	return f.ParseCall.Returns.Targets, f.ParseCall.Returns.Err
}
