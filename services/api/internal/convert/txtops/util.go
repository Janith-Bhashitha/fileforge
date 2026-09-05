package txtops

import (
	"path/filepath"

	"github.com/google/uuid"
)

func outputPath(dir, prefix, ext string) string {
	return filepath.Join(dir, prefix+"-"+uuid.New().String()+ext)
}
