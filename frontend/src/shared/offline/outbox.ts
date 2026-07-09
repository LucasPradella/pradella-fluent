// Offline outbox: written exercise attempts made offline are queued here
// and replayed FIFO on reconnect. The server dedupes by attemptId, so a
// double replay creates exactly one progress log (FR-023, US5).
import type { AttemptResult } from '../api/types';
import { db, type OutboxEntry } from './db';

export interface AttemptPayload {
  attemptId: string;
  exerciseId: string;
  answer: string;
  completedAt: string;
  isReview: boolean;
}

/** enqueue stores a pending attempt (id doubles as the dedupe key). */
export async function enqueue(payload: AttemptPayload): Promise<void> {
  const entry: OutboxEntry = {
    id: payload.attemptId,
    exerciseId: payload.exerciseId,
    answer: payload.answer,
    completedAt: payload.completedAt,
    isReview: payload.isReview,
    createdAt: Date.now(),
  };
  await db.outbox.put(entry);
}

export async function pendingCount(): Promise<number> {
  return db.outbox.count();
}

/** Poster submits one attempt to the API (injected for tests). */
export type Poster = (entry: OutboxEntry) => Promise<AttemptResult>;

/**
 * replay drains the outbox in FIFO order. Entries are removed only after
 * the server acknowledges them (2xx). A network failure stops the drain —
 * remaining entries wait for the next reconnect. HTTP errors other than
 * network failures also remove the entry (the server rejected it for good).
 */
export async function replay(post: Poster): Promise<{ sent: number; kept: number }> {
  const entries = await db.outbox.orderBy('createdAt').toArray();
  let sent = 0;
  for (const entry of entries) {
    try {
      await post(entry);
      await db.outbox.delete(entry.id);
      sent++;
    } catch (err) {
      if (isNetworkError(err)) {
        return { sent, kept: entries.length - sent };
      }
      // Server-side rejection (4xx): drop to avoid a poison-pill loop.
      await db.outbox.delete(entry.id);
    }
  }
  return { sent, kept: 0 };
}

function isNetworkError(err: unknown): boolean {
  if (err instanceof TypeError) return true; // fetch network failure
  return err instanceof Error && err.name === 'OfflineError';
}
