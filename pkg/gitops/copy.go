package gitops

import (
	"io/fs"
)

type FileCopier interface {
	CopyFiles(src fs.FS, dst string, sendMsg func(string)) error
}
