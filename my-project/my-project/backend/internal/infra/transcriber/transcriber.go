// Package transcriber defines the Transcriber port and its provider
// adapters (research R2). Providers are NEVER called outside this package.
package transcriber

import (
	"context"
	"errors"
	"io"
)

// Transcript is the provider-agnostic transcription result.
type Transcript struct {
	Text     string
	Provider string // groq | openai
}

// TranscribeOpts carries per-request hints.
type TranscribeOpts struct {
	MIMEType string // audio/webm | audio/mp4
	Language string // BCP-47 hint, e.g. "en"
}

// Transcriber is the port implemented by every provider adapter.
type Transcriber interface {
	Transcribe(ctx context.Context, audio io.Reader, opts TranscribeOpts) (Transcript, error)
}

// ErrProvidersUnavailable signals that every configured provider failed —
// the API surfaces it as 503 (US3 edge case).
var ErrProvidersUnavailable = errors.New("all transcription providers unavailable")
