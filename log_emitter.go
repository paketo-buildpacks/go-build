package gobuild

import (
	"io"

	"github.com/paketo-buildpacks/packit/scribe"
)

type LogEmitter struct {
	scribe.Logger
}

func NewLogEmitter(output io.Writer) LogEmitter {
	return LogEmitter{
		Logger: scribe.NewLogger(output),
	}
}
