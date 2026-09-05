package validate

import (
	"fmt"

	"github.com/gabriel-vasile/mimetype"
)

const MaxUploadSize = 50 * 1024 * 1024 // 50MB

var allowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"application/pdf": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"text/plain": true,
}

// DetectAndValidate sniffs the real content type from the file bytes (not the
// filename/extension) and rejects anything outside the supported set.
func DetectAndValidate(data []byte) (mimeType string, err error) {
	mt := mimetype.Detect(data)
	detected := mt.String()

	for allowed := range allowedMimeTypes {
		if mt.Is(allowed) {
			return allowed, nil
		}
	}

	return "", fmt.Errorf("unsupported file type: %s", detected)
}

// DetectFile applies the same rule to a file on disk. It exists for the
// presigned-upload path, where the bytes never passed through the API and
// so have to be sniffed after the fact rather than in memory.
func DetectFile(path string) (mimeType string, err error) {
	mt, err := mimetype.DetectFile(path)
	if err != nil {
		return "", fmt.Errorf("read file for type detection: %w", err)
	}

	for allowed := range allowedMimeTypes {
		if mt.Is(allowed) {
			return allowed, nil
		}
	}

	return "", fmt.Errorf("unsupported file type: %s", mt.String())
}

func ValidateSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("file is empty")
	}
	if size > MaxUploadSize {
		return fmt.Errorf("file exceeds maximum size of %d bytes", MaxUploadSize)
	}
	return nil
}
