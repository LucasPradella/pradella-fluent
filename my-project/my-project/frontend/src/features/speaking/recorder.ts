// MediaRecorder wrapper (research R11): webm/opus preferred, mp4 fallback
// for Safari, hard 30 s auto-stop (FR-013). Framework-free state machine so
// it is unit-testable with fakes.

export const MAX_RECORDING_MS = 30_000;

export type RecorderState = 'idle' | 'requesting' | 'recording' | 'stopped' | 'denied' | 'error';

export interface Recording {
  blob: Blob;
  mimeType: string;
}

/** pickMimeType chooses the first container both we and the browser support. */
export function pickMimeType(
  isTypeSupported: (t: string) => boolean = (t) =>
    typeof MediaRecorder !== 'undefined' && MediaRecorder.isTypeSupported(t),
): string | null {
  const candidates = ['audio/webm;codecs=opus', 'audio/webm', 'audio/mp4'];
  for (const c of candidates) {
    if (isTypeSupported(c)) return c;
  }
  return null;
}

export interface RecorderCallbacks {
  onState: (state: RecorderState) => void;
  onFinish: (recording: Recording) => void;
  onError: (message: string) => void;
}

/** Recorder drives one getUserMedia + MediaRecorder session. */
export class Recorder {
  private mediaRecorder: MediaRecorder | null = null;
  private stream: MediaStream | null = null;
  private chunks: BlobPart[] = [];
  private timer: ReturnType<typeof setTimeout> | null = null;
  private readonly cb: RecorderCallbacks;

  constructor(cb: RecorderCallbacks) {
    this.cb = cb;
  }

  async start(): Promise<void> {
    const mimeType = pickMimeType();
    if (!mimeType) {
      this.cb.onState('error');
      this.cb.onError('Este navegador não suporta gravação de áudio.');
      return;
    }

    this.cb.onState('requesting');
    try {
      this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      // Permission denied → the caller offers the skip path (FR-016).
      this.cb.onState('denied');
      return;
    }

    this.chunks = [];
    this.mediaRecorder = new MediaRecorder(this.stream, { mimeType });
    this.mediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) this.chunks.push(e.data);
    };
    this.mediaRecorder.onstop = () => {
      const base = mimeType.split(';')[0];
      this.cb.onFinish({ blob: new Blob(this.chunks, { type: base }), mimeType: base });
      this.cleanup();
      this.cb.onState('stopped');
    };
    this.mediaRecorder.start();
    this.cb.onState('recording');

    // Hard 30 s cap (FR-013).
    this.timer = setTimeout(() => this.stop(), MAX_RECORDING_MS);
  }

  stop(): void {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    if (this.mediaRecorder && this.mediaRecorder.state !== 'inactive') {
      this.mediaRecorder.stop();
    }
  }

  cancel(): void {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    if (this.mediaRecorder && this.mediaRecorder.state !== 'inactive') {
      this.mediaRecorder.ondataavailable = null;
      this.mediaRecorder.onstop = null;
      this.mediaRecorder.stop();
    }
    this.cleanup();
    this.cb.onState('idle');
  }

  private cleanup(): void {
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
    this.mediaRecorder = null;
  }
}
