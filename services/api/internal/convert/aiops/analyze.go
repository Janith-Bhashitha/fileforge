package aiops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/aiclient"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// AIAnalyzeProcessor is the one operation in the whole registry that costs
// money (against Google's free quota) and leaves the machine. Every other
// processor is a value type constructed with `Processor{}`; this one holds
// a client because there is a real API key and HTTP endpoint behind it.
type AIAnalyzeProcessor struct {
	Client *aiclient.Client
}

func (p AIAnalyzeProcessor) Process(ctx context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	if !p.Client.Configured() {
		return convert.ConversionResult{}, aiclient.ErrNotConfigured
	}

	dir := filepath.Dir(req.InputPath)
	lang := req.Options["language"]
	if lang == "" {
		lang = "eng"
	}

	text, ocrUsed, err := ExtractText(ctx, req.InputPath, lang, dir)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("extract text: %w", err)
	}
	if len(text) < 20 {
		return convert.ConversionResult{}, fmt.Errorf("document has too little extractable text to analyze")
	}

	analysis, err := p.Client.Analyze(ctx, text, req.Options["summary_length"])
	if err != nil {
		return convert.ConversionResult{}, err
	}

	result := struct {
		*aiclient.Analysis
		OCRUsed bool `json:"ocr_used"`
	}{analysis, ocrUsed}

	outPath := outputPath(dir, "ai-analysis", ".json")
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("marshal analysis: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("write analysis: %w", err)
	}

	return convert.ConversionResult{
		OutputPath: outPath,
		MimeType:   "application/json",
		Filename:   "ai-analysis.json",
	}, nil
}
