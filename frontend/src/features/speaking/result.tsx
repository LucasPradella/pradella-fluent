// Speaking result: similarity %, target sentence with missed words in
// desaturated red (FR-014, FR-015).
import type { SpeechResult } from '../../shared/api/types';

function normalizeWord(w: string): string {
  return w.toLowerCase().replace(/[.,!?;:"“”]/g, '');
}

export function SpeakingResult({
  target,
  result,
  onRetry,
  onContinue,
}: {
  target: string;
  result: SpeechResult;
  onRetry: () => void;
  onContinue: () => void;
}) {
  const missed = new Set(result.missedWords.map(normalizeWord));
  const pct = Math.round(result.similarity * 100);

  return (
    <div className="card" role="status" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {result.passed ? (
        <p style={{ color: 'var(--success)', fontWeight: 700, fontSize: 20 }}>
          ✅ Muito bem! Similaridade: {pct}%
        </p>
      ) : (
        <p style={{ color: 'var(--error)', fontWeight: 700, fontSize: 20 }}>
          Quase lá — similaridade {pct}% (mínimo 80%)
        </p>
      )}

      <p style={{ fontSize: 18 }}>
        {target.split(/\s+/).map((word, i) => {
          const isMissed = missed.has(normalizeWord(word));
          return (
            <span key={i} className={isMissed ? 'missed-word' : undefined}>
              {word}{' '}
            </span>
          );
        })}
      </p>
      {result.missedWords.length > 0 && (
        <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>
          Palavras destacadas foram omitidas ou não foram entendidas.
        </p>
      )}

      <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>
        O que entendemos: “{result.transcript}”
      </p>
      {result.xpAwarded > 0 && (
        <p style={{ color: 'var(--success)', fontWeight: 600 }}>+{result.xpAwarded} XP</p>
      )}

      <div style={{ display: 'flex', gap: 8 }}>
        <button className="secondary" onClick={onRetry}>
          Tentar de novo
        </button>
        <button onClick={onContinue} style={{ flex: 1 }}>
          Continuar
        </button>
      </div>
    </div>
  );
}
