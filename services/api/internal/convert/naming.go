package convert

import (
	"path/filepath"
	"strings"
)

// DisplayFilename builds a human-readable output name from the original
// upload, e.g. "report.docx" + "docx-to-pdf" + ".pdf" -> "report-docx-to-pdf.pdf".
// Shared by the synchronous /convert handler and every async worker so a
// converted file never surfaces its internal (often UUID-based) storage
// path as its name.
func DisplayFilename(originalName, operation, ext string) string {
	base := strings.TrimSuffix(originalName, filepath.Ext(originalName))
	return base + "-" + operation + ext
}
