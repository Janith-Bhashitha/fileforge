package pdfops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// SplitProcessor splits a PDF into one file per page and returns a ZIP of
// the results (Phase 2 keeps this synchronous and single-output; a real
// multi-output batch flow arrives in Phase 4).
type SplitProcessor struct{}

func (SplitProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	dir := filepath.Dir(req.InputPath)
	splitDir, err := os.MkdirTemp(dir, "split-*")
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("create split dir: %w", err)
	}
	defer os.RemoveAll(splitDir)

	if err := pdfcpuapi.SplitFile(req.InputPath, splitDir, 1, nil); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("split pdf: %w", err)
	}

	zipPath := outputPath(dir, "split", ".zip")
	if err := convert.ZipDir(splitDir, zipPath); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("zip split output: %w", err)
	}

	return convert.ConversionResult{
		OutputPath: zipPath,
		MimeType:   "application/zip",
		Filename:   "split.zip",
	}, nil
}
