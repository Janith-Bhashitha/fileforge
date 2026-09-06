// Package convertsetup wires up the operation registry. It is separate from
// internal/convert because it imports every processor subpackage, and those
// already import internal/convert - keeping it here avoids an import cycle.
package convertsetup

import (
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/aiclient"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/aiops"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/imageops"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/office"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/pdfops"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/txtops"
)

// BuildRegistry registers every operation under its "name:version" key. The
// API and every worker call this, so one place lists what FileForge can do.
//
// geminiClient is passed in because it is the only processor with real
// configuration behind it. An empty-key client is expected when AI is
// unconfigured; ai-analyze then fails per-request rather than at startup.
func BuildRegistry(geminiClient *aiclient.Client) *convert.Registry {
	reg := convert.NewRegistry()
	reg.Register("image-to-pdf", "v1", imageops.ImageToPDFProcessor{})
	reg.Register("pdf-to-image", "v1", imageops.PDFToImageProcessor{})
	reg.Register("image-convert", "v1", imageops.ImageConvertProcessor{})
	reg.Register("image-resize", "v1", imageops.ImageResizeProcessor{})
	reg.Register("docx-to-pdf", "v1", office.OfficeToPDFProcessor{})
	reg.Register("pptx-to-pdf", "v1", office.OfficeToPDFProcessor{})
	reg.Register("xlsx-to-pdf", "v1", office.OfficeToPDFProcessor{})
	reg.Register("txt-to-pdf", "v1", txtops.TxtToPDFProcessor{})
	reg.Register("pdf-merge", "v1", pdfops.MergeProcessor{})
	reg.Register("pdf-split", "v1", pdfops.SplitProcessor{})
	reg.Register("pdf-compress", "v1", pdfops.CompressProcessor{})
	reg.Register("pdf-rotate", "v1", pdfops.RotateProcessor{})
	reg.Register("pdf-remove-pages", "v1", pdfops.RemovePagesProcessor{})
	reg.Register("pdf-extract-pages", "v1", pdfops.ExtractPagesProcessor{})
	reg.Register("pdf-watermark", "v1", pdfops.WatermarkProcessor{})
	reg.Register("pdf-overlay", "v1", pdfops.OverlayProcessor{})
	reg.Register("pdf-protect", "v1", pdfops.ProtectProcessor{})
	reg.Register("pdf-unlock", "v1", pdfops.UnlockProcessor{})
	reg.Register("ocr", "v1", aiops.OCRProcessor{})
	reg.Register("document-insights", "v1", aiops.InsightsProcessor{})
	reg.Register("ai-analyze", "v1", aiops.AIAnalyzeProcessor{Client: geminiClient})
	return reg
}
