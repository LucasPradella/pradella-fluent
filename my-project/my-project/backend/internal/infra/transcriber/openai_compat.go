package transcriber

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"
)

// openAICompat implements the OpenAI-style POST /audio/transcriptions call
// shared by Groq and OpenAI (both accept webm/opus and mp4 — research R11).
type openAICompat struct {
	name    string // provider label for logs/metrics
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
	logger  *slog.Logger
}

// NewGroq builds the Groq adapter (whisper-large-v3-turbo).
func NewGroq(baseURL, apiKey string, logger *slog.Logger) Transcriber {
	return &openAICompat{
		name: "groq", baseURL: baseURL, apiKey: apiKey,
		model:  "whisper-large-v3-turbo",
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

// NewOpenAI builds the OpenAI adapter (gpt-4o-mini-transcribe).
func NewOpenAI(baseURL, apiKey string, logger *slog.Logger) Transcriber {
	return &openAICompat{
		name: "openai", baseURL: baseURL, apiKey: apiKey,
		model:  "gpt-4o-mini-transcribe",
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

func (p *openAICompat) Transcribe(ctx context.Context, audio io.Reader, opts TranscribeOpts) (Transcript, error) {
	start := time.Now()

	ext := ".webm"
	if opts.MIMEType == "audio/mp4" {
		ext = ".mp4"
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "audio"+ext)
	if err != nil {
		return Transcript{}, err
	}
	if _, err := io.Copy(fw, audio); err != nil {
		return Transcript{}, fmt.Errorf("%s: read audio: %w", p.name, err)
	}
	_ = mw.WriteField("model", p.model)
	if opts.Language != "" {
		_ = mw.WriteField("language", opts.Language)
	}
	_ = mw.WriteField("response_format", "json")
	if err := mw.Close(); err != nil {
		return Transcript{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/audio/transcriptions", &body)
	if err != nil {
		return Transcript{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := p.client.Do(req)
	if err != nil {
		p.observe(ctx, start, 0, err)
		return Transcript{}, fmt.Errorf("%s: %w", p.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		limited, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		err := fmt.Errorf("%s: status %d: %s", p.name, resp.StatusCode, string(limited))
		p.observe(ctx, start, resp.StatusCode, err)
		return Transcript{}, err
	}

	var out struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		p.observe(ctx, start, resp.StatusCode, err)
		return Transcript{}, fmt.Errorf("%s: decode: %w", p.name, err)
	}

	p.observe(ctx, start, resp.StatusCode, nil)
	return Transcript{Text: out.Text, Provider: p.name}, nil
}

// observe emits per-provider latency/error metrics as structured logs
// (research R2). Never logs audio content.
func (p *openAICompat) observe(ctx context.Context, start time.Time, status int, err error) {
	if p.logger == nil {
		return
	}
	attrs := []any{
		slog.String("provider", p.name),
		slog.Int64("latency_ms", time.Since(start).Milliseconds()),
		slog.Int("status", status),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		p.logger.WarnContext(ctx, "transcription provider call failed", attrs...)
		return
	}
	p.logger.InfoContext(ctx, "transcription provider call", attrs...)
}
