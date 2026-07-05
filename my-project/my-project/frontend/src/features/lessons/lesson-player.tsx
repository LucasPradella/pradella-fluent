// Lesson player (US2): steps through exercises, shows feedback, celebrates
// completion with XP (FR-008..FR-012). Speaking steps host the US3 flow.
import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useLesson } from '../../shared/api/hooks';
import { ApiError } from '../../shared/api/client';
import type { Exercise, SpeechResult } from '../../shared/api/types';
import {
  AttemptFeedback,
  ListeningChoice,
  ListeningOrder,
  QueuedFeedback,
  WritingExercise,
} from './exercise-widgets';
import { useSubmitAttempt, type SubmitOutcome } from './use-submit-attempt';
import { SpeakingExercise } from '../speaking/speaking-exercise';
import { LessonCompletion } from './completion';

type StepState =
  | { phase: 'answering' }
  | { phase: 'feedback'; outcome: SubmitOutcome };

export function LessonPlayer() {
  const { lessonId = '' } = useParams();
  const lesson = useLesson(lessonId);
  const submitAttempt = useSubmitAttempt();

  const [index, setIndex] = useState(0);
  const [step, setStep] = useState<StepState>({ phase: 'answering' });
  const [finished, setFinished] = useState(false);
  const [xpAwarded, setXPAwarded] = useState(0);

  if (lesson.isLoading) return <p>Carregando lição…</p>;
  if (lesson.isError) {
    const err = lesson.error;
    if (err instanceof ApiError && err.status === 403) {
      return (
        <div className="card">
          <p role="alert">🔒 Esta lição pertence a uma trilha acima do seu nível atual.</p>
          <Link to="/tracks">Voltar às trilhas</Link>
        </div>
      );
    }
    return (
      <p role="alert" style={{ color: 'var(--error)' }}>
        Não foi possível carregar a lição. Verifique sua conexão.
      </p>
    );
  }
  const data = lesson.data!;
  const exercises = data.exercises;

  if (finished || index >= exercises.length) {
    return <LessonCompletion lesson={data} xpAwarded={xpAwarded} />;
  }

  const exercise = exercises[index];

  const advance = () => {
    setStep({ phase: 'answering' });
    if (index + 1 >= exercises.length) {
      setFinished(true);
    } else {
      setIndex(index + 1);
    }
  };

  const handleOutcome = (outcome: SubmitOutcome) => {
    if (outcome.kind === 'scored' && outcome.result.xpAwarded > 0) {
      setXPAwarded(outcome.result.xpAwarded);
    }
    setStep({ phase: 'feedback', outcome });
  };

  const submit = async (answer: string): Promise<SubmitOutcome> =>
    submitAttempt.mutateAsync({ exerciseId: exercise.id, answer });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <header>
        <h1 style={{ fontSize: 20 }}>{data.title}</h1>
        <p style={{ color: 'var(--text-secondary)' }}>{data.pedagogicalFocus}</p>
        <p aria-live="polite" style={{ color: 'var(--text-secondary)', fontSize: 14 }}>
          Exercício {index + 1} de {exercises.length}
        </p>
        <progress value={index} max={exercises.length} style={{ width: '100%' }} />
      </header>

      {step.phase === 'answering' ? (
        exercise.exerciseType === 'speaking' ? (
          // Speaking shows its own result/retry UI (US3); the skip path
          // never blocks lesson completion (FR-016).
          <SpeakingExercise
            exercise={exercise}
            onFinished={(result: SpeechResult | null) => {
              if (result && result.xpAwarded > 0) setXPAwarded(result.xpAwarded);
              advance();
            }}
          />
        ) : (
          <ExerciseHost
            exercise={exercise}
            busy={submitAttempt.isPending}
            submit={submit}
            onDone={handleOutcome}
          />
        )
      ) : step.outcome.kind === 'scored' ? (
        <>
          <ReadOnlyPrompt exercise={exercise} />
          <AttemptFeedback result={step.outcome.result} onNext={advance} />
        </>
      ) : (
        <QueuedFeedback onNext={advance} />
      )}
    </div>
  );
}

function ReadOnlyPrompt({ exercise }: { exercise: Exercise }) {
  return <p style={{ fontSize: 18, color: 'var(--text-secondary)' }}>{exercise.promptContext}</p>;
}

function ExerciseHost({
  exercise,
  busy,
  submit,
  onDone,
}: {
  exercise: Exercise;
  busy: boolean;
  submit: (answer: string) => Promise<SubmitOutcome>;
  onDone: (o: SubmitOutcome) => void;
}) {
  switch (exercise.exerciseType) {
    case 'translate':
    case 'fill_blank':
      return <WritingExercise exercise={exercise} submit={submit} onDone={onDone} busy={busy} />;
    case 'listening_choice':
      return <ListeningChoice exercise={exercise} submit={submit} onDone={onDone} busy={busy} />;
    case 'listening_order':
      return <ListeningOrder exercise={exercise} submit={submit} onDone={onDone} busy={busy} />;
    default:
      return null;
  }
}
