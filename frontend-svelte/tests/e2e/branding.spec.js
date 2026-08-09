import { expect, test } from '@playwright/test';
const iconCases = [
  ['/icons/apple-touch-icon.png?v=2.8.0', 180],
  ['/icons/kiss-startpage-icon-192.png?v=2.8.0', 192],
  ['/icons/kiss-startpage-icon-512.png?v=2.8.0', 512],
  ['/icons/kiss-startpage-icon-maskable-512.png?v=2.8.0', 512],
];

test('publishes complete standalone app branding', async ({ page, request }) => {
  await page.goto('/');

  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#0f172a');
  await expect(page.locator('meta[name="mobile-web-app-capable"]')).toHaveAttribute('content', 'yes');
  await expect(page.locator('meta[name="apple-mobile-web-app-capable"]')).toHaveAttribute('content', 'yes');
  await expect(page.locator('meta[name="apple-mobile-web-app-title"]')).toHaveAttribute('content', 'KISS');
  await expect(page.locator('link[rel="manifest"]')).toHaveAttribute('href', '/manifest.webmanifest');
  await expect(page.locator('link[rel="icon"]')).toHaveAttribute('href', '/brand/kiss-startpage-logo.svg?v=2.8.0');
  await expect(page.locator('link[rel="apple-touch-icon"]')).toHaveAttribute('href', '/icons/apple-touch-icon.png?v=2.8.0');

  const manifestResponse = await request.get('/manifest.webmanifest');
  expect(manifestResponse.ok()).toBe(true);
  expect(manifestResponse.headers()['content-type']).toBe('application/manifest+json; charset=utf-8');
  expect(await manifestResponse.json()).toEqual({
    name: 'KISS Startpage',
    short_name: 'KISS',
    description: 'A minimal self-hosted start page',
    start_url: '/',
    scope: '/',
    display: 'standalone',
    background_color: '#050709',
    theme_color: '#0d1216',
    icons: [
      { src: '/icons/kiss-startpage-icon-192.png?v=2.8.0', sizes: '192x192', type: 'image/png', purpose: 'any' },
      { src: '/icons/kiss-startpage-icon-512.png?v=2.8.0', sizes: '512x512', type: 'image/png', purpose: 'any' },
      { src: '/icons/kiss-startpage-icon-maskable-512.png?v=2.8.0', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
    ],
  });

  for (const [url, expectedSize] of iconCases) {
    const response = await request.get(url);
    expect(response.ok()).toBe(true);
    expect(response.headers()['content-type']).toBe('image/png');
    const png = await response.body();
    expect(png.subarray(1, 4).toString('ascii')).toBe('PNG');
    expect(png.readUInt32BE(16)).toBe(expectedSize);
    expect(png.readUInt32BE(20)).toBe(expectedSize);
  }
});
