// Spaced-review quick session (US4): runs due exercises reusing the lesson
// widgets with isReview=true so completions log is_review (FR-020).
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useReviewQueue } from '../../shared/api/hooks';
import {
  AttemptFeedback,
  ListeningChoice,
  ListeningOrder,
  QueuedFeedback,
  WritingExercise,
} from '../lessons/exercise-widgets';
import { useSubmitAttempt, type SubmitOutcome } from '../lessons/use-submit-attempt';
import { SpeakingExercise } from '../speaking/speaking-exercise';

export function ReviewSession() {
  const queue = useReviewQueue();
  const submitAttempt = useSubmitAttempt();
  const [index, setIndex] = useState(0);
  const [outcome, setOutcome] = useState<SubmitOutcome | null>(null);

  if (queue.isLoading) return <p>Carregando revisões…</p>;
  if (queue.isError || !queue.data) {
    return (
      <p role="alert" style={{ color: 'var(--error)' }}>
        Não foi possível carregar a fila de revisão. Verifique sua conexão.
      </p>
    );
  }

  const items = queue.data;
  if (items.length === 0 || index >= items.length) {
    return (
      <div className="card" style={{ textAlign: 'center', padding: 32 }}>
        <div aria-hidden style={{ fontSize: 48 }}>🎯</div>
        <h1>{items.length === 0 ? 'Nada para revisar agora' : 'Revisão concluída!'}</h1>
        <p style={{ color: 'var(--text-secondary)' }}>
          Itens com erro voltam aqui em intervalos crescentes: 1, 3, 7 e 21 dias.
        </p>
        <Link to="/dashboard">
          <button>Voltar ao painel</button>
        </Link>
      </div>
    );
  }

  const item = items[index];
  const exercise = item.exercise;
  const advance = () => {
    setOutcome(null);
    setIndex(index + 1);
  };
  const submit = async (answer: string) =>
    submitAttempt.mutateAsync({ exerciseId: exercise.id, answer, isReview: true });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <header>
        <h1 style={{ fontSize: 20 }}>Revisão espaçada</h1>
        <p aria-live="polite" style={{ color: 'var(--text-secondary)' }}>
          Item {index + 1} de {items.length}
        </p>
        <progress value={index} max={items.length} style={{ width: '100%' }} />
      </header>

      {outcome ? (
        outcome.kind === 'scored' ? (
          <AttemptFeedback result={outcome.result} onNext={advance} />
        ) : (
          <QueuedFeedback onNext={advance} />
        )
      ) : exercise.exerciseType === 'speaking' ? (
        <SpeakingExercise exercise={exercise} onFinished={advance} />
      ) : exercise.exerciseType === 'listening_choice' ? (
        <ListeningChoice exercise={exercise} submit={submit} onDone={setOutcome} busy={submitAttempt.isPending} />
      ) : exercise.exerciseType === 'listening_order' ? (
        <ListeningOrder exercise={exercise} submit={submit} onDone={setOutcome} busy={submitAttempt.isPending} />
      ) : (
        <WritingExercise exercise={exercise} submit={submit} onDone={setOutcome} busy={submitAttempt.isPending} />
      )}
    </div>
  );
}
