// Outbox replay rules: FIFO, idempotent double-replay, network vs server
// failures (US5 — T064). Uses fake-indexeddb under jsdom.
import 'fake-indexeddb/auto';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { db, type OutboxEntry } from './db';
import { enqueue, pendingCount, replay } from './outbox';
import type { AttemptResult } from '../api/types';

const okResult: AttemptResult = {
  correct: true,
  accuracyScore: 1,
  toleratedTypos: [],
  expectedAnswer: null,
  lessonCompleted: false,
  xpAwarded: 0,
};

function payload(n: number) {
  return {
    attemptId: `00000000-0000-7000-8000-00000000000${n}`,
    exerciseId: 'ex-1',
    answer: `answer ${n}`,
    completedAt: new Date(2026, 6, 1, 10, n).toISOString(),
    isReview: false,
  };
}

beforeEach(async () => {
  await db.outbox.clear();
  vi.restoreAllMocks();
});

describe('outbox', () => {
  it('replays FIFO on reconnect', async () => {
    const nowSpy = vi.spyOn(Date, 'now');
    nowSpy.mockReturnValue(1000);
    await enqueue(payload(1));
    nowSpy.mockReturnValue(2000);
    await enqueue(payload(2));
    nowSpy.mockReturnValue(3000);
    await enqueue(payload(3));
    nowSpy.mockRestore();

    const sentOrder: string[] = [];
    const { sent, kept } = await replay(async (e: OutboxEntry) => {
      sentOrder.push(e.answer);
      return okResult;
    });

    expect(sent).toBe(3);
    expect(kept).toBe(0);
    expect(sentOrder).toEqual(['answer 1', 'answer 2', 'answer 3']);
    expect(await pendingCount()).toBe(0);
  });

  it('is idempotent: enqueueing the same attemptId twice stores one entry', async () => {
    await enqueue(payload(1));
    await enqueue(payload(1)); // duplicate replay of the same attempt
    expect(await pendingCount()).toBe(1);

    let posts = 0;
    await replay(async () => {
      posts++;
      return okResult;
    });
    expect(posts).toBe(1); // server receives the attemptId once from this client
  });

  it('keeps remaining entries when the network drops mid-drain', async () => {
    const nowSpy = vi.spyOn(Date, 'now');
    nowSpy.mockReturnValue(1000);
    await enqueue(payload(1));
    nowSpy.mockReturnValue(2000);
    await enqueue(payload(2));
    nowSpy.mockRestore();

    let calls = 0;
    const { sent, kept } = await replay(async () => {
      calls++;
      if (calls === 2) throw new TypeError('Failed to fetch'); // network died
      return okResult;
    });

    expect(sent).toBe(1);
    expect(kept).toBe(1);
    expect(await pendingCount()).toBe(1); // unsent entry waits for the next reconnect
  });

  it('drops entries the server rejects for good (no poison pill)', async () => {
    await enqueue(payload(1));

    const rejection = Object.assign(new Error('gone'), { name: 'ApiError', status: 404 });
    const { sent, kept } = await replay(async () => {
      throw rejection;
    });

    expect(sent).toBe(0);
    expect(kept).toBe(0);
    expect(await pendingCount()).toBe(0);
  });

  it('clamps nothing client-side: completedAt is preserved for the server to clamp', async () => {
    const p = payload(1);
    await enqueue(p);
    const stored = await db.outbox.get(p.attemptId);
    expect(stored?.completedAt).toBe(p.completedAt);
  });
});
