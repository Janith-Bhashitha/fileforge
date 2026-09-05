package pdfops

import (
	"path/filepath"

	"github.com/google/uuid"
)

func outputPath(dir, prefix, ext string) string {
	return filepath.Join(dir, prefix+"-"+uuid.New().String()+ext)
}

func splitPaths(joined string) []string {
	if joined == "" {
		return nil
	}
	var out []string
	start := 0
	for i, c := range joined {
		if c == ';' {
			out = append(out, joined[start:i])
			start = i + 1
		}
	}
	out = append(out, joined[start:])
	return out
}
