package pdfops

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	// Registers the decoders DecodeConfig needs to read intrinsic image
	// dimensions. Blank imports: only the side effect is wanted.
	_ "image/jpeg"
	_ "image/png"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// OverlayProcessor stamps user-placed elements - typed text, and images such
// as a drawn signature - onto an existing PDF. This is deliberately overlay
// editing, not content editing: the original page content is left completely
// untouched underneath, and elements are composited on top.
//
// Editing the *existing* text of a PDF is a genuinely different problem
// (extracting text runs, re-flowing paragraphs, subsetting fonts) that
// neither pdfcpu nor fpdf can do, so it isn't attempted here. Everything
// this supports - add text, sign, stamp, redact - is overlay work, which is
// also what the great majority of "edit this PDF" actually means in practice.
//
// Each element is applied as its own pdfcpu stamp pass, chaining through
// temp files. That costs one rewrite per element rather than building a
// composite overlay document, which is the right trade for the handful of
// elements a person places by hand, and it reuses the same well-trodden
// stamping path as WatermarkProcessor instead of hand-rolling PDF content
// streams.
type OverlayProcessor struct{}

// maxElements bounds the per-element rewrite cost, and maxImageBytes bounds
// what arrives inline in Options. A drawn signature PNG is a few KB; a
// megabyte of base64 in a queue message is a mistake, not a signature.
const (
	maxElements   = 50
	maxImageBytes = 2 << 20 // 2 MiB decoded
)

// OverlayElement is one placed item.
//
// X/Y are PDF points measured from the bottom-left of the page, and address
// the element's own bottom-left corner - pdfcpu's "position:bl, offset:x y"
// anchors a stamp's bounding box exactly that way, so the frontend's
// coordinates survive to the page with no reinterpretation in between.
type OverlayElement struct {
	Type string  `json:"type"` // "text" or "image"
	Page int     `json:"page"` // 1-based
	X    float64 `json:"x"`
	Y    float64 `json:"y"`

	// Text elements.
	Text  string  `json:"text,omitempty"`
	Size  float64 `json:"size,omitempty"`  // font size in points, default 12
	Color string  `json:"color,omitempty"` // "#rrggbb" or "r g b", default black

	// Image elements. Data is a data: URI or bare base64 - a drawn signature
	// arrives straight off a canvas as one.
	Data   string  `json:"data,omitempty"`
	Width  float64 `json:"width,omitempty"`  // desired width in points
	Height float64 `json:"height,omitempty"` // reserved; aspect ratio is preserved from Width

	Rotation float64  `json:"rotation,omitempty"`
	Opacity  *float64 `json:"opacity,omitempty"` // pointer so 0 stays meaningful
}

func (OverlayProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	raw := req.Options["elements"]
	if strings.TrimSpace(raw) == "" {
		return convert.ConversionResult{}, fmt.Errorf("elements is required")
	}

	var elements []OverlayElement
	if err := json.Unmarshal([]byte(raw), &elements); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("elements must be a JSON array: %w", err)
	}
	if len(elements) == 0 {
		return convert.ConversionResult{}, fmt.Errorf("at least one element is required")
	}
	if len(elements) > maxElements {
		return convert.ConversionResult{}, fmt.Errorf("too many elements: %d (max %d)", len(elements), maxElements)
	}

	dir := filepath.Dir(req.InputPath)

	// Scratch files created along the way - intermediate PDFs and decoded
	// images - all get removed on the way out. Only the final PDF survives,
	// and it is the one path the caller is handed.
	var scratch []string
	defer func() {
		for _, p := range scratch {
			os.Remove(p)
		}
	}()

	current := req.InputPath
	for i, el := range elements {
		outPath := outputPath(dir, "edited", ".pdf")

		if err := applyElement(current, outPath, el, dir, &scratch); err != nil {
			os.Remove(outPath)
			return convert.ConversionResult{}, fmt.Errorf("element %d: %w", i, err)
		}

		// The input to this pass is disposable once the next one exists,
		// but never the caller's original input file.
		if current != req.InputPath {
			scratch = append(scratch, current)
		}
		current = outPath
	}

	return convert.ConversionResult{OutputPath: current, MimeType: "application/pdf", Filename: "edited.pdf"}, nil
}

