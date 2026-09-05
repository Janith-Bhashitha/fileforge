// Package convertsetup wires up the operation registry. It exists as its
// own package (rather than living in internal/convert itself) because it
// has to import every processor subpackage (imageops, pdfops, office,
// txtops), and those subpackages already import internal/convert for the
// shared Processor/Registry types - putting this here instead of inside
// internal/convert avoids a Go import cycle.
package convertsetup

import (
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/imageops"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/office"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/pdfops"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/txtops"
)

// BuildRegistry registers every known operation under its "name:version"
// key. The API (for its synchronous /convert endpoint) and every worker
// (for its async job processing) call this same function, so there's one
// place that ever lists what FileForge can do.
func BuildRegistry() *convert.Registry {
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
	reg.Register("pdf-protect", "v1", pdfops.ProtectProcessor{})
	reg.Register("pdf-unlock", "v1", pdfops.UnlockProcessor{})
	return reg
}
