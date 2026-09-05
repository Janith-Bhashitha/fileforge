package pdfops

import (
	"context"
	"fmt"
	"path/filepath"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// MergeProcessor merges multiple input PDFs (paths joined by ';' in
// Options["inputs"]) into one output PDF.
type MergeProcessor struct{}

func (MergeProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	inputs := splitPaths(req.Options["inputs"])
	if len(inputs) == 0 {
		inputs = []string{req.InputPath}
	}

	outPath := outputPath(filepath.Dir(req.InputPath), "merged", ".pdf")

	if err := pdfcpuapi.MergeCreateFile(inputs, outPath, false, nil); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("merge pdf: %w", err)
	}

	return convert.ConversionResult{
		OutputPath: outPath,
		MimeType:   "application/pdf",
		Filename:   "merged.pdf",
	}, nil
}
