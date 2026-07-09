// Streak counter (FR-018).
export function Streak({ current, longest }: { current: number; longest: number }) {
  return (
    <div className="card" style={{ display: 'flex', gap: 24 }}>
      <div>
        <div style={{ fontSize: 32, fontWeight: 700 }} aria-label={`Sequência atual: ${current} dias`}>
          🔥 {current}
        </div>
        <div style={{ color: 'var(--text-secondary)', fontSize: 14 }}>dias seguidos</div>
      </div>
      <div>
        <div style={{ fontSize: 32, fontWeight: 700 }} aria-label={`Maior sequência: ${longest} dias`}>
          🏆 {longest}
        </div>
        <div style={{ color: 'var(--text-secondary)', fontSize: 14 }}>recorde</div>
      </div>
    </div>
  );
}
