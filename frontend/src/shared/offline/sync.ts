// Sync engine: replays the outbox on reconnect/app-open and refreshes the
// Dexie cache mirrors after successful writes (research R7 — Background
// Sync API is unsupported on iOS Safari, so we sync on 'online' + app open).
import type { QueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { AttemptResult } from '../api/types';
import { db, type OutboxEntry } from './db';
import { replay } from './outbox';

async function postAttempt(entry: OutboxEntry): Promise<AttemptResult> {
  return api.post<AttemptResult>(`/exercises/${entry.exerciseId}/attempts`, {
    attemptId: entry.id,
    answer: entry.answer,
    completedAt: entry.completedAt,
    isReview: entry.isReview,
  });
}

/** syncNow drains the outbox and invalidates server-state queries. */
export async function syncNow(queryClient: QueryClient): Promise<void> {
  const { sent } = await replay(postAttempt);
  if (sent > 0) {
    await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    await queryClient.invalidateQueries({ queryKey: ['tracks'] });
    await queryClient.invalidateQueries({ queryKey: ['me'] });
  }
}

/** startSync wires the reconnect trigger; returns a cleanup function. */
export function startSync(queryClient: QueryClient): () => void {
  const onOnline = () => {
    void syncNow(queryClient);
  };
  window.addEventListener('online', onOnline);
  if (navigator.onLine) void syncNow(queryClient); // app-open drain
  return () => window.removeEventListener('online', onOnline);
}

// ─── Cache mirrors (SWR reads survive full page loads offline) ──────────

export async function mirrorProfile(user: import('../api/types').User): Promise<void> {
  await db.cachedProfile.put({ key: 'me', user, updatedAt: Date.now() });
}

export async function readMirroredProfile() {
  return (await db.cachedProfile.get('me'))?.user;
}

export async function mirrorTracks(modules: import('../api/types').Module[]): Promise<void> {
  await db.cachedTracks.put({ key: 'tracks', modules, updatedAt: Date.now() });
}

export async function readMirroredTracks() {
  return (await db.cachedTracks.get('tracks'))?.modules;
}

export async function mirrorLesson(lesson: import('../api/types').Lesson): Promise<void> {
  await db.cachedLessons.put({ id: lesson.id, lesson, updatedAt: Date.now() });
}

export async function readMirroredLesson(id: string) {
  return (await db.cachedLessons.get(id))?.lesson;
}

export async function mirrorDashboard(data: import('../api/types').Dashboard): Promise<void> {
  await db.cachedDashboard.put({ key: 'dashboard', data, updatedAt: Date.now() });
}

export async function readMirroredDashboard() {
  return (await db.cachedDashboard.get('dashboard'))?.data;
}
