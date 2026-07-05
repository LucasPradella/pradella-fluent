// US4 journey (quickstart V7): streak + heatmap rendering and the spaced
// review loop logging is_review=true. Multi-day time travel is exercised in
// backend unit tests (streak_test.go); here the dashboard aggregate is
// mocked to a two-day history and the review round-trip runs for real.
import { expect, test } from '@playwright/test';
import { completePlacement, register, uniqueEmail } from './helpers';

// page.route mocks must not be swallowed by the Workbox service worker.
test.use({ serviceWorkers: 'block' });

function twoDayDashboard() {
  const heatmap = [];
  const today = new Date();
  for (let i = 89; i >= 0; i--) {
    const d = new Date(today.getTime() - i * 86_400_000);
    const interactions = i === 0 ? 3 : i === 1 ? 1 : 0;
    heatmap.push({
      date: d.toISOString().slice(0, 10),
      interactions,
      level: interactions === 0 ? 0 : interactions <= 2 ? 1 : 2,
    });
  }
  return {
    currentStreak: 2,
    longestStreak: 2,
    totalXp: 50,
    heatmap,
    dueReviews: 1,
  };
}

test('V7: streak of 2 renders, heatmap saturates with volume, review entry visible', async ({
  page,
}) => {
  await page.route('**/api/v1/dashboard', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(twoDayDashboard()),
    }),
  );

  await register(page, uniqueEmail('v7'));
  await completePlacement(page);
  await page.goto('/dashboard');

  await expect(page.getByLabel('Sequência atual: 2 dias')).toBeVisible();
  const grid = page.getByTestId('heatmap');
  await expect(grid.locator('[data-level="2"]')).toHaveCount(1);
  await expect(grid.locator('[data-level="1"]')).toHaveCount(1);
  await expect(page.getByRole('button', { name: 'Revisar agora' })).toBeVisible();
});

test('V7b: failed exercise resurfaces in review and completes with is_review', async ({
  page,
}) => {
  await register(page, uniqueEmail('v7b'));
  await completePlacement(page);
  await page.getByRole('button', { name: 'Começar a estudar' }).click();

  // Fail the first writing exercise → item enters the review queue (1 day).
  await page.getByRole('link', { name: /Fazendo check-in/ }).click();
  await page.getByLabel(/Sua resposta/).fill('completely wrong sentence here');
  await page.getByRole('button', { name: 'Verificar' }).click();
  await expect(page.getByText(/Ainda não/)).toBeVisible();

  // The item is due tomorrow, so today's queue shows the empty state —
  // mock the queue as due-now to run the review UI loop end to end.
  await page.route('**/api/v1/review-queue', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          id: '11111111-1111-7111-8111-111111111111',
          dueAt: new Date().toISOString(),
          failureCount: 1,
          exercise: {
            id: '22222222-2222-7222-8222-222222222222',
            exerciseType: 'translate',
            promptContext: 'Traduza: "Eu tenho uma mala para despachar."',
            options: null,
            audioAssetUrl: null,
            targetSentence: null,
          },
        },
      ]),
    }),
  );
  let reviewFlagSent = false;
  await page.route('**/api/v1/exercises/*/attempts', async (route) => {
    const body = route.request().postDataJSON() as { isReview?: boolean };
    reviewFlagSent = body.isReview === true;
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({
        correct: true,
        accuracyScore: 1,
        toleratedTypos: [],
        expectedAnswer: null,
        lessonCompleted: false,
        xpAwarded: 0,
      }),
    });
  });

  await page.goto('/review');
  await page.getByLabel(/Sua resposta/).fill('I have one bag to check');
  await page.getByRole('button', { name: 'Verificar' }).click();
  await page.getByRole('button', { name: 'Continuar' }).click();

  await expect(page.getByRole('heading', { name: 'Revisão concluída!' })).toBeVisible();
  expect(reviewFlagSent, 'review completions must send isReview=true (FR-020)').toBe(true);
});
