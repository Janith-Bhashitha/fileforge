package imageops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// PDFToImageProcessor rasterizes each page of a PDF into an image via
// poppler's pdftoppm (shelled out, same isolation pattern as the LibreOffice
// office processor — no mature pure-Go PDF rasterizer exists).
type PDFToImageProcessor struct{}

func (PDFToImageProcessor) Process(ctx context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	format := req.Options["format"]
	if format != "png" {
		format = "jpeg"
	}

	dir := filepath.Dir(req.InputPath)
	rasterDir, err := os.MkdirTemp(dir, "raster-*")
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("create raster dir: %w", err)
	}
	defer os.RemoveAll(rasterDir)

	prefix := filepath.Join(rasterDir, "page")
	args := []string{"-" + format, "-r", "150", req.InputPath, prefix}

	cmd := exec.CommandContext(ctx, "pdftoppm", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("pdftoppm: %w: %s", err, out)
	}

	entries, err := os.ReadDir(rasterDir)
	if err != nil || len(entries) == 0 {
		return convert.ConversionResult{}, fmt.Errorf("pdftoppm produced no output")
	}

	ext := ".jpg"
	mimeType := "image/jpeg"
	if format == "png" {
		ext = ".png"
		mimeType = "image/png"
	}

	if len(entries) == 1 {
		outPath := outputPath(dir, "page", ext)
		if err := os.Rename(filepath.Join(rasterDir, entries[0].Name()), outPath); err != nil {
			return convert.ConversionResult{}, fmt.Errorf("move rasterized page: %w", err)
		}
		return convert.ConversionResult{OutputPath: outPath, MimeType: mimeType, Filename: "page" + ext}, nil
	}

	zipPath := outputPath(dir, "pages", ".zip")
	if err := convert.ZipDir(rasterDir, zipPath); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("zip rasterized pages: %w", err)
	}
	return convert.ConversionResult{OutputPath: zipPath, MimeType: "application/zip", Filename: "pages.zip"}, nil
}
