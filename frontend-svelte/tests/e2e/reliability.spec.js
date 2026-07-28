import { expect, test } from '@playwright/test';

async function openHomelab(page) {
  await page.getByRole('button', { name: 'Homelab' }).click();
  await expect(page.locator('[data-group-id="group-arr-stack"]')).toBeVisible();
  await expect(page.locator('[data-group-id="group-arr-stack"] [data-button-id]')).toHaveCount(8);
}

async function authenticateEditor(page) {
  const credentials = { username: 'playwright-admin', password: 'playwright-password' };
  const bootstrap = await page.request.post('/api/auth/bootstrap', { data: credentials });
  const response = bootstrap.status() === 409
    ? await page.request.post('/api/login', { data: credentials })
    : bootstrap;
  expect(response.ok()).toBe(true);
}

test('all Homelab groups survive repeated Brave/Chromium reloads', async ({ page }) => {
  await page.goto('/');
  await openHomelab(page);

  for (let attempt = 0; attempt < 25; attempt += 1) {
    await page.reload();
    await expect(page.locator('#startpageContent [data-group-id]')).toHaveCount(7);
    await expect(page.locator('[data-group-id="group-arr-stack"]')).toBeVisible();
    await expect(page.locator('[data-group-id="group-arr-stack"] [data-button-id]')).toHaveCount(8);
  }
});

test('all groups survive repeated tab switches', async ({ page }) => {
  await page.goto('/');

  for (let attempt = 0; attempt < 25; attempt += 1) {
    await openHomelab(page);
    await page.getByRole('button', { name: 'Frequent' }).click();
    await expect(page.locator('#startpageContent [data-group-id]')).toHaveCount(7);
  }
});

test('an API failure shows Retry and never renders demo data', async ({ page }) => {
  let failConfig = true;
  await page.route('**/api/config', async (route) => {
    if (failConfig) {
      await route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'temporary failure' })
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/');
  await expect(page.getByText('The startpage configuration could not be loaded.')).toBeVisible();
  await expect(page.locator('[data-group-id]')).toHaveCount(0);

  failConfig = false;
  await page.getByRole('button', { name: 'Try again' }).click();
  await openHomelab(page);
  await expect(page.locator('#startpageContent [data-group-id]')).toHaveCount(7);
});

test('the editor searches enabled providers without dropdowns and persists the selection', async ({ page }) => {
  await authenticateEditor(page);
  await page.route('**/api/icons/search?**', async (route) => {
    const requestUrl = new URL(route.request().url());
    if (!requestUrl.searchParams.get('q') && requestUrl.searchParams.get('externalUrl')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          query: '',
          enabledProviders: ['iconify', 'selfhst', 'website'],
          groups: [{
            provider: 'website',
            label: 'Website icon',
            status: 'ready',
            message: 'Found 1 icon(s).',
            items: [{
              provider: 'website',
              source: 'website',
              name: 'example.com',
              reference: 'https://example.com/favicon.ico',
              category: 'Website icon',
              previewUrl: ''
            }]
          }]
        })
      });
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        query: 'home',
        enabledProviders: ['iconify', 'selfhst'],
        groups: [
          {
            provider: 'iconify',
            label: 'Iconify',
            status: 'ready',
            message: 'Found 1 icon(s).',
            items: [{
              provider: 'iconify',
              source: 'iconify-lucide',
              name: 'Home',
              reference: 'lucide:home',
              category: 'Lucide',
              license: 'ISC',
              previewUrl: ''
            }]
          },
          {
            provider: 'selfhst',
            label: 'selfh.st/icons',
            status: 'empty',
            message: 'No icons found.',
            items: []
          }
        ]
      })
    });
  });

  await page.goto('/edit');
  await expect(page.getByRole('button', { name: 'Log Out' })).toBeVisible();
  await page.getByRole('button', { name: 'Add button' }).first().click();
  await expect(page.locator('#entryIconSearchStatus')).toHaveText('Enter at least 2 characters, or enter a URL for its website icon.');
  await expect(page.locator('#entryIconSearchResults .icon-search-group')).toHaveCount(0);
  await expect(page.getByText('No icons found.', { exact: true })).toHaveCount(0);
  await expect(page.locator('.icon-provider-option')).toHaveCount(6);
  await page.getByRole('button', { name: 'Cancel', exact: true }).click();

  await page.getByRole('button', { name: 'Edit button Button 1.1' }).click();

  await expect(page.locator('#entryIconProviderSelect')).toHaveCount(0);
  await expect(page.locator('#entryIconCollectionSelect')).toHaveCount(0);
  const providerOptions = page.locator('.icon-provider-option');
  await expect(providerOptions).toHaveCount(6);
  await expect(providerOptions).toContainText([
    'Iconify 14 collections',
    'selfh.st/icons',
    'Dashboard Icons',
    'Local icons',
    'Website icon',
    'Wikimedia Commons'
  ]);

  const wikimedia = providerOptions.filter({ hasText: 'Wikimedia Commons' }).locator('input');
  const preferenceResponse = page.waitForResponse((response) =>
    response.url().endsWith('/api/icons/preferences') && response.request().method() === 'POST'
  );
  await wikimedia.uncheck();
  expect((await preferenceResponse).ok()).toBe(true);
  await page.getByRole('button', { name: 'Cancel', exact: true }).click();
  await page.getByRole('button', { name: 'Edit button Button 1.1' }).click();
  await expect(providerOptions.filter({ hasText: 'Wikimedia Commons' }).locator('input')).not.toBeChecked();

  await page.locator('#entryExternalUrlInput').fill('example.com/service');
  await expect(page.locator('.icon-search-group').filter({ hasText: 'Website icon' })).toBeVisible();
  await page.locator('#entryIconSearchInput').fill('home');
  await expect(page.locator('.icon-search-group').filter({ hasText: 'Iconify' })).toContainText('lucide:home · Lucide · ISC');
  await expect(page.locator('.icon-search-group').filter({ hasText: 'selfh.st/icons' })).toHaveCount(0);
  await expect(page.locator('.icon-search-group')).toHaveCount(1);
  const groupStyles = await page.locator('.icon-search-group').evaluate((element) => {
    const group = getComputedStyle(element);
    const container = getComputedStyle(element.parentElement);
    return {
      borderWidth: group.borderTopWidth,
      backgroundColor: group.backgroundColor,
      maxHeight: container.maxHeight,
      overflowY: container.overflowY
    };
  });
  expect(groupStyles).toEqual({
    borderWidth: '0px',
    backgroundColor: 'rgba(0, 0, 0, 0)',
    maxHeight: 'none',
    overflowY: 'visible'
  });
});

