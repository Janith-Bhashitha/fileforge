package pdfops

import (
	"context"
	"fmt"
	"path/filepath"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// ProtectProcessor password-protects a PDF. Options["password"] becomes the
// user password (needed to open the file); the owner password is set to the
// same value so the caller retains full rights over their own document
// without having to manage two secrets.
type ProtectProcessor struct{}

func (ProtectProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	password := req.Options["password"]
	if password == "" {
		return convert.ConversionResult{}, fmt.Errorf("password is required")
	}

	conf := model.NewDefaultConfiguration()
	conf.UserPW = password
	conf.OwnerPW = password

	outPath := outputPath(filepath.Dir(req.InputPath), "protected", ".pdf")
	if err := pdfcpuapi.EncryptFile(req.InputPath, outPath, conf); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("protect pdf: %w", err)
	}

	return convert.ConversionResult{OutputPath: outPath, MimeType: "application/pdf", Filename: "protected.pdf"}, nil
}

// UnlockProcessor removes password protection, given the correct password.
// A wrong password fails the conversion rather than producing a broken file.
type UnlockProcessor struct{}

func (UnlockProcessor) Process(_ context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	password := req.Options["password"]
	if password == "" {
		return convert.ConversionResult{}, fmt.Errorf("password is required")
	}

	conf := model.NewDefaultConfiguration()
	conf.UserPW = password
	conf.OwnerPW = password

	outPath := outputPath(filepath.Dir(req.InputPath), "unlocked", ".pdf")
	if err := pdfcpuapi.DecryptFile(req.InputPath, outPath, conf); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("unlock pdf: %w", err)
	}

	return convert.ConversionResult{OutputPath: outPath, MimeType: "application/pdf", Filename: "unlocked.pdf"}, nil
}
