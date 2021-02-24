package gobuild

import (
	"io"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
)

type LogEmitter struct {
	scribe.Emitter
}

func NewLogEmitter(output io.Writer) LogEmitter {
	return LogEmitter{
		Emitter: scribe.NewEmitter(output),
	}
}

func (l LogEmitter) ListProcesses(processes []packit.Process) {
	for _, p := range processes {
		l.Subprocess("%s: %s", p.Type, p.Command)
	}
}
