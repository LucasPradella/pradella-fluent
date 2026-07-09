package transcriber

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"time"
)

// primaryDeadline is the soft deadline for the primary provider before we
// fail over (research R2 — fits the 3.5 s p95 speech-loop budget).
const primaryDeadline = 2500 * time.Millisecond

// Failover tries the primary provider under a soft deadline and retries
// once on the secondary on any error, timeout or throttle response.
type Failover struct {
	primary   Transcriber
	secondary Transcriber
	logger    *slog.Logger
}

// NewFailover builds the failover chain. secondary may be nil (single provider).
func NewFailover(primary, secondary Transcriber, logger *slog.Logger) *Failover {
	return &Failover{primary: primary, secondary: secondary, logger: logger}
}

func (f *Failover) Transcribe(ctx context.Context, audio io.Reader, opts TranscribeOpts) (Transcript, error) {
	// Buffer once so the secondary can replay the same audio.
	buf, err := io.ReadAll(audio)
	if err != nil {
		return Transcript{}, err
	}

	pctx, cancel := context.WithTimeout(ctx, primaryDeadline)
	t, perr := f.primary.Transcribe(pctx, bytes.NewReader(buf), opts)
	cancel()
	if perr == nil {
		return t, nil
	}

	if f.secondary == nil {
		return Transcript{}, ErrProvidersUnavailable
	}
	if f.logger != nil {
		f.logger.WarnContext(ctx, "failing over to secondary transcriber",
			slog.String("error", perr.Error()))
	}

	t, serr := f.secondary.Transcribe(ctx, bytes.NewReader(buf), opts)
	if serr != nil {
		return Transcript{}, ErrProvidersUnavailable
	}
	return t, nil
}
