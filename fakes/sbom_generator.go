package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit/v2/sbom"
)

type SBOMGenerator struct {
	GenerateCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Dir string
		}
		Returns struct {
			SBOM  sbom.SBOM
			Error error
		}
		Stub func(string) (sbom.SBOM, error)
	}
}

func (f *SBOMGenerator) Generate(param1 string) (sbom.SBOM, error) {
	f.GenerateCall.mutex.Lock()
	defer f.GenerateCall.mutex.Unlock()
	f.GenerateCall.CallCount++
	f.GenerateCall.Receives.Dir = param1
	if f.GenerateCall.Stub != nil {
		return f.GenerateCall.Stub(param1)
	}
	return f.GenerateCall.Returns.SBOM, f.GenerateCall.Returns.Error
}
