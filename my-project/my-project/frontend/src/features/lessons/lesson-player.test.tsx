// Exercise widget behavior: typo highlight, expected answer, XP (T042).
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { AttemptFeedback, ListeningOrder, WritingExercise } from './exercise-widgets';
import type { AttemptResult, Exercise } from '../../shared/api/types';

const writing: Exercise = {
  id: 'e1',
  exerciseType: 'translate',
  promptContext: 'Traduza: "Eu tenho uma mala para despachar."',
  options: null,
  audioAssetUrl: null,
  targetSentence: null,
};

const listeningOrder: Exercise = {
  id: 'e2',
  exerciseType: 'listening_order',
  promptContext: 'Monte a frase que você ouviu.',
  options: ['stay', 'how', 'long'],
  audioAssetUrl: '/audio/immigration-howlong.mp3',
  targetSentence: null,
};

function scored(result: Partial<AttemptResult>): AttemptResult {
  return {
    correct: false,
    accuracyScore: 0,
    toleratedTypos: [],
    expectedAnswer: null,
    lessonCompleted: false,
    xpAwarded: 0,
    ...result,
  };
}

describe('WritingExercise', () => {
  it('submits the typed answer', async () => {
    const submit = vi.fn().mockResolvedValue({ kind: 'scored', result: scored({ correct: true }) });
    const onDone = vi.fn();
    render(<WritingExercise exercise={writing} submit={submit} onDone={onDone} busy={false} />);

    await userEvent.type(screen.getByLabelText(/Sua resposta/), 'I have one bag to check');
    await userEvent.click(screen.getByRole('button', { name: 'Verificar' }));

    expect(submit).toHaveBeenCalledWith('I have one bag to check');
    expect(onDone).toHaveBeenCalled();
  });

  it('does not submit an empty answer', () => {
    render(<WritingExercise exercise={writing} submit={vi.fn()} onDone={vi.fn()} busy={false} />);
    expect(screen.getByRole('button', { name: 'Verificar' })).toBeDisabled();
  });
});

describe('ListeningOrder', () => {
  it('offers replayable audio and builds the sentence', async () => {
    const submit = vi.fn().mockResolvedValue({ kind: 'scored', result: scored({ correct: true }) });
    render(<ListeningOrder exercise={listeningOrder} submit={submit} onDone={vi.fn()} busy={false} />);

    expect(screen.getByLabelText(/Áudio do exercício/)).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'how' }));
    await userEvent.click(screen.getByRole('button', { name: 'long' }));
    await userEvent.click(screen.getByRole('button', { name: 'stay' }));
    await userEvent.click(screen.getByRole('button', { name: 'Confirmar' }));

    expect(submit).toHaveBeenCalledWith('how long stay');
  });
});

describe('AttemptFeedback', () => {
  it('highlights tolerated typos on a correct answer (US2 scenario 2)', () => {
    render(
      <AttemptFeedback
        result={scored({ correct: true, accuracyScore: 1, toleratedTypos: ['chek'] })}
        onNext={vi.fn()}
      />,
    );
    expect(screen.getByText(/Correto!/)).toBeInTheDocument();
    expect(screen.getByText('chek')).toHaveClass('tolerated-typo');
  });

  it('shows the expected answer when wrong (US2 scenario 3)', () => {
    render(
      <AttemptFeedback
        result={scored({ correct: false, expectedAnswer: 'I have one bag to check' })}
        onNext={vi.fn()}
      />,
    );
    expect(screen.getByText(/Ainda não/)).toBeInTheDocument();
    expect(screen.getByText('I have one bag to check')).toBeInTheDocument();
  });

  it('advances via the continue button', async () => {
    const onNext = vi.fn();
    render(<AttemptFeedback result={scored({ correct: true })} onNext={onNext} />);
    await userEvent.click(screen.getByRole('button', { name: 'Continuar' }));
    expect(onNext).toHaveBeenCalled();
  });
});
