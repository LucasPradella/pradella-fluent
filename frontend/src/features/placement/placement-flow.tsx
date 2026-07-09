// Adaptive placement flow (US1): start/resume, progress n/12, result with
// level + locked/unlocked tracks (FR-002..FR-006).
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { api, ApiError } from '../../shared/api/client';
import type { PlacementState } from '../../shared/api/types';
import { QuestionCard } from './question-card';
import { PlacementResult } from './result';

export function PlacementFlow() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const session = useQuery<PlacementState | null>({
    queryKey: ['placement'],
    queryFn: async () => {
      try {
        // Resume an active session if one exists (US1 scenario 6).
        return await api.get<PlacementState>('/placement/session');
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) return null;
        throw err;
      }
    },
    staleTime: 0,
    gcTime: 0,
  });

  const start = useMutation({
    mutationFn: () => api.post<PlacementState>('/placement/session'),
    onSuccess: (state) => queryClient.setQueryData(['placement'], state),
  });

  const answer = useMutation({
    mutationFn: (input: { questionId: string; answer: string }) =>
      api.post<PlacementState>('/placement/session/answers', input),
    onSuccess: async (state) => {
      queryClient.setQueryData(['placement'], state);
      if (state.status === 'completed') {
        await queryClient.invalidateQueries({ queryKey: ['me'] });
        await queryClient.invalidateQueries({ queryKey: ['tracks'] });
      }
    },
  });

  if (session.isLoading) return <p style={{ padding: 24 }}>Carregando teste…</p>;
  if (session.isError) {
    return (
      <main style={{ padding: 24 }}>
        <p role="alert" style={{ color: 'var(--error)' }}>
          O teste de nivelamento precisa de conexão com a internet.
        </p>
      </main>
    );
  }

  const state = session.data;

  if (!state) {
    return (
      <main style={{ maxWidth: 560, margin: '0 auto', padding: 24, display: 'flex', flexDirection: 'column', gap: 16 }}>
        <h1>Teste de nivelamento</h1>
        <p>
          Vamos descobrir seu nível de inglês com no máximo <strong>12 perguntas</strong>. A
          dificuldade se adapta às suas respostas — responda sem ajuda para um resultado preciso.
        </p>
        <button onClick={() => start.mutate()} disabled={start.isPending}>
          {start.isPending ? 'Preparando…' : 'Começar o teste'}
        </button>
      </main>
    );
  }

  if (state.status === 'completed') {
    return <PlacementResult level={state.assignedLevel} onContinue={() => navigate('/tracks')} />;
  }

  return (
    <main style={{ maxWidth: 560, margin: '0 auto', padding: 24, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <header>
        <h1 style={{ fontSize: 20 }}>Teste de nivelamento</h1>
        <p aria-live="polite" style={{ color: 'var(--text-secondary)' }}>
          Pergunta {state.questionsServed + 1} de 12
        </p>
        <progress value={state.questionsServed} max={12} style={{ width: '100%' }} />
      </header>

      {state.nextQuestion && (
        <QuestionCard
          question={state.nextQuestion}
          disabled={answer.isPending}
          onAnswer={(a) =>
            answer.mutate({ questionId: state.nextQuestion!.id, answer: a })
          }
        />
      )}
      {answer.isError && (
        <p role="alert" style={{ color: 'var(--error)' }}>
          Não foi possível enviar a resposta. Verifique sua conexão e tente de novo.
        </p>
      )}
    </main>
  );
}
