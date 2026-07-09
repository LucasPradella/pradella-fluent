// Dexie (IndexedDB) schema — cache only, never source of truth (FR-023).
import Dexie, { type EntityTable } from 'dexie';
import type { Dashboard, Lesson, Module, User } from '../api/types';

/** One pending offline write, replayed FIFO on reconnect. */
export interface OutboxEntry {
  id: string; // client-generated UUID == attemptId (server dedupe key)
  exerciseId: string;
  answer: string;
  completedAt: string; // ISO — clamped server-side
  isReview: boolean;
  createdAt: number; // epoch ms, FIFO ordering
}

interface CachedProfile {
  key: 'me';
  user: User;
  updatedAt: number;
}

interface CachedTracks {
  key: 'tracks';
  modules: Module[];
  updatedAt: number;
}

interface CachedLesson {
  id: string;
  lesson: Lesson;
  updatedAt: number;
}

interface CachedDashboard {
  key: 'dashboard';
  data: Dashboard;
  updatedAt: number;
}

const db = new Dexie('fluentdev') as Dexie & {
  cachedProfile: EntityTable<CachedProfile, 'key'>;
  cachedTracks: EntityTable<CachedTracks, 'key'>;
  cachedLessons: EntityTable<CachedLesson, 'id'>;
  cachedDashboard: EntityTable<CachedDashboard, 'key'>;
  outbox: EntityTable<OutboxEntry, 'id'>;
};

db.version(1).stores({
  cachedProfile: 'key',
  cachedTracks: 'key',
  cachedLessons: 'id',
  cachedDashboard: 'key',
  outbox: 'id, createdAt',
});

export { db };
