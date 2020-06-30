package fakes

import "sync"

type SourceRemover struct {
	ClearCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			Error error
		}
		Stub func(string) error
	}
}

func (f *SourceRemover) Clear(param1 string) error {
	f.ClearCall.Lock()
	defer f.ClearCall.Unlock()
	f.ClearCall.CallCount++
	f.ClearCall.Receives.Path = param1
	if f.ClearCall.Stub != nil {
		return f.ClearCall.Stub(param1)
	}
	return f.ClearCall.Returns.Error
}
