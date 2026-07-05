// Server-state hooks: SWR reads with Dexie mirrors so cached data renders
// offline even after a cold start (US5).
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiError } from './client';
import type { Dashboard, Lesson, Module, ReviewItem, User } from './types';
import {
  mirrorDashboard,
  mirrorLesson,
  mirrorProfile,
  mirrorTracks,
  readMirroredDashboard,
  readMirroredLesson,
  readMirroredProfile,
  readMirroredTracks,
} from '../offline/sync';

/** useMe resolves the session user; null when unauthenticated. */
export function useMe() {
  return useQuery<User | null>({
    queryKey: ['me'],
    queryFn: async () => {
      try {
        const user = await api.get<User>('/me');
        await mirrorProfile(user);
        return user;
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) return null;
        // Offline: fall back to the mirrored profile.
        const cached = await readMirroredProfile();
        if (cached) return cached;
        throw err;
      }
    },
    staleTime: 60_000,
  });
}

export function useTracks() {
  return useQuery<Module[]>({
    queryKey: ['tracks'],
    queryFn: async () => {
      try {
        const modules = await api.get<Module[]>('/tracks');
        await mirrorTracks(modules);
        return modules;
      } catch (err) {
        const cached = await readMirroredTracks();
        if (cached) return cached;
        throw err;
      }
    },
  });
}

export function useLesson(lessonId: string) {
  return useQuery<Lesson>({
    queryKey: ['lesson', lessonId],
    queryFn: async () => {
      try {
        const lesson = await api.get<Lesson>(`/lessons/${lessonId}`);
        await mirrorLesson(lesson);
        return lesson;
      } catch (err) {
        const cached = await readMirroredLesson(lessonId);
        if (cached) return cached;
        throw err;
      }
    },
  });
}

export function useDashboard() {
  return useQuery<Dashboard>({
    queryKey: ['dashboard'],
    queryFn: async () => {
      try {
        const data = await api.get<Dashboard>('/dashboard');
        await mirrorDashboard(data);
        return data;
      } catch (err) {
        const cached = await readMirroredDashboard();
        if (cached) return cached;
        throw err;
      }
    },
  });
}

export function useReviewQueue() {
  return useQuery<ReviewItem[]>({
    queryKey: ['review-queue'],
    queryFn: () => api.get<ReviewItem[]>('/review-queue'),
  });
}

/** useInvalidateServerState refreshes everything after a write. */
export function useInvalidateServerState() {
  const qc = useQueryClient();
  return async () => {
    await Promise.all([
      qc.invalidateQueries({ queryKey: ['me'] }),
      qc.invalidateQueries({ queryKey: ['tracks'] }),
      qc.invalidateQueries({ queryKey: ['dashboard'] }),
      qc.invalidateQueries({ queryKey: ['review-queue'] }),
    ]);
  };
}
