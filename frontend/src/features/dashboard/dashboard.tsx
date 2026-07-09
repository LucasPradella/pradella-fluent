// Dashboard (US4): streak, XP, 90-day heatmap, due-reviews entry point.
import { Link } from 'react-router-dom';
import { useDashboard } from '../../shared/api/hooks';
import { Heatmap } from './heatmap';
import { Streak } from './streak';

export function DashboardPage() {
  const { data, isLoading, isError } = useDashboard();

  if (isLoading) return <p>Carregando painel…</p>;
  if (isError || !data) {
    return (
      <p role="alert" style={{ color: 'var(--error)' }}>
        Não foi possível carregar o painel. Verifique sua conexão.
      </p>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <h1>Seu progresso</h1>

      <Streak current={data.currentStreak} longest={data.longestStreak} />

      <div className="card">
        <div style={{ fontSize: 32, fontWeight: 700 }}>⭐ {data.totalXp} XP</div>
        <div style={{ color: 'var(--text-secondary)', fontSize: 14 }}>experiência acumulada</div>
      </div>

      <section className="card" aria-label="Atividade dos últimos 90 dias">
        <h2 style={{ marginTop: 0, fontSize: 18 }}>Últimos 90 dias</h2>
        <Heatmap days={data.heatmap} />
      </section>

      <section className="card" aria-label="Revisões pendentes">
        <h2 style={{ marginTop: 0, fontSize: 18 }}>Revisão espaçada</h2>
        {data.dueReviews > 0 ? (
          <>
            <p>
              Você tem <strong>{data.dueReviews}</strong>{' '}
              {data.dueReviews === 1 ? 'item para revisar' : 'itens para revisar'} — revisar agora
              fortalece a memória.
            </p>
            <Link to="/review">
              <button>Revisar agora</button>
            </Link>
          </>
        ) : (
          <p style={{ color: 'var(--text-secondary)' }}>Nenhuma revisão pendente. 🎯</p>
        )}
      </section>
    </div>
  );
}
