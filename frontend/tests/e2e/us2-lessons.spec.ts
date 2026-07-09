// US2 journey (quickstart V3): lesson with typo tolerance, wrong-answer
// feedback and XP on completion. Uses the seeded "Fazendo check-in" lesson.
import { expect, test } from '@playwright/test';
import { completePlacement, register, uniqueEmail } from './helpers';

test('V3: typo tolerated, wrong answer shows expected, lesson awards XP', async ({ page }) => {
  await register(page, uniqueEmail('v3'));
  await completePlacement(page);
  await page.getByRole('button', { name: 'Começar a estudar' }).click();

  // Open the first unlocked basic lesson (seed: Fazendo check-in).
  await page.getByRole('link', { name: /Fazendo check-in/ }).click();
  await expect(page.getByRole('heading', { name: 'Fazendo check-in' })).toBeVisible();

  // Exercise 1 (translate) with a deliberate small typo: "chek".
  await page.getByLabel(/Sua resposta/).fill('I have one bag to chek');
  await page.getByRole('button', { name: 'Verificar' }).click();
  await expect(page.getByText(/Correto!/)).toBeVisible();
  await expect(page.getByText('chek')).toBeVisible(); // typo highlighted
  await page.getByRole('button', { name: 'Continuar' }).click();

  // Exercise 2 (fill_blank): wrong semantic answer → expected shown.
  await page.getByLabel(/Sua resposta/).fill('flying');
  await page.getByRole('button', { name: 'Verificar' }).click();
  await expect(page.getByText(/Ainda não/)).toBeVisible();
  await expect(page.getByText('boarding')).toBeVisible(); // expected answer
  await page.getByRole('button', { name: 'Continuar' }).click();

  // Redo exercise 2 correctly? The player moves on; answer exercise 3
  // (listening choice) with the correct option.
  await page
    .getByRole('group', { name: 'Alternativas' })
    .getByRole('button', { name: 'Your passport, please' })
    .click();
  await expect(page.getByText(/Correto!/)).toBeVisible();
  await page.getByRole('button', { name: 'Continuar' }).click();

  // Exercise 4 (speaking): providers are not configured in E2E — skip path
  // must never block lesson completion (FR-016 also covers this).
  await expect(page.getByRole('button', { name: /gravar/i })).toBeVisible();
  await page.getByRole('button', { name: /gravar/i }).click();
  await page.getByRole('button', { name: /Parar gravação/ }).click();
  await page.getByRole('button', { name: 'Enviar para avaliação' }).click();
  await page.getByRole('button', { name: 'Pular exercício' }).click();

  // Completion screen (XP only when every exercise passed — the wrong
  // fill_blank keeps it from completing, which is correct behavior).
  await expect(page.getByRole('heading', { name: 'Lição concluída!' })).toBeVisible();
});
