// US1 journey (quickstart V1 + V2): register → adaptive placement →
// level assigned → resume support.
import { expect, test } from '@playwright/test';
import { completePlacement, register, uniqueEmail, PASSWORD } from './helpers';

test('V1: register, complete placement in <=12 questions, see level and locked tracks', async ({
  page,
}) => {
  const email = uniqueEmail('v1');
  await register(page, email);

  // The test starts immediately after signup (FR-002).
  await expect(page.getByRole('heading', { name: 'Teste de nivelamento' })).toBeVisible();
  await completePlacement(page);

  // Result: level + track lock overview (FR-005, FR-006).
  await expect(page.getByRole('heading', { name: /Seu nível/ })).toContainText(
    /Básico|Intermediário|Avançado/,
  );
  await expect(page.getByLabel('Bloqueada').first()).toBeVisible();
  await expect(page.getByLabel('Desbloqueada').first()).toBeVisible();

  await page.getByRole('button', { name: 'Começar a estudar' }).click();
  await expect(page).toHaveURL(/\/tracks/);
});

test('V2: abandoning after 5 answers resumes at question 6', async ({ page }) => {
  const email = uniqueEmail('v2');
  await register(page, email);
  await page.getByRole('button', { name: 'Começar o teste' }).click();

  const { answerCurrentQuestion } = await import('./helpers');
  for (let i = 0; i < 5; i++) {
    await answerCurrentQuestion(page);
  }

  // Simulate quitting: fresh navigation + fresh login session.
  await page.goto('/login');
  await page.getByLabel('E-mail').fill(email);
  await page.getByLabel(/^Senha/).fill(PASSWORD);
  await page.getByRole('button', { name: 'Entrar' }).click();

  // Unplaced user lands back on placement, at question 6 (US1 scenario 6).
  await expect(page).toHaveURL(/\/placement/);
  await expect(page.getByText('Pergunta 6 de 12')).toBeVisible();
});
