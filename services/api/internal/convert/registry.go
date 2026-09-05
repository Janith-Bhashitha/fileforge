package convert

import (
	"context"
	"fmt"
)

// ConversionRequest describes one unit of work: an input file's bytes, the
// operation to run, and any options. Storage is left to the caller (Phase 2
// writes to local disk; Phase 6 swaps this for S3 without touching processors).
type ConversionRequest struct {
	InputPath string
	Options   map[string]string
}

// ConversionResult is what a Processor hands back: the path to the produced
// output file(s) plus its detected MIME type.
type ConversionResult struct {
	OutputPath string
	MimeType   string
	Filename   string
}

// Processor is the interface every conversion operation implements. Keeping
// this stable is what lets Phase 3 wrap it with a queue and Phase 6 wrap it
// with S3 without rewriting operation logic.
type Processor interface {
	Process(ctx context.Context, req ConversionRequest) (ConversionResult, error)
}

// Registry maps "operation:version" keys to their Processor implementation.
type Registry struct {
	processors map[string]Processor
}

func NewRegistry() *Registry {
	return &Registry{processors: make(map[string]Processor)}
}

func (r *Registry) Register(name, version string, p Processor) {
	r.processors[key(name, version)] = p
}

func (r *Registry) Resolve(name, version string) (Processor, error) {
	p, ok := r.processors[key(name, version)]
	if !ok {
		return nil, fmt.Errorf("unknown operation: %s:%s", name, version)
	}
	return p, nil
}

func key(name, version string) string {
	return name + ":" + version
}
