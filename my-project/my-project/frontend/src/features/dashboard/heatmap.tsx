// 90-day activity heatmap as a CSS grid with 5 saturation buckets and
// accessible labels (FR-019).
import type { HeatmapDay } from '../../shared/api/types';

const levelColors = ['var(--heat-0)', 'var(--heat-1)', 'var(--heat-2)', 'var(--heat-3)', 'var(--heat-4)'];

function formatDate(iso: string): string {
  const [y, m, d] = iso.split('-');
  return `${d}/${m}/${y}`;
}

export function Heatmap({ days }: { days: HeatmapDay[] }) {
  return (
    <div>
      <div
        role="img"
        aria-label={`Mapa de atividade dos últimos ${days.length} dias`}
        style={{
          display: 'grid',
          gridTemplateRows: 'repeat(7, 14px)',
          gridAutoFlow: 'column',
          gap: 3,
          justifyContent: 'start',
        }}
        data-testid="heatmap"
      >
        {days.map((day) => (
          <div
            key={day.date}
            data-level={day.level}
            title={`${formatDate(day.date)}: ${day.interactions} ${day.interactions === 1 ? 'interação' : 'interações'}`}
            style={{
              width: 14,
              height: 14,
              borderRadius: 3,
              background: levelColors[day.level] ?? levelColors[0],
            }}
          />
        ))}
      </div>
      <div
        aria-hidden
        style={{ display: 'flex', gap: 4, alignItems: 'center', marginTop: 8, fontSize: 12, color: 'var(--text-secondary)' }}
      >
        Menos
        {levelColors.map((c) => (
          <span key={c} style={{ width: 12, height: 12, borderRadius: 3, background: c, display: 'inline-block' }} />
        ))}
        Mais
      </div>
    </div>
  );
}
