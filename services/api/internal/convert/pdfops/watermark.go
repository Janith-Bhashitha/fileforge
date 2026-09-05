package pdfops

import (
	"context"
	"fmt"
	"path/filepath"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// WatermarkProcessor stamps Options["text"] diagonally across every page
// (or Options["pages"]). The look is fixed rather than exposed as a dozen
// knobs — pdfcpu's description syntax is powerful but not something to put
// in front of an end user; a sensible grey 45-degree stamp covers the
// actual use case.
type WatermarkProcessor struct{}

// Parameter names are spelled out in full: pdfcpu matches them by prefix and
// rejects an ambiguous one (e.g. "sc" matches both scalefactor and scriptname).
const watermarkStyle = "fontname:Helvetica, points:48, fillcolor:0.6 0.6 0.6, rotation:45, scalefactor:0.9 rel, opacity:0.4"

func (WatermarkProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	text := req.Options["text"]
	if text == "" {
		return convert.ConversionResult{}, fmt.Errorf("text is required")
	}

	outPath := outputPath(filepath.Dir(req.InputPath), "watermarked", ".pdf")
	err := pdfcpuapi.AddTextWatermarksFile(
		req.InputPath, outPath,
		pageSelection(req.Options["pages"]),
		true, // onTop: a stamp over the content, not a background watermark
		text, watermarkStyle, nil,
	)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("watermark pdf: %w", err)
	}

	return convert.ConversionResult{OutputPath: outPath, MimeType: "application/pdf", Filename: "watermarked.pdf"}, nil
}
