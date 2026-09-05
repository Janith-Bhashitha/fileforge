package office

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// OfficeToPDFProcessor shells out to LibreOffice headless, the same pattern
// the master plan uses for anything without a mature native Go equivalent —
// no maintained pure-Go Office-to-PDF renderer exists. LibreOffice
// auto-detects the input format, so this single processor handles DOCX,
// PPTX, and XLSX alike (Writer, Impress, and Calc are all installed in the
// image) rather than needing one implementation per format.
type OfficeToPDFProcessor struct{}

func (OfficeToPDFProcessor) Process(ctx context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	dir := filepath.Dir(req.InputPath)
	outDir, err := os.MkdirTemp(dir, "office-*")
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("create office out dir: %w", err)
	}
	defer os.RemoveAll(outDir)

	// --env vars isolate each conversion's LibreOffice user profile so
	// concurrent conversions in the same container don't collide.
	profileDir, err := os.MkdirTemp(dir, "lo-profile-*")
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("create libreoffice profile dir: %w", err)
	}
	defer os.RemoveAll(profileDir)

	args := []string{
		"--headless",
		"--norestore",
		"--convert-to", "pdf",
		"--outdir", outDir,
		fmt.Sprintf("-env:UserInstallation=file://%s", filepath.ToSlash(profileDir)),
		req.InputPath,
	}

	cmd := exec.CommandContext(ctx, "soffice", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("libreoffice convert: %w: %s", err, out)
	}

	base := strings.TrimSuffix(filepath.Base(req.InputPath), filepath.Ext(req.InputPath))
	producedPath := filepath.Join(outDir, base+".pdf")
	if _, err := os.Stat(producedPath); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("libreoffice did not produce expected output: %w", err)
	}

	outPath := filepath.Join(dir, "office-to-pdf-"+base+".pdf")
	if err := os.Rename(producedPath, outPath); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("move converted file: %w", err)
	}

	return convert.ConversionResult{
		OutputPath: outPath,
		MimeType:   "application/pdf",
		Filename:   base + ".pdf",
	}, nil
}
