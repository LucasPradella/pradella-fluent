// Writing and listening exercise widgets (FR-009, FR-010).
import { type FormEvent, useState } from 'react';
import type { AttemptResult, Exercise } from '../../shared/api/types';
import type { SubmitOutcome } from './use-submit-attempt';

export interface ExerciseProps {
  exercise: Exercise;
  submit: (answer: string) => Promise<SubmitOutcome>;
  onDone: (outcome: SubmitOutcome) => void;
  busy: boolean;
}

/** Feedback block shared by all written/listening types. */
export function AttemptFeedback({ result, onNext }: { result: AttemptResult; onNext: () => void }) {
  return (
    <div className="card" role="status" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {result.correct ? (
        <p style={{ color: 'var(--success)', fontWeight: 600 }}>
          ✅ Correto!{' '}
          {result.toleratedTypos.length > 0 && (
            <span style={{ color: 'var(--warning)', fontWeight: 400 }}>
              (com um pequeno erro de digitação:{' '}
              {result.toleratedTypos.map((w) => (
                <span key={w} className="tolerated-typo">
                  {w}{' '}
                </span>
              ))}
              )
            </span>
          )}
        </p>
      ) : (
        <>
          <p style={{ color: 'var(--error)', fontWeight: 600 }}>✖️ Ainda não.</p>
          {result.expectedAnswer && (
            <p>
              Resposta esperada: <strong>{result.expectedAnswer}</strong>
            </p>
          )}
        </>
      )}
      <button onClick={onNext}>Continuar</button>
    </div>
  );
}

export function QueuedFeedback({ onNext }: { onNext: () => void }) {
  return (
    <div className="card" role="status" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <p style={{ color: 'var(--warning)' }}>
        📴 Você está offline. Resposta salva — ela será corrigida e sincronizada quando a conexão
        voltar.
      </p>
      <button onClick={onNext}>Continuar</button>
    </div>
  );
}

/** translate / fill_blank: free text input with typo tolerance server-side. */
export function WritingExercise({ exercise, submit, onDone, busy }: ExerciseProps) {
  const [answer, setAnswer] = useState('');

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!answer.trim()) return;
    onDone(await submit(answer));
  }

  return (
    <form onSubmit={onSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <p style={{ fontSize: 18 }}>{exercise.promptContext}</p>
      <label>
        Sua resposta em inglês
        <input
          value={answer}
          onChange={(e) => setAnswer(e.target.value)}
          maxLength={1000}
          autoCapitalize="off"
          autoCorrect="off"
          spellCheck={false}
        />
      </label>
      <button type="submit" disabled={busy || !answer.trim()}>
        {busy ? 'Corrigindo…' : 'Verificar'}
      </button>
    </form>
  );
}

/** listening_choice: replayable audio + alternatives (FR-010). */
export function ListeningChoice({ exercise, submit, onDone, busy }: ExerciseProps) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <p style={{ fontSize: 18 }}>{exercise.promptContext}</p>
      {exercise.audioAssetUrl && (
        <audio controls src={exercise.audioAssetUrl} aria-label="Áudio do exercício (pode repetir)" />
      )}
      <div role="group" aria-label="Alternativas" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {(exercise.options ?? []).map((opt) => (
          <button
            key={opt}
            className="secondary"
            disabled={busy}
            style={{ textAlign: 'left' }}
            onClick={async () => onDone(await submit(opt))}
          >
            {opt}
          </button>
        ))}
      </div>
    </div>
  );
}

/** listening_order: build the heard sentence from word blocks (FR-010). */
export function ListeningOrder({ exercise, submit, onDone, busy }: ExerciseProps) {
  const [picked, setPicked] = useState<number[]>([]);
  const options = exercise.options ?? [];
  const sentence = picked.map((i) => options[i]).join(' ');
  const remaining = options.map((w, i) => ({ w, i })).filter(({ i }) => !picked.includes(i));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <p style={{ fontSize: 18 }}>{exercise.promptContext}</p>
      {exercise.audioAssetUrl && (
        <audio controls src={exercise.audioAssetUrl} aria-label="Áudio do exercício (pode repetir)" />
      )}
      <div className="card" aria-label="Sua frase" style={{ minHeight: 52, background: 'var(--surface-2)' }}>
        {sentence || <span style={{ color: 'var(--text-disabled)' }}>Monte a frase que você ouviu…</span>}
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
        {remaining.map(({ w, i }) => (
          <button key={i} className="secondary" disabled={busy} onClick={() => setPicked((p) => [...p, i])}>
            {w}
          </button>
        ))}
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        <button
          className="secondary"
          disabled={busy || picked.length === 0}
          onClick={() => setPicked((p) => p.slice(0, -1))}
        >
          Desfazer
        </button>
        <button
          disabled={busy || picked.length !== options.length}
          style={{ flex: 1 }}
          onClick={async () => onDone(await submit(sentence))}
        >
          {busy ? 'Corrigindo…' : 'Confirmar'}
        </button>
      </div>
    </div>
  );
}
