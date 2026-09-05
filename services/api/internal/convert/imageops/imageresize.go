package imageops

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/image/draw"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

const defaultMaxDimension = 1200

// ImageResizeProcessor scales an image down so its longer side matches
// Options["max_width"] (default 1200px), preserving aspect ratio. Images
// already within that size pass through unresized. Output keeps the
// original format (JPEG stays JPEG, PNG stays PNG).
type ImageResizeProcessor struct{}

func (ImageResizeProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	in, err := os.Open(req.InputPath)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("open image: %w", err)
	}
	defer in.Close()

	img, decodedFormat, err := image.Decode(in)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("decode image: %w", err)
	}

	maxDim := defaultMaxDimension
	if v := req.Options["max_width"]; v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxDim = parsed
		}
	}

	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW > maxDim || srcH > maxDim {
		var dstW, dstH int
		if srcW >= srcH {
			dstW = maxDim
			dstH = srcH * maxDim / srcW
		} else {
			dstH = maxDim
			dstW = srcW * maxDim / srcH
		}
		dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		img = dst
	}

	ext, mimeType := extensionFor(decodedFormat)

	outPath := outputPath(filepath.Dir(req.InputPath), "resized", ext)
	out, err := os.Create(outPath)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	if err := encodeImage(out, img, ext); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("encode image: %w", err)
	}

	return convert.ConversionResult{OutputPath: outPath, MimeType: mimeType, Filename: "resized" + ext}, nil
}
