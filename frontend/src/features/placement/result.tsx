// Placement result screen: assigned level + which tracks unlock (FR-006).
import { useTracks } from '../../shared/api/hooks';
import type { ProficiencyLevel } from '../../shared/api/types';

const levelLabel: Record<ProficiencyLevel, string> = {
  basic: 'Básico',
  intermediate: 'Intermediário',
  advanced: 'Avançado',
};

export function PlacementResult({
  level,
  onContinue,
}: {
  level: ProficiencyLevel | null;
  onContinue: () => void;
}) {
  const { data: modules } = useTracks();

  return (
    <main style={{ maxWidth: 560, margin: '0 auto', padding: 24, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <h1>Seu nível: {level ? levelLabel[level] : '—'}</h1>
      <p>
        Com base nas suas respostas, desbloqueamos as trilhas do seu nível. As trilhas acima dele
        ficam visíveis, mas bloqueadas até você evoluir.
      </p>

      {modules && (
        <ul style={{ listStyle: 'none', padding: 0, display: 'flex', flexDirection: 'column', gap: 8 }}>
          {modules.map((m) => (
            <li key={m.id} className="card" style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span>
                {m.title}{' '}
                <span style={{ color: 'var(--text-secondary)' }}>({levelLabel[m.difficultyLevel]})</span>
              </span>
              <span aria-label={m.locked ? 'Bloqueada' : 'Desbloqueada'}>
                {m.locked ? '🔒' : '✅'}
              </span>
            </li>
          ))}
        </ul>
      )}

      <button onClick={onContinue}>Começar a estudar</button>
    </main>
  );
}
