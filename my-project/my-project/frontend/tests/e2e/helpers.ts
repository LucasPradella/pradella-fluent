import { expect, type Page } from '@playwright/test';

/** uniqueEmail avoids collisions across runs against the shared dev DB. */
export function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e6)}@e2e.test`;
}

export const PASSWORD = 'senha-super-secreta-123';

/** register signs a fresh user up through the real UI. */
export async function register(page: Page, email: string, name = 'Dev E2E'): Promise<void> {
  await page.goto('/register');
  await page.getByLabel('Nome').fill(name);
  await page.getByLabel('E-mail').fill(email);
  await page.getByLabel(/^Senha/).fill(PASSWORD);
  await page.getByRole('button', { name: /Criar conta/ }).click();
  await expect(page).toHaveURL(/\/placement/, { timeout: 15_000 });
}

function resultHeading(page: Page) {
  return page.getByRole('heading', { name: /Seu nível/ });
}

/**
 * completePlacement answers questions until the result screen appears.
 * Answers are arbitrary — the assigned level is irrelevant to callers.
 */
export async function completePlacement(page: Page): Promise<void> {
  await page.getByRole('button', { name: 'Começar o teste' }).click();

  for (let i = 0; i < 14; i++) {
    if (await resultHeading(page).isVisible().catch(() => false)) break;
    await answerCurrentQuestion(page);
  }
  await expect(resultHeading(page)).toBeVisible({ timeout: 15_000 });
}

/**
 * answerCurrentQuestion submits one answer (choice or order question) and
 * waits for the server response, so the next question is rendered before
 * returning.
 */
export async function answerCurrentQuestion(page: Page): Promise<void> {
  await expect(page.getByText(/Pergunta \d+ de 12/)).toBeVisible({ timeout: 15_000 });

  const responsePromise = page.waitForResponse(
    (r) => r.url().includes('/placement/session/answers') && r.request().method() === 'POST',
    { timeout: 15_000 },
  );

  const confirm = page.getByRole('button', { name: 'Confirmar' });
  if (await confirm.isVisible().catch(() => false)) {
    // Order question: tap word blocks until the bank is empty, confirm.
    for (let guard = 0; guard < 16; guard++) {
      if (await confirm.isEnabled()) break;
      await page
        .locator('button.secondary')
        .filter({ hasNotText: 'Desfazer' })
        .first()
        .click();
    }
    await confirm.click();
  } else {
    await page.getByRole('group', { name: 'Alternativas' }).getByRole('button').first().click();
  }

  const response = await responsePromise;
  expect(response.status(), 'answer must be accepted').toBe(200);
  // Let React commit the next state before the caller inspects the page.
  await page.waitForTimeout(50);
}
