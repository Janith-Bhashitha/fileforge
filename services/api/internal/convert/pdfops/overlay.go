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

	// Decoders for DecodeConfig, which reads intrinsic image dimensions.
	_ "image/jpeg"
	_ "image/png"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// OverlayProcessor composites text and images onto an existing PDF, leaving
// the original page content untouched. It does not edit existing text -
// neither pdfcpu nor fpdf can do that.
//
// Each element is a separate pdfcpu stamp pass, chained through temp files.
type OverlayProcessor struct{}

const (
	maxElements   = 50
	maxImageBytes = 2 << 20
)

// OverlayElement is one placed item. X/Y are PDF points from the bottom-left
// of the page, addressing the element's own bottom-left corner.
type OverlayElement struct {
	Type string  `json:"type"` // "text" or "image"
	Page int     `json:"page"` // 1-based
	X    float64 `json:"x"`
	Y    float64 `json:"y"`

	// Text elements.
	Text  string  `json:"text,omitempty"`
	Size  float64 `json:"size,omitempty"`  // points, default 12
	Color string  `json:"color,omitempty"` // "#rrggbb" or "r g b", default black

	// Image elements. Data is a data: URI or bare base64.
	Data   string  `json:"data,omitempty"`
	Width  float64 `json:"width,omitempty"`  // points; height follows the aspect ratio
	Height float64 `json:"height,omitempty"` // unused

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

	// Intermediate PDFs and decoded images; only the final PDF survives.
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

		// Disposable once the next pass exists, but never the caller's input.
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

// Parameter names are spelled out in full: pdfcpu matches by prefix and
// rejects an ambiguous one ("sc" matches both scalefactor and scriptname).
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
		// Default scaling is relative to the page, which would resize text
		// away from the requested point size.
		"scalefactor:1 absolute",
	}
	return strings.Join(parts, ", ")
}

func imageDesc(el OverlayElement, imgPath string) (string, error) {
	// pdfcpu sizes an image stamp from its pixel dimensions taken as points,
	// so a width in points becomes a scale factor against the intrinsic
	// width. Height follows the aspect ratio rather than distorting.
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

// decodeImage writes a data: URI (or bare base64) to disk, which is what
// pdfcpu's image stamping takes.
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

func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
