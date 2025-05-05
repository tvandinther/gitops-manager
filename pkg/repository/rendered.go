package repository

import (
	"io/fs"
	"os"
)

type Rendered struct {
}

func NewRendered(path string) *Rendered {
	fs := os.DirFS(path)

	rendered := &Rendered{}

	rendered.Init(fs)

	return rendered
}

func (r *Rendered) Init(fs fs.FS) error {
	panic("Not implemented")
}
