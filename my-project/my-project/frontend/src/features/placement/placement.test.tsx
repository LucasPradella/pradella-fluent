// Question-type rendering and result states (US1 — T031).
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { QuestionCard } from './question-card';
import type { NextQuestion } from '../../shared/api/types';

const choice: NextQuestion = {
  id: 'q1',
  questionType: 'choice',
  prompt: 'Complete: "I ___ from Brazil."',
  options: ['am', 'is', 'are'],
  audioAssetUrl: null,
};

const order: NextQuestion = {
  id: 'q2',
  questionType: 'order',
  prompt: 'Ordene as palavras.',
  options: ['name', 'my', 'is', 'Ana'],
  audioAssetUrl: null,
};

describe('QuestionCard', () => {
  it('renders choice options and submits the tapped one', async () => {
    const onAnswer = vi.fn();
    render(<QuestionCard question={choice} onAnswer={onAnswer} disabled={false} />);

    expect(screen.getByText(/I ___ from Brazil/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'am' }));
    expect(onAnswer).toHaveBeenCalledWith('am');
  });

  it('disables options while an answer is in flight', () => {
    render(<QuestionCard question={choice} onAnswer={vi.fn()} disabled />);
    for (const opt of choice.options) {
      expect(screen.getByRole('button', { name: opt })).toBeDisabled();
    }
  });

  it('builds an ordered sentence from word blocks and confirms', async () => {
    const onAnswer = vi.fn();
    render(<QuestionCard question={order} onAnswer={onAnswer} disabled={false} />);

    // Confirm stays disabled until every word is used.
    expect(screen.getByRole('button', { name: 'Confirmar' })).toBeDisabled();

    await userEvent.click(screen.getByRole('button', { name: 'my' }));
    await userEvent.click(screen.getByRole('button', { name: 'name' }));
    await userEvent.click(screen.getByRole('button', { name: 'is' }));
    await userEvent.click(screen.getByRole('button', { name: 'Ana' }));

    const confirm = screen.getByRole('button', { name: 'Confirmar' });
    expect(confirm).toBeEnabled();
    await userEvent.click(confirm);
    expect(onAnswer).toHaveBeenCalledWith('my name is Ana');
  });

  it('undo removes the last picked word', async () => {
    render(<QuestionCard question={order} onAnswer={vi.fn()} disabled={false} />);

    await userEvent.click(screen.getByRole('button', { name: 'my' }));
    await userEvent.click(screen.getByRole('button', { name: 'name' }));
    await userEvent.click(screen.getByRole('button', { name: 'Desfazer' }));

    // "name" returns to the word bank.
    expect(screen.getByRole('button', { name: 'name' })).toBeInTheDocument();
  });
});
