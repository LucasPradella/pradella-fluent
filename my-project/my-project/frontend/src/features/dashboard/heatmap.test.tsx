// Heatmap rendering: 90 buckets, 5 saturation levels, accessible labels (T063).
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Heatmap } from './heatmap';
import type { HeatmapDay } from '../../shared/api/types';

function makeDays(): HeatmapDay[] {
  const days: HeatmapDay[] = [];
  const start = new Date('2026-04-06T00:00:00Z');
  for (let i = 0; i < 90; i++) {
    const d = new Date(start.getTime() + i * 86_400_000);
    const interactions = i % 12;
    const level = (interactions === 0 ? 0 : interactions <= 2 ? 1 : interactions <= 5 ? 2 : interactions <= 9 ? 3 : 4) as HeatmapDay['level'];
    days.push({ date: d.toISOString().slice(0, 10), interactions, level });
  }
  return days;
}

describe('Heatmap', () => {
  it('renders one cell per day with its saturation level', () => {
    const days = makeDays();
    render(<Heatmap days={days} />);

    const grid = screen.getByTestId('heatmap');
    expect(grid.children).toHaveLength(90);
    expect(grid).toHaveAccessibleName(/últimos 90 dias/i);

    const levels = new Set<string>();
    for (const cell of Array.from(grid.children)) {
      levels.add((cell as HTMLElement).dataset.level!);
    }
    expect(levels).toEqual(new Set(['0', '1', '2', '3', '4']));
  });

  it('labels each day with date and interaction count', () => {
    render(<Heatmap days={makeDays()} />);
    const grid = screen.getByTestId('heatmap');
    const first = grid.children[0] as HTMLElement;
    expect(first.title).toMatch(/06\/04\/2026: 0 interações/);
  });
});
