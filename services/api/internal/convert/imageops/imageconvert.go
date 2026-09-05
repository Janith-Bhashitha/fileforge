package imageops

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// ImageConvertProcessor converts between raster image formats (JPG/PNG)
// without resizing — decode with the standard library, encode into
// whichever format Options["format"] requests.
type ImageConvertProcessor struct{}

func (ImageConvertProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	in, err := os.Open(req.InputPath)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("open image: %w", err)
	}
	defer in.Close()

	img, _, err := image.Decode(in)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("decode image: %w", err)
	}

	ext, mimeType := extensionFor(req.Options["format"])

	outPath := outputPath(filepath.Dir(req.InputPath), "converted", ext)
	out, err := os.Create(outPath)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	if err := encodeImage(out, img, ext); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("encode image: %w", err)
	}

	return convert.ConversionResult{OutputPath: outPath, MimeType: mimeType, Filename: "converted" + ext}, nil
}

func extensionFor(format string) (ext, mimeType string) {
	if format == "png" {
		return ".png", "image/png"
	}
	return ".jpg", "image/jpeg"
}

func encodeImage(out *os.File, img image.Image, ext string) error {
	if ext == ".png" {
		return png.Encode(out, img)
	}
	return jpeg.Encode(out, img, &jpeg.Options{Quality: 90})
}
