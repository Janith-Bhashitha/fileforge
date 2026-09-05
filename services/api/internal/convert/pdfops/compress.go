package pdfops

import (
	"context"
	"fmt"
	"path/filepath"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// CompressProcessor runs pdfcpu's optimizer to reduce file size.
type CompressProcessor struct{}

func (CompressProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	outPath := outputPath(filepath.Dir(req.InputPath), "compressed", ".pdf")

	if err := pdfcpuapi.OptimizeFile(req.InputPath, outPath, nil); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("compress pdf: %w", err)
	}

	return convert.ConversionResult{
		OutputPath: outPath,
		MimeType:   "application/pdf",
		Filename:   "compressed.pdf",
	}, nil
}