test('admin action buttons follow light and dark theme presets', async ({ page }) => {
  await authenticateEditor(page);
  await page.goto('/edit');
  await expect(page.getByRole('button', { name: 'Log Out' })).toBeVisible();
  await page.getByRole('button', { name: 'Show theme editor' }).click();

  await page.locator('#themePresetSelect').selectOption('builtin-paper-ink');
  await page.getByRole('button', { name: 'Edit button Button 1.1' }).click();
  const paperStyles = await page.evaluate(() => ({
    cancelBg: getComputedStyle(document.querySelector('#entryCancelBtn')).backgroundColor,
    cancelText: getComputedStyle(document.querySelector('#entryCancelBtn')).color,
    saveBg: getComputedStyle(document.querySelector('#entrySaveBtn')).backgroundColor,
    saveText: getComputedStyle(document.querySelector('#entrySaveBtn')).color,
    deleteBg: getComputedStyle(document.querySelector('#entryDeleteBtn')).backgroundColor
  }));
  expect(paperStyles).toEqual({
    cancelBg: 'rgb(201, 207, 216)',
    cancelText: 'rgb(15, 23, 42)',
    saveBg: 'rgb(15, 23, 42)',
    saveText: 'rgb(248, 250, 252)',
    deleteBg: 'rgb(220, 38, 38)'
  });
  await page.getByRole('button', { name: 'Cancel', exact: true }).click();

  await page.locator('#themePresetSelect').selectOption('builtin-sunset-control');
  await page.getByRole('button', { name: 'Edit button Button 1.1' }).click();
  const sunsetStyles = await page.evaluate(() => ({
    cancelBg: getComputedStyle(document.querySelector('#entryCancelBtn')).backgroundColor,
    cancelText: getComputedStyle(document.querySelector('#entryCancelBtn')).color,
    saveBg: getComputedStyle(document.querySelector('#entrySaveBtn')).backgroundColor,
    saveText: getComputedStyle(document.querySelector('#entrySaveBtn')).color
  }));
  expect(sunsetStyles).toEqual({
    cancelBg: 'rgb(68, 49, 79)',
    cancelText: 'rgb(248, 250, 252)',
    saveBg: 'rgb(249, 115, 22)',
    saveText: 'rgb(31, 10, 4)'
  });
});
