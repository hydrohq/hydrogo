import { test, expect } from '@playwright/test';
import fs from 'node:fs/promises';
import path from 'node:path';

for (const collection of ['project', 'article']) {
  test(`${collection} generated form and table render in a browser`, async ({ page }) => {
    const root = path.resolve(process.cwd(), '..', 'examples', `${collection}-edition`, 'generated', collection);
    const [table, form] = await Promise.all([
      fs.readFile(path.join(root, 'table.html'), 'utf8'),
      fs.readFile(path.join(root, 'form.html'), 'utf8'),
    ]);
    await page.setContent(`<main>${table}${form}</main>`);
    await expect(page.locator(`[data-hydro-table="${collection}"] table`)).toBeVisible();
    await expect(page.locator(`[data-hydro-form="${collection}"]`)).toBeVisible();
    await expect(page.locator(`[data-hydro-form="${collection}"] input[required]`)).not.toHaveCount(0);
    await expect(page.locator('script')).toHaveCount(0);
  });
}

