package aiops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/abadojack/whatlanggo"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
)

// InsightsProcessor computes real, checkable facts about a document - no
// model, no guessing. Everything here is either counted directly or looked
// up from the file's own metadata, which is what makes it safe to run for
// free with no rate limit: it costs CPU, not an API quota.
type InsightsProcessor struct{}

type Insights struct {
	WordCount      int               `json:"word_count"`
	CharacterCount int               `json:"character_count"`
	SentenceCount  int               `json:"sentence_count"`
	Language       string            `json:"language"`
	LanguageName   string            `json:"language_name"`
	Confident      bool              `json:"language_confident"`
	OCRUsed        bool              `json:"ocr_used"`
	TopKeywords    []string          `json:"top_keywords"`
	PDFMetadata    *PDFMetadata      `json:"pdf_metadata,omitempty"`
	Preview        string            `json:"text_preview"`
	Extra          map[string]string `json:"-"`
}

type PDFMetadata struct {
	PageCount    int    `json:"page_count"`
	Title        string `json:"title,omitempty"`
	Author       string `json:"author,omitempty"`
	Subject      string `json:"subject,omitempty"`
	Creator      string `json:"creator,omitempty"`
	CreationDate string `json:"creation_date,omitempty"`
}

func (InsightsProcessor) Process(ctx context.Context, req convert.ConversionRequest) (convert.ConversionResult, error) {
	dir := filepath.Dir(req.InputPath)

	lang := req.Options["language"]
	if lang == "" {
		lang = "eng"
	}

	text, ocrUsed, err := ExtractText(ctx, req.InputPath, lang, dir)
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("extract text: %w", err)
	}

	insights := analyzeText(text)
	insights.OCRUsed = ocrUsed

	if strings.EqualFold(filepath.Ext(req.InputPath), ".pdf") {
		if meta, err := pdfMetadata(req.InputPath); err == nil {
			insights.PDFMetadata = meta
		}
		// A missing/unreadable metadata block isn't fatal - the text-based
		// insights are still valid and worth returning on their own.
	}

	outPath := outputPath(dir, "insights", ".json")
	data, err := json.MarshalIndent(insights, "", "  ")
	if err != nil {
		return convert.ConversionResult{}, fmt.Errorf("marshal insights: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return convert.ConversionResult{}, fmt.Errorf("write insights: %w", err)
	}

	return convert.ConversionResult{
		OutputPath: outPath,
		MimeType:   "application/json",
		Filename:   "insights.json",
	}, nil
}

var (
	sentenceSplit = regexp.MustCompile(`[.!?]+(\s|$)`)
	wordSplit     = regexp.MustCompile(`[\p{L}\p{N}']+`)
)

// A closed set of function words to strip before ranking keywords by
// frequency. Not exhaustive - it doesn't need to be, since the top of a
// frequency table is dominated by the highest-frequency words in any
// language regardless of a few misses.
var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"to": true, "of": true, "in": true, "on": true, "at": true, "for": true,
	"with": true, "by": true, "from": true, "as": true, "it": true, "this": true,
	"that": true, "these": true, "those": true, "i": true, "you": true, "he": true,
	"she": true, "we": true, "they": true, "his": true, "her": true, "its": true,
	"their": true, "not": true, "no": true, "so": true, "if": true, "then": true,
	"than": true, "there": true, "here": true, "which": true, "who": true, "what": true,
	"when": true, "where": true, "how": true, "will": true, "would": true, "can": true,
	"could": true, "should": true, "do": true, "does": true, "did": true, "has": true,
	"have": true, "had": true, "page": true,
}

func analyzeText(text string) Insights {
	words := wordSplit.FindAllString(text, -1)
	sentences := sentenceSplit.Split(strings.TrimSpace(text), -1)
	sentenceCount := 0
	for _, s := range sentences {
		if strings.TrimSpace(s) != "" {
			sentenceCount++
		}
	}

	langInfo := whatlanggo.Detect(text)

	preview := strings.TrimSpace(text)
	// Runes, not bytes: truncating multi-byte UTF-8 mid-character produces
	// invalid output for any non-ASCII script.
	runes := []rune(preview)
	if len(runes) > 500 {
		preview = string(runes[:500]) + "…"
	}

	return Insights{
		WordCount:      len(words),
		CharacterCount: len([]rune(text)),
		SentenceCount:  sentenceCount,
		Language:       langInfo.Lang.Iso6391(),
		LanguageName:   langInfo.Lang.String(),
		Confident:      langInfo.IsReliable(),
		TopKeywords:    topKeywords(words, 10),
		Preview:        preview,
	}
}

func topKeywords(words []string, n int) []string {
	freq := make(map[string]int)
	for _, w := range words {
		lw := strings.ToLower(w)
		if len(lw) < 3 || stopwords[lw] {
			continue
		}
		freq[lw]++
	}

	type kv struct {
		word  string
		count int
	}
	ranked := make([]kv, 0, len(freq))
	for w, c := range freq {
		ranked = append(ranked, kv{w, c})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].count != ranked[j].count {
			return ranked[i].count > ranked[j].count
		}
		return ranked[i].word < ranked[j].word // stable tie-break
	})

	if len(ranked) > n {
		ranked = ranked[:n]
	}
	out := make([]string, len(ranked))
	for i, kv := range ranked {
		out[i] = kv.word
	}
	return out
}

func pdfMetadata(path string) (*PDFMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := pdfcpuapi.PDFInfo(f, path, nil, false, nil)
	if err != nil {
		return nil, err
	}

	return &PDFMetadata{
		PageCount:    info.PageCount,
		Title:        info.Title,
		Author:       info.Author,
		Subject:      info.Subject,
		Creator:      info.Creator,
		CreationDate: info.CreationDate,
	}, nil
}
