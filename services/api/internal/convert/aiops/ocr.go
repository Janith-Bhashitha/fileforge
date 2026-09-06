// Package aiops holds the document-understanding operations: OCR, text
// extraction and the analysis built on top of them.
//
// Tesseract and poppler are shelled out, the same pattern office.go uses for
// LibreOffice - there is no maintained pure-Go OCR engine, and reimplementing
// one is not the job. Everything here runs locally: no API key, no per-page
// cost, nothing leaves the machine.
package aiops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// OCRProcessor extracts text from an image, or from a PDF by rasterising its
// pages first. Options["language"] selects the Tesseract language pack
// (default "eng"); only packs installed in the image are available.
type OCRProcessor struct{}

func (OCRProcessor) Process(ctx context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	lang := req.Options["language"]
	if lang == "" {
		lang = "eng"
	}
	// The language is passed to a subprocess, so it must not be able to carry
	// arbitrary arguments or shell metacharacters.
	if !isSafeLangCode(lang) {
		return convert.ConversionResult{}, fmt.Errorf("invalid language code: %q", lang)
	}

	dir := filepath.Dir(req.InputPath)

	text, err := extractTextByOCR(ctx, req.InputPath, lang, dir)
	if err != nil {
		return convert.ConversionResult{}, err
	}

	outPath := outputPath(dir, "ocr", ".txt")
	if err := os.WriteFile(outPath, []byte(text), 0o644); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("write ocr output: %w", err)
	}

	return convert.ConversionResult{
		OutputPath: outPath,
		MimeType:   "text/plain",
		Filename:   "ocr.txt",
	}, nil
}

// extractTextByOCR handles both shapes of input. A PDF is rasterised to PNGs
// first because Tesseract reads images, not PDFs; each page is OCR'd in turn
// and the results concatenated in page order.
func extractTextByOCR(ctx context.Context, inputPath, lang, workDir string) (string, error) {
	if strings.EqualFold(filepath.Ext(inputPath), ".pdf") {
		return ocrPDF(ctx, inputPath, lang, workDir)
	}
	return ocrImage(ctx, inputPath, lang)
}

func ocrImage(ctx context.Context, imagePath, lang string) (string, error) {
	// "stdout" tells Tesseract to write the text to stdout instead of a file.
	cmd := exec.CommandContext(ctx, "tesseract", imagePath, "stdout", "-l", lang)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tesseract: %w: %s", err, stderrOf(err))
	}
	return string(out), nil
}

func ocrPDF(ctx context.Context, pdfPath, lang, workDir string) (string, error) {
	rasterDir, err := os.MkdirTemp(workDir, "ocr-raster-*")
	if err != nil {
		return "", fmt.Errorf("create raster dir: %w", err)
	}
	defer os.RemoveAll(rasterDir)

	// 300 DPI is the usual floor for reliable OCR; below roughly 200 the
	// character shapes degrade enough that accuracy drops sharply.
	prefix := filepath.Join(rasterDir, "page")
	cmd := exec.CommandContext(ctx, "pdftoppm", "-png", "-r", "300", pdfPath, prefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("rasterise pdf: %w: %s", err, out)
	}

	entries, err := os.ReadDir(rasterDir)
	if err != nil {
		return "", fmt.Errorf("read raster dir: %w", err)
	}

	var pages []string
	for _, e := range entries {
		if !e.IsDir() {
			pages = append(pages, filepath.Join(rasterDir, e.Name()))
		}
	}
	if len(pages) == 0 {
		return "", fmt.Errorf("pdf produced no pages to read")
	}
	// pdftoppm names files page-01, page-02 ... so lexical order is page order.
	sort.Strings(pages)

	var sb strings.Builder
	for i, page := range pages {
		text, err := ocrImage(ctx, page, lang)
		if err != nil {
			return "", fmt.Errorf("page %d: %w", i+1, err)
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		fmt.Fprintf(&sb, "--- Page %d ---\n%s", i+1, text)
	}

	return sb.String(), nil
}

// isSafeLangCode allows only what Tesseract language packs actually look
// like: lowercase letters, and "+" for combining packs ("eng+sin").
func isSafeLangCode(lang string) bool {
	if len(lang) == 0 || len(lang) > 32 {
		return false
	}
	for _, r := range lang {
		if (r < 'a' || r > 'z') && r != '+' {
			return false
		}
	}
	return true
}

func stderrOf(err error) string {
	var ee *exec.ExitError
	if ok := asExitError(err, &ee); ok {
		return string(ee.Stderr)
	}
	return ""
}
