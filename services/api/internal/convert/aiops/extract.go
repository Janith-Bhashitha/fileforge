package aiops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

// ExtractText pulls the text out of a document by whichever route suits it.
//
// For a PDF it tries pdftotext first: a PDF with a real text layer gives back
// exact characters instantly, where OCR would guess at them from pixels. Only
// when that yields nothing - a scan, or an image-only export - does it fall
// back to rasterise-and-OCR, which is orders of magnitude slower.
func ExtractText(ctx context.Context, path, lang, workDir string) (text string, ocrUsed bool, err error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".txt", ".md", ".csv":
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", false, fmt.Errorf("read text file: %w", readErr)
		}
		return string(b), false, nil

	case ".pdf":
		if extracted, tryErr := pdfToText(ctx, path); tryErr == nil && hasRealText(extracted) {
			return extracted, false, nil
		}
		// No usable text layer - this is a scanned document.
		ocrText, ocrErr := ocrPDF(ctx, path, lang, workDir)
		if ocrErr != nil {
			return "", true, ocrErr
		}
		return ocrText, true, nil

	default:
		// Images have no text layer by definition.
		ocrText, ocrErr := ocrImage(ctx, path, lang)
		if ocrErr != nil {
			return "", true, ocrErr
		}
		return ocrText, true, nil
	}
}

func pdfToText(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", path, "-")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w: %s", err, stderrOf(err))
	}
	return string(out), nil
}

// hasRealText decides whether an extraction is worth trusting. A scanned PDF
// often still returns a handful of stray characters, so a non-empty result is
// not on its own evidence of a text layer.
func hasRealText(s string) bool {
	letters := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if letters >= 50 {
				return true
			}
		}
	}
	return false
}
