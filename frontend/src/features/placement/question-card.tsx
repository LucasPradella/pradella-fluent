// Placement question renderer per type (choice, listening_choice, order).
import { useState } from 'react';
import type { NextQuestion } from '../../shared/api/types';

interface Props {
  question: NextQuestion;
  onAnswer: (answer: string) => void;
  disabled: boolean;
}

export function QuestionCard({ question, onAnswer, disabled }: Props) {
  if (question.questionType === 'order') {
    return <OrderQuestion key={question.id} question={question} onAnswer={onAnswer} disabled={disabled} />;
  }
  return <ChoiceQuestion key={question.id} question={question} onAnswer={onAnswer} disabled={disabled} />;
}

function ChoiceQuestion({ question, onAnswer, disabled }: Props) {
  return (
    <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {question.audioAssetUrl && (
        <audio controls src={question.audioAssetUrl} aria-label="Áudio da questão" />
      )}
      <p style={{ fontSize: 18 }}>{question.prompt}</p>
      <div role="group" aria-label="Alternativas" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {question.options.map((opt) => (
          <button
            key={opt}
            className="secondary"
            disabled={disabled}
            onClick={() => onAnswer(opt)}
            style={{ textAlign: 'left' }}
          >
            {opt}
          </button>
        ))}
      </div>
    </div>
  );
}

/** Word-block ordering: tap words to build the sentence. */
function OrderQuestion({ question, onAnswer, disabled }: Props) {
  const [picked, setPicked] = useState<number[]>([]);

  const sentence = picked.map((i) => question.options[i]).join(' ');
  const remaining = question.options.map((w, i) => ({ w, i })).filter(({ i }) => !picked.includes(i));

  return (
    <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {question.audioAssetUrl && (
        <audio controls src={question.audioAssetUrl} aria-label="Áudio da questão" />
      )}
      <p style={{ fontSize: 18 }}>{question.prompt}</p>
      <div
        aria-label="Sua frase"
        className="card"
        style={{ minHeight: 52, background: 'var(--surface-2)' }}
      >
        {sentence || <span style={{ color: 'var(--text-disabled)' }}>Toque nas palavras abaixo…</span>}
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
        {remaining.map(({ w, i }) => (
          <button
            key={i}
            className="secondary"
            disabled={disabled}
            onClick={() => setPicked((p) => [...p, i])}
          >
            {w}
          </button>
        ))}
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        <button
          className="secondary"
          disabled={disabled || picked.length === 0}
          onClick={() => setPicked((p) => p.slice(0, -1))}
        >
          Desfazer
        </button>
        <button
          disabled={disabled || picked.length !== question.options.length}
          onClick={() => onAnswer(sentence)}
          style={{ flex: 1 }}
        >
          Confirmar
        </button>
      </div>
    </div>
  );
}
