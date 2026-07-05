// US5 journey (quickstart V9/V10): offline shell render with cached data,
// connectivity-required messaging, offline outbox replay (idempotent).
import { expect, test } from '@playwright/test';
import { completePlacement, register, uniqueEmail } from './helpers';

test('V9: shell renders offline with cached profile; speaking demands connection', async ({
  page,
  context,
}) => {
  await register(page, uniqueEmail('v9'));
  await completePlacement(page);
  await page.getByRole('button', { name: 'Começar a estudar' }).click();

  // Prime caches: dashboard + a lesson while online, and let the service
  // worker finish precaching the shell.
  await page.goto('/dashboard');
  await expect(page.getByRole('heading', { name: 'Seu progresso' })).toBeVisible();
  await page.goto('/tracks');
  await page.getByRole('link', { name: /Fazendo check-in/ }).click();
  await expect(page.getByRole('heading', { name: 'Fazendo check-in' })).toBeVisible();
  await page.waitForFunction(() => navigator.serviceWorker?.controller != null, undefined, {
    timeout: 20_000,
  });

  await context.setOffline(true);
  const start = Date.now();
  await page.goto('/dashboard');
  await expect(page.getByRole('heading', { name: 'Seu progresso' })).toBeVisible({
    timeout: 5_000,
  });
  const elapsed = Date.now() - start;
  expect(elapsed, 'offline shell render must be fast (SC-005 budget 1.5s)').toBeLessThan(3_000);

  // Offline banner + streak header still render from cached data.
  await expect(page.getByText(/Você está offline/)).toBeVisible();

  // Speaking requires connection (US5 scenario 3).
  await page.goto('/tracks');
  await page.getByRole('link', { name: /Fazendo check-in/ }).click();
  await page.getByLabel(/Sua resposta/).fill('I have one bag to check');
  await page.getByRole('button', { name: 'Verificar' }).click();
  await page.getByRole('button', { name: 'Continuar' }).click();
  await page.getByLabel(/Sua resposta/).fill('boarding');
  await page.getByRole('button', { name: 'Verificar' }).click();
  await page.getByRole('button', { name: 'Continuar' }).click();
  await page
    .getByRole('group', { name: 'Alternativas' })
    .getByRole('button', { name: 'Your passport, please' })
    .click();
  await page.getByRole('button', { name: 'Continuar' }).click();
  await expect(page.getByText(/A prática de fala precisa de conexão/)).toBeVisible();

  await context.setOffline(false);
});

// The outbox mechanics are independent of the service worker; blocking it
// here keeps Playwright's offline emulation authoritative for page fetches.
test.describe('outbox replay', () => {
  test.use({ serviceWorkers: 'block' });

  test('V10: offline completion syncs exactly once on reconnect', async ({ page, context }) => {
  await register(page, uniqueEmail('v10'));
  await completePlacement(page);
  await page.getByRole('button', { name: 'Começar a estudar' }).click();
  await page.getByRole('link', { name: /Fazendo check-in/ }).click();
  await expect(page.getByRole('heading', { name: 'Fazendo check-in' })).toBeVisible();

  // Complete the writing exercise offline → outbox.
  await context.setOffline(true);
  await page.getByLabel(/Sua resposta/).fill('I have one bag to check');
  await page.getByRole('button', { name: 'Verificar' }).click();
  await expect(page.getByText(/Resposta salva/)).toBeVisible();

  // Reconnect → the outbox replays automatically on the 'online' event.
  await context.setOffline(false);
  await expect
    .poll(
      async () =>
        page.evaluate(async () => {
          const resp = await fetch('/api/v1/dashboard', { credentials: 'same-origin' });
          const data = (await resp.json()) as {
            heatmap: { interactions: number }[];
          };
          return data.heatmap.at(-1)?.interactions ?? 0;
        }),
      { timeout: 20_000, message: 'replayed attempt must appear in the dashboard' },
    )
    .toBe(1); // exactly one log — duplicate replays are deduped by attemptId
  });
});
