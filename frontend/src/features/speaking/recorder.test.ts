// Recorder state machine with faked MediaRecorder/getUserMedia (T052).
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MAX_RECORDING_MS, pickMimeType, Recorder, type RecorderState } from './recorder';

class FakeMediaRecorder {
  static supported = new Set(['audio/webm;codecs=opus', 'audio/webm']);
  static isTypeSupported(t: string) {
    return FakeMediaRecorder.supported.has(t);
  }

  state: 'inactive' | 'recording' = 'inactive';
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void) | null = null;

  constructor(
    public stream: unknown,
    public options: { mimeType: string },
  ) {}

  start() {
    this.state = 'recording';
  }

  stop() {
    this.state = 'inactive';
    this.ondataavailable?.({ data: new Blob(['x'], { type: 'audio/webm' }) });
    this.onstop?.();
  }
}

const fakeStream = {
  getTracks: () => [{ stop: vi.fn() }],
};

describe('pickMimeType', () => {
  it('prefers webm/opus', () => {
    expect(pickMimeType(() => true)).toBe('audio/webm;codecs=opus');
  });

  it('falls back to mp4 on Safari', () => {
    expect(pickMimeType((t) => t === 'audio/mp4')).toBe('audio/mp4');
  });

  it('returns null when nothing is supported', () => {
    expect(pickMimeType(() => false)).toBeNull();
  });
});

describe('Recorder', () => {
  const states: RecorderState[] = [];
  let finished: { blob: Blob; mimeType: string } | null = null;

  beforeEach(() => {
    states.length = 0;
    finished = null;
    vi.useFakeTimers();
    vi.stubGlobal('MediaRecorder', FakeMediaRecorder);
    vi.stubGlobal('navigator', {
      ...navigator,
      mediaDevices: { getUserMedia: vi.fn().mockResolvedValue(fakeStream) },
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  function makeRecorder() {
    return new Recorder({
      onState: (s) => states.push(s),
      onFinish: (r) => {
        finished = r;
      },
      onError: () => {},
    });
  }

  it('walks requesting → recording → stopped and emits the blob', async () => {
    const rec = makeRecorder();
    await rec.start();
    expect(states).toEqual(['requesting', 'recording']);

    rec.stop();
    expect(states).toEqual(['requesting', 'recording', 'stopped']);
    expect(finished).not.toBeNull();
    expect(finished!.mimeType).toBe('audio/webm');
  });

  it('auto-stops at the 30 s hard cap (FR-013)', async () => {
    const rec = makeRecorder();
    await rec.start();
    expect(states.at(-1)).toBe('recording');

    vi.advanceTimersByTime(MAX_RECORDING_MS + 1);
    expect(states.at(-1)).toBe('stopped');
    expect(finished).not.toBeNull();
  });

  it('reports denied when the user blocks the microphone (FR-016)', async () => {
    vi.stubGlobal('navigator', {
      ...navigator,
      mediaDevices: { getUserMedia: vi.fn().mockRejectedValue(new Error('NotAllowedError')) },
    });
    const rec = makeRecorder();
    await rec.start();
    expect(states).toEqual(['requesting', 'denied']);
  });

  it('reports error when no supported container exists', async () => {
    FakeMediaRecorder.supported = new Set();
    const rec = makeRecorder();
    await rec.start();
    expect(states).toEqual(['error']);
    FakeMediaRecorder.supported = new Set(['audio/webm;codecs=opus', 'audio/webm']);
  });
});
