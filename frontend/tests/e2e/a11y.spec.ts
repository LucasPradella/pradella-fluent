// Accessibility audit (quickstart V11, FR-024): axe-core on the key screens
// in dark theme — zero contrast violations.
import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';
import { completePlacement, register, uniqueEmail } from './helpers';

async function expectNoContrastViolations(page: import('@playwright/test').Page) {
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa'])
    .analyze();
  const contrast = results.violations.filter((v) => v.id === 'color-contrast');
  expect(contrast, JSON.stringify(contrast, null, 2)).toEqual([]);
  return results;
}

test('login screen has no contrast violations', async ({ page }) => {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: 'Entrar' })).toBeVisible();
  await expectNoContrastViolations(page);
});

test('placement screen has no contrast violations', async ({ page }) => {
  await register(page, uniqueEmail('a11y-p'));
  await expect(page.getByRole('heading', { name: 'Teste de nivelamento' })).toBeVisible();
  await expectNoContrastViolations(page);
});

test('dashboard and lesson screens have no contrast violations', async ({ page }) => {
  await register(page, uniqueEmail('a11y-d'));
  await completePlacement(page);
  await page.goto('/dashboard');
  await expect(page.getByRole('heading', { name: 'Seu progresso' })).toBeVisible();
  await expectNoContrastViolations(page);

  await page.goto('/tracks');
  await page.getByRole('link', { name: /Fazendo check-in/ }).click();
  await expect(page.getByRole('heading', { name: 'Fazendo check-in' })).toBeVisible();
  await expectNoContrastViolations(page);
});
