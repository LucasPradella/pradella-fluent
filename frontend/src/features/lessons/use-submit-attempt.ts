// Attempt submission with offline outbox fallback (US2 + US5).
import { useMutation } from '@tanstack/react-query';
import { api, ApiError, OfflineError } from '../../shared/api/client';
import type { AttemptResult } from '../../shared/api/types';
import { enqueue } from '../../shared/offline/outbox';
import { useInvalidateServerState } from '../../shared/api/hooks';

export interface SubmitInput {
  exerciseId: string;
  answer: string;
  isReview?: boolean;
}

export type SubmitOutcome =
  | { kind: 'scored'; result: AttemptResult }
  | { kind: 'queued' }; // saved offline, will sync on reconnect

export function useSubmitAttempt() {
  const invalidate = useInvalidateServerState();

  return useMutation<SubmitOutcome, Error, SubmitInput>({
    mutationFn: async ({ exerciseId, answer, isReview = false }) => {
      const attemptId = crypto.randomUUID();
      const completedAt = new Date().toISOString();
      try {
        const result = await api.post<AttemptResult>(`/exercises/${exerciseId}/attempts`, {
          attemptId,
          answer,
          completedAt,
          isReview,
        });
        return { kind: 'scored', result };
      } catch (err) {
        const offline =
          err instanceof OfflineError || (err instanceof TypeError && !navigator.onLine);
        if (offline) {
          await enqueue({ attemptId, exerciseId, answer, completedAt, isReview });
          return { kind: 'queued' };
        }
        if (err instanceof ApiError) throw err;
        throw err;
      }
    },
    onSuccess: async (outcome) => {
      if (outcome.kind === 'scored') await invalidate();
    },
  });
}
