package fakes

import "sync"

type ConfigurationParser struct {
	ParseCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			Targets []string
			Flags   []string
			Err     error
		}
		Stub func(string) ([]string, []string, error)
	}
}

func (f *ConfigurationParser) Parse(param1 string) ([]string, []string, error) {
	f.ParseCall.Lock()
	defer f.ParseCall.Unlock()
	f.ParseCall.CallCount++
	f.ParseCall.Receives.Path = param1
	if f.ParseCall.Stub != nil {
		return f.ParseCall.Stub(param1)
	}
	return f.ParseCall.Returns.Targets, f.ParseCall.Returns.Flags, f.ParseCall.Returns.Err
}
