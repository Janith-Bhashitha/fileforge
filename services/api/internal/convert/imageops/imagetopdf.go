package imageops

import (
	"context"
	"fmt"
	"path/filepath"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// ImageToPDFProcessor wraps a single image (JPG/PNG) into a one-page PDF.
type ImageToPDFProcessor struct{}

func (ImageToPDFProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	outPath := outputPath(filepath.Dir(req.InputPath), "image-to-pdf", ".pdf")

	imp := pdfcpu.DefaultImportConfig()
	if err := pdfcpuapi.ImportImagesFile([]string{req.InputPath}, outPath, imp, nil); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("image to pdf: %w", err)
	}

	return convert.ConversionResult{
		OutputPath: outPath,
		MimeType:   "application/pdf",
		Filename:   "converted.pdf",
	}, nil
}