func applyElement(inPath, outPath string, el OverlayElement, dir string, scratch *[]string) error {
	if el.Page < 1 {
		return fmt.Errorf("page must be 1 or greater, got %d", el.Page)
	}
	pages := []string{strconv.Itoa(el.Page)}

	switch el.Type {
	case "text":
		if strings.TrimSpace(el.Text) == "" {
			return fmt.Errorf("text is required for a text element")
		}
		desc := textDesc(el)
		if err := pdfcpuapi.AddTextWatermarksFile(inPath, outPath, pages, true, el.Text, desc, nil); err != nil {
			return fmt.Errorf("stamp text: %w", err)
		}
		return nil

	case "image":
		imgPath, err := decodeImage(el.Data, dir)
		if err != nil {
			return err
		}
		*scratch = append(*scratch, imgPath)

		desc, err := imageDesc(el, imgPath)
		if err != nil {
			return err
		}
		if err := pdfcpuapi.AddImageWatermarksFile(inPath, outPath, pages, true, imgPath, desc, nil); err != nil {
			return fmt.Errorf("stamp image: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown element type %q (want \"text\" or \"image\")", el.Type)
	}
}

// Parameter names are spelled out in full throughout: pdfcpu matches them by
// prefix and rejects an ambiguous one (e.g. "sc" matches both scalefactor
// and scriptname).
func textDesc(el OverlayElement) string {
	size := el.Size
	if size <= 0 {
		size = 12
	}
	color := strings.TrimSpace(el.Color)
	if color == "" {
		color = "#000000"
	}

	parts := []string{
		"fontname:Helvetica",
		"points:" + trimFloat(size),
		"fillcolor:" + color,
		"position:bl",
		"offset:" + trimFloat(el.X) + " " + trimFloat(el.Y),
		"rotation:" + trimFloat(el.Rotation),
		"opacity:" + trimFloat(opacityOf(el)),
		// A stamp is scaled relative to the page by default, which would
		// silently resize text away from the requested point size.
		"scalefactor:1 absolute",
	}
	return strings.Join(parts, ", ")
}

func imageDesc(el OverlayElement, imgPath string) (string, error) {
	// pdfcpu sizes an image stamp from its pixel dimensions taken as points,
	// so a requested width in points becomes a scale factor against the
	// intrinsic width. Height follows from the aspect ratio - letting both
	// be set independently would distort a signature, which is never wanted.
	scale := 1.0
	if el.Width > 0 {
		f, err := os.Open(imgPath)
		if err != nil {
			return "", fmt.Errorf("open decoded image: %w", err)
		}
		cfg, _, err := image.DecodeConfig(f)
		f.Close()
		if err != nil {
			return "", fmt.Errorf("read image dimensions: %w", err)
		}
		if cfg.Width == 0 {
			return "", fmt.Errorf("image has zero width")
		}
		scale = el.Width / float64(cfg.Width)
	}

	parts := []string{
		"position:bl",
		"offset:" + trimFloat(el.X) + " " + trimFloat(el.Y),
		"rotation:" + trimFloat(el.Rotation),
		"opacity:" + trimFloat(opacityOf(el)),
		"scalefactor:" + trimFloat(scale) + " absolute",
	}
	return strings.Join(parts, ", "), nil
}

func opacityOf(el OverlayElement) float64 {
	if el.Opacity == nil {
		return 1
	}
	if *el.Opacity < 0 {
		return 0
	}
	if *el.Opacity > 1 {
		return 1
	}
	return *el.Opacity
}

// decodeImage turns a data: URI (or bare base64) into a real file on disk,
// which is what pdfcpu's image stamping takes.
func decodeImage(data, dir string) (string, error) {
	if strings.TrimSpace(data) == "" {
		return "", fmt.Errorf("data is required for an image element")
	}

	ext := ".png"
	payload := data
	if strings.HasPrefix(data, "data:") {
		comma := strings.Index(data, ",")
		if comma < 0 {
			return "", fmt.Errorf("malformed data URI")
		}
		header := data[:comma]
		payload = data[comma+1:]
		if !strings.Contains(header, ";base64") {
			return "", fmt.Errorf("data URI must be base64-encoded")
		}
		switch {
		case strings.Contains(header, "image/png"):
			ext = ".png"
		case strings.Contains(header, "image/jpeg"), strings.Contains(header, "image/jpg"):
			ext = ".jpg"
		default:
			return "", fmt.Errorf("image must be PNG or JPEG")
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload))
	if err != nil {
		return "", fmt.Errorf("decode image data: %w", err)
	}
	if len(decoded) == 0 {
		return "", fmt.Errorf("image data is empty")
	}
	if len(decoded) > maxImageBytes {
		return "", fmt.Errorf("image is too large: %d bytes (max %d)", len(decoded), maxImageBytes)
	}

	path := outputPath(dir, "element", ext)
	if err := os.WriteFile(path, decoded, 0o644); err != nil {
		return "", fmt.Errorf("write decoded image: %w", err)
	}
	return path, nil
}

// trimFloat keeps the description string readable ("72" not "72.000000") -
// pdfcpu parses either, but these strings end up in error messages and logs.
func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
