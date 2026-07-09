// Lesson completion celebration with XP (FR-011).
import { Link } from 'react-router-dom';
import type { Lesson } from '../../shared/api/types';

export function LessonCompletion({ lesson, xpAwarded }: { lesson: Lesson; xpAwarded: number }) {
  return (
    <div
      className="card"
      role="status"
      style={{ textAlign: 'center', display: 'flex', flexDirection: 'column', gap: 16, padding: 32 }}
    >
      <div aria-hidden style={{ fontSize: 56 }}>
        🎉
      </div>
      <h1>Lição concluída!</h1>
      <p>
        Você terminou <strong>{lesson.title}</strong>.
      </p>
      {xpAwarded > 0 ? (
        <p style={{ color: 'var(--success)', fontSize: 24, fontWeight: 700 }}>+{xpAwarded} XP</p>
      ) : (
        <p style={{ color: 'var(--text-secondary)' }}>Revisão concluída — continue praticando!</p>
      )}
      <div style={{ display: 'flex', gap: 8, justifyContent: 'center' }}>
        <Link to="/tracks">
          <button className="secondary">Voltar às trilhas</button>
        </Link>
        <Link to="/dashboard">
          <button>Ver meu progresso</button>
        </Link>
      </div>
    </div>
  );
}
