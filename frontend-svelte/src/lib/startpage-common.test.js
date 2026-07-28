import { afterEach, describe, expect, it, vi } from 'vitest';
import { StartpageCommon } from './startpage-common.js';
import { deriveEditModeColors } from './theme.js';

afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe('normalizeConfig', () => {
  it('preserves every group and repairs duplicate group and button IDs', () => {
    const groups = Array.from({ length: 7 }, (_, groupIndex) => ({
      id: groupIndex < 2 ? 'duplicate-group' : `group-${groupIndex}`,
      title: groupIndex === 4 ? 'Arr-stack' : `Group ${groupIndex + 1}`,
      entries: Array.from({ length: groupIndex === 4 ? 8 : 3 }, (_, entryIndex) => ({
        id: entryIndex === 0 ? 'duplicate-button' : `button-${groupIndex}-${entryIndex}`,
        name: `Button ${groupIndex + 1}.${entryIndex + 1}`,
        links: { external: `https://example.com/${groupIndex}/${entryIndex}` }
      }))
    }));

    const normalized = StartpageCommon.normalizeConfig({
      title: 'Reliability fixture',
      dashboards: [{ id: 'homelab', label: 'Homelab', groups }]
    });

    const normalizedGroups = normalized.dashboards[0].groups;
    expect(normalizedGroups).toHaveLength(7);
    expect(normalizedGroups.find((group) => group.title === 'Arr-stack').entries).toHaveLength(8);

    const groupIDs = normalizedGroups.map((group) => group.id);
    expect(new Set(groupIDs).size).toBe(groupIDs.length);

    const buttonIDs = normalizedGroups.flatMap((group) => group.entries.map((entry) => entry.id));
    expect(new Set(buttonIDs).size).toBe(buttonIDs.length);
  });
});

describe('getConfig', () => {
  it('retries the API once and succeeds without loading the demo config', async () => {
    vi.useFakeTimers();
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response('temporary failure', { status: 503 }))
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            config: {
              title: 'Recovered',
              dashboards: [{ id: 'home', label: 'Home', groups: [] }]
            }
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      );

    const resultPromise = StartpageCommon.getConfig();
    await vi.advanceTimersByTimeAsync(300);
    const result = await resultPromise;

    expect(result.title).toBe('Recovered');
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock.mock.calls.every(([path]) => path === '/api/config')).toBe(true);
  });

  it('rejects after two API failures instead of showing demo data', async () => {
    vi.useFakeTimers();
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockImplementation(async () => new Response('unavailable', { status: 503 }));

    const resultPromise = StartpageCommon.getConfig();
    const capturedError = resultPromise.catch((error) => error);
    await vi.advanceTimersByTimeAsync(300);
    expect((await capturedError).message).toContain('unavailable');

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock.mock.calls.every(([path]) => path === '/api/config')).toBe(true);
  });
});

describe('deriveEditModeColors', () => {
  it('derives neutral and primary controls from both light and dark themes', () => {
    const paper = deriveEditModeColors({
      backgroundColor: '#f8fafc',
      groupBackgroundColor: '#e2e8f0',
      textColor: '#0f172a',
      buttonTextColor: '#ffffff',
      tabColor: '#cbd5e1',
      activeTabColor: '#0f172a',
      tabTextColor: '#0f172a',
      activeTabTextColor: '#f8fafc',
      groupBorderColor: '#cbd5e1'
    });
    const sunset = deriveEditModeColors({
      backgroundColor: '#1f1027',
      groupBackgroundColor: '#2c1637',
      textColor: '#fae8ff',
      buttonTextColor: '#240f2d',
      tabColor: '#4a1d5d',
      activeTabColor: '#f97316',
      tabTextColor: '#f5d0fe',
      activeTabTextColor: '#1f0a04',
      groupBorderColor: '#4a1d5d'
    });

    expect(paper['--admin-button-bg']).not.toBe(sunset['--admin-button-bg']);
    expect(paper['--admin-button-primary-bg']).toBe('#0f172a');
    expect(paper['--admin-button-primary-text']).toBe('#f8fafc');
    expect(sunset['--admin-button-primary-bg']).toBe('#f97316');
    expect(sunset['--admin-button-primary-text']).toBe('#1f0a04');
  });
});
