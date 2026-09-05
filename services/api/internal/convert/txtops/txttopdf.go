package txtops

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-pdf/fpdf"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// TxtToPDFProcessor renders plain text into a paginated PDF. Unlike the
// office/image processors, there's no external tool to shell out to here —
// laying text onto a page is simple enough that a small Go PDF library
// (fpdf) does the whole job directly.
type TxtToPDFProcessor struct{}

func (TxtToPDFProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	in, err := os.Open(req.InputPath)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("open txt file: %w", err)
	}
	defer in.Close()

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()
	pdf.SetFont("Arial", "", 11)

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			pdf.Ln(5)
			continue
		}
		pdf.MultiCell(0, 6, line, "", "L", false)
	}
	if err := scanner.Err(); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("read txt file: %w", err)
	}

	if pdf.Error() != nil {
		return convert.ConversionResult{}, fmt.Errorf("render pdf: %w", pdf.Error())
	}

	outPath := outputPath(filepath.Dir(req.InputPath), "txt-to-pdf", ".pdf")
	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("write pdf: %w", err)
	}

	return convert.ConversionResult{
		OutputPath: outPath,
		MimeType:   "application/pdf",
		Filename:   "converted.pdf",
	}, nil
}
