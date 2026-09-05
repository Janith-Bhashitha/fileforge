package pdfops

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// RotateProcessor turns every page (or Options["pages"] worth of them) by
// Options["angle"] degrees — 90, 180 or 270, defaulting to 90.
type RotateProcessor struct{}

func (RotateProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	angle := 90
	if v := req.Options["angle"]; v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || (parsed != 90 && parsed != 180 && parsed != 270 && parsed != -90) {
			return convert.ConversionResult{}, fmt.Errorf("angle must be 90, 180 or 270")
		}
		angle = parsed
	}

	outPath := outputPath(filepath.Dir(req.InputPath), "rotated", ".pdf")
	if err := pdfcpuapi.RotateFile(req.InputPath, outPath, angle, pageSelection(req.Options["pages"]), nil); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("rotate pdf: %w", err)
	}

	return convert.ConversionResult{OutputPath: outPath, MimeType: "application/pdf", Filename: "rotated.pdf"}, nil
}

// RemovePagesProcessor drops the pages named in Options["pages"] and keeps
// everything else.
type RemovePagesProcessor struct{}

func (RemovePagesProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	pages := pageSelection(req.Options["pages"])
	if len(pages) == 0 {
		return convert.ConversionResult{}, fmt.Errorf(`pages is required, e.g. "1,3,5-7"`)
	}

	outPath := outputPath(filepath.Dir(req.InputPath), "pages-removed", ".pdf")
	if err := pdfcpuapi.RemovePagesFile(req.InputPath, outPath, pages, nil); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("remove pages: %w", err)
	}

	return convert.ConversionResult{OutputPath: outPath, MimeType: "application/pdf", Filename: "pages-removed.pdf"}, nil
}

// ExtractPagesProcessor is the inverse of RemovePages — it keeps only the
// pages named in Options["pages"], as one PDF (pdfcpu's TrimFile), rather
// than one file per page, which is what pdf-split already covers.
type ExtractPagesProcessor struct{}

func (ExtractPagesProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	pages := pageSelection(req.Options["pages"])
	if len(pages) == 0 {
		return convert.ConversionResult{}, fmt.Errorf(`pages is required, e.g. "1,3,5-7"`)
	}

	outPath := outputPath(filepath.Dir(req.InputPath), "extracted", ".pdf")
	if err := pdfcpuapi.TrimFile(req.InputPath, outPath, pages, nil); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("extract pages: %w", err)
	}

	return convert.ConversionResult{OutputPath: outPath, MimeType: "application/pdf", Filename: "extracted.pdf"}, nil
}

// pageSelection turns a user-supplied "1,3,5-7" into the string slice
// pdfcpu expects. An empty string means "all pages", which pdfcpu
// represents as a nil selection.
func pageSelection(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
