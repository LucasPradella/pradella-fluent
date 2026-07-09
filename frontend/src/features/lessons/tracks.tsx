// Track listing grouped by theme + level with lock badges (FR-006, FR-007).
import { Link } from 'react-router-dom';
import { useTracks } from '../../shared/api/hooks';
import type { Module, ProficiencyLevel } from '../../shared/api/types';

const levelLabel: Record<ProficiencyLevel, string> = {
  basic: 'Básico',
  intermediate: 'Intermediário',
  advanced: 'Avançado',
};

const themeLabel = { travel: '✈️ Viagem', tech: '💻 Tecnologia' } as const;

export function TracksPage() {
  const { data: modules, isLoading, isError } = useTracks();

  if (isLoading) return <p>Carregando trilhas…</p>;
  if (isError || !modules) {
    return (
      <p role="alert" style={{ color: 'var(--error)' }}>
        Não foi possível carregar as trilhas. Verifique sua conexão.
      </p>
    );
  }

  const byLevel = new Map<ProficiencyLevel, Module[]>();
  for (const m of modules) {
    const list = byLevel.get(m.difficultyLevel) ?? [];
    list.push(m);
    byLevel.set(m.difficultyLevel, list);
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      <h1>Trilhas</h1>
      {(['basic', 'intermediate', 'advanced'] as const).map((level) => {
        const list = byLevel.get(level);
        if (!list?.length) return null;
        return (
          <section key={level} aria-label={`Nível ${levelLabel[level]}`}>
            <h2 style={{ fontSize: 18, color: 'var(--text-secondary)' }}>{levelLabel[level]}</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              {list.map((m) => (
                <ModuleCard key={m.id} module={m} />
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}

function ModuleCard({ module }: { module: Module }) {
  const done = module.lessons.filter((l) => l.completed).length;
  return (
    <article className="card" aria-label={module.title} style={{ opacity: module.locked ? 0.6 : 1 }}>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
        <h3 style={{ margin: 0 }}>
          {themeLabel[module.themeType]} — {module.title} {module.locked && <span aria-label="Trilha bloqueada">🔒</span>}
        </h3>
        <span style={{ color: 'var(--text-secondary)', fontSize: 14 }}>
          {done}/{module.lessons.length}
        </span>
      </header>
      <p style={{ color: 'var(--text-secondary)' }}>{module.description}</p>
      {module.locked ? (
        <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>
          Disponível quando você alcançar o nível necessário.
        </p>
      ) : (
        <ul style={{ listStyle: 'none', padding: 0, display: 'flex', flexDirection: 'column', gap: 8 }}>
          {module.lessons.map((l) => (
            <li key={l.id}>
              <Link
                to={`/lessons/${l.id}`}
                className="card"
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  textDecoration: 'none',
                  color: 'var(--text-primary)',
                  background: 'var(--surface-2)',
                }}
              >
                <span>
                  {l.completed ? '✅' : '📘'} {l.title}
                </span>
                <span style={{ color: 'var(--text-secondary)' }}>+{l.xpReward} XP</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </article>
  );
}
