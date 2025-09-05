package copier

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

type Subpath struct {
	// Required to contain destructive actions. E.g. "manifests"
	ManifestDirectoryName string
	Path                  string
}

func (c *Subpath) CopyFiles(src fs.FS, dst string, sendMsg func(string)) error {
	finalDst := filepath.Join(dst, c.ManifestDirectoryName, c.Path)

	slog.Debug("copying files", "src", src, "dst", finalDst)
	sendMsg(fmt.Sprintf("copying files to %s", c.Path))
	os.RemoveAll(finalDst) // Maintains a declarative workflow
	err := os.CopyFS(finalDst, src)
	if err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	return nil
}
