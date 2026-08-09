import { expect, test } from '@playwright/test';

const credentials = { username: 'playwright-admin', password: 'playwright-password' };

async function authenticateEditor(page) {
  const bootstrap = await page.request.post('/api/auth/bootstrap', { data: credentials });
  const response = bootstrap.status() === 409
    ? await page.request.post('/api/login', { data: credentials })
    : bootstrap;
  expect(response.ok()).toBe(true);
}

async function setSafeAreas(page, values) {
  await page.evaluate((safeAreas) => {
    const root = document.documentElement;
    for (const [side, value] of Object.entries(safeAreas)) {
      root.style.setProperty(`--kiss-safe-area-${side}`, `${value}px`);
    }
  }, values);
}

test('mobile content respects portrait and landscape iPhone safe areas', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/');
  await setSafeAreas(page, { top: 59, right: 0, bottom: 34, left: 0 });

  const portrait = await page.locator('.section').evaluate((element) => {
    const style = getComputedStyle(element);
    const toolbar = element.querySelector('.toolbar')?.getBoundingClientRect();
    return {
      paddingTop: style.paddingTop,
      paddingBottom: style.paddingBottom,
      toolbarTop: toolbar?.top ?? 0
    };
  });
  expect(portrait).toEqual({ paddingTop: '71px', paddingBottom: '46px', toolbarTop: 71 });

  await page.setViewportSize({ width: 844, height: 390 });
  await setSafeAreas(page, { top: 0, right: 47, bottom: 21, left: 47 });
  const landscape = await page.locator('.section').evaluate((element) => {
    const style = getComputedStyle(element);
    const rect = element.querySelector('.toolbar')?.getBoundingClientRect();
    return {
      paddingLeft: style.paddingLeft,
      paddingRight: style.paddingRight,
      toolbarLeft: rect?.left ?? 0,
      toolbarRightGap: rect ? window.innerWidth - rect.right : 0,
      overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth
    };
  });
  expect(landscape).toEqual({
    paddingLeft: '63px',
    paddingRight: '63px',
    toolbarLeft: 63,
    toolbarRightGap: 63,
    overflow: false
  });
});

test('editor modals stay clear of the Dynamic Island and home indicator', async ({ page }) => {
  await authenticateEditor(page);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/edit');
  await expect(page.getByRole('button', { name: 'Log Out' })).toBeVisible();
  await setSafeAreas(page, { top: 59, right: 0, bottom: 34, left: 0 });
  await page.getByRole('button', { name: 'Edit button Button 1.1' }).click();

  const geometry = await page.locator('#entryModal').evaluate((modal) => {
    const modalStyle = getComputedStyle(modal);
    const card = modal.querySelector('.modal-card')?.getBoundingClientRect();
    return {
      paddingTop: modalStyle.paddingTop,
      paddingBottom: modalStyle.paddingBottom,
      cardTop: card?.top ?? 0,
      cardBottomGap: card ? window.innerHeight - card.bottom : 0
    };
  });
  expect(geometry.paddingTop).toBe('71px');
  expect(geometry.paddingBottom).toBe('46px');
  expect(geometry.cardTop).toBeGreaterThanOrEqual(71);
  expect(geometry.cardBottomGap).toBeGreaterThanOrEqual(46);
});

test('KISS Brand follows the logo palette while launcher buttons keep cycling', async ({ page }) => {
  await authenticateEditor(page);
  await page.goto('/edit');
  await expect(page.getByRole('button', { name: 'Log Out' })).toBeVisible();
  await page.getByRole('button', { name: 'Show theme editor' }).click();
  const brandSave = page.waitForResponse((response) =>
    response.url().endsWith('/api/config') && response.request().method() === 'POST'
  );
  await page.locator('#themePresetSelect').selectOption('builtin-kiss-brand');
  expect((await brandSave).ok()).toBe(true);

  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#050709');
  await expect(page.locator('#themeButtonColorModeSelect')).toHaveValue('cycle-custom');
  const appearance = await page.evaluate(() => {
    const root = getComputedStyle(document.documentElement);
    const buttons = [...document.querySelectorAll('.entry-preview-button')].slice(0, 2);
    const group = getComputedStyle(document.querySelector('.group-box'));
    const themeEditor = getComputedStyle(document.querySelector('#liveColorEditorPanel'));
    const settingsPanel = getComputedStyle(document.querySelector('.startpage-settings-box'));
    return {
      page: root.getPropertyValue('--startpage-page-bg').trim(),
      group: root.getPropertyValue('--startpage-group-bg').trim(),
      border: root.getPropertyValue('--startpage-group-border').trim(),
      tab: root.getPropertyValue('--startpage-tab-bg').trim(),
      tabHover: root.getPropertyValue('--startpage-tab-hover-bg').trim(),
      activeTab: root.getPropertyValue('--startpage-tab-active-bg').trim(),
      text: root.getPropertyValue('--startpage-text-color').trim(),
      groupBorder: `${group.borderTopWidth} ${group.borderTopStyle} ${group.borderTopColor}`,
      themeEditorBorder: `${themeEditor.borderTopWidth} ${themeEditor.borderTopStyle} ${themeEditor.borderTopColor}`,
      settingsPanelBorder: `${settingsPanel.borderTopWidth} ${settingsPanel.borderTopStyle} ${settingsPanel.borderTopColor}`,
      radii: [
        group.borderTopLeftRadius,
        settingsPanel.borderTopLeftRadius,
        getComputedStyle(document.querySelector('.group-add-panel')).borderTopLeftRadius
      ],
      buttonColors: buttons.map((button) => button.style.getPropertyValue('--entry-btn-base'))
    };
  });
  expect(appearance).toEqual({
    page: '#050709',
    group: '#12171a',
    border: '#2a3138',
    tab: '#151b20',
    tabHover: '#1b2329',
    activeTab: '#00cbe5',
    text: '#f2f4f5',
    groupBorder: '1px solid rgb(42, 49, 56)',
    themeEditorBorder: '1px solid rgb(42, 49, 56)',
    settingsPanelBorder: '1px solid rgb(42, 49, 56)',
    radii: ['13.6px', '13.6px', '13.6px'],
    buttonColors: ['hsl(0, 70%, 74%)', 'hsl(15, 70%, 74%)']
  });

  await page.reload();
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#050709');

  await page.getByRole('button', { name: 'Show theme editor' }).click();
  const defaultSave = page.waitForResponse((response) =>
    response.url().endsWith('/api/config') && response.request().method() === 'POST'
  );
  await page.locator('#themePresetSelect').selectOption('builtin-default-theme');
  expect((await defaultSave).ok()).toBe(true);
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#101728');
});
