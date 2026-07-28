import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.resolve(currentDir, '../..');
const repoDir = path.resolve(frontendDir, '..');
const backendDir = path.join(repoDir, 'backend-go');
const dataDir = await mkdtemp(path.join(tmpdir(), 'kiss-startpage-e2e-'));

function createEntries(groupIndex, count, iconOffset) {
  return Array.from({ length: count }, (_, entryIndex) => {
    const iconIndex = iconOffset + entryIndex;
    return {
      id: `button-${groupIndex + 1}-${entryIndex + 1}`,
      name: `Button ${groupIndex + 1}.${entryIndex + 1}`,
      icon: '',
      iconData:
        iconIndex < 36
          ? 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjwvc3ZnPg=='
          : '',
      links: {
        external: `https://example.com/${groupIndex + 1}/${entryIndex + 1}`,
        internal: ''
      }
    };
  });
}

function createGroups(names, counts, startOffset = 0) {
  let iconOffset = startOffset;
  return names.map((title, index) => {
    const entries = createEntries(index + startOffset, counts[index], iconOffset);
    iconOffset += counts[index];
    return {
      id: `group-${title.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`,
      title,
      groupEnd: true,
      buttonSolidColor: '',
      entries
    };
  });
}

const frequentCounts = [4, 5, 3, 6, 4, 2, 2];
const homelabCounts = [8, 9, 10, 8, 8, 5, 3];
const config = {
  title: 'KISS reliability fixture',
  dashboards: [
    {
      id: 'frequent',
      label: 'Frequent',
      enableInternalLinks: false,
      openLinksInNewTab: true,
      groups: createGroups(
        ['Work', 'Frequent', 'AI', 'Automation', 'Search', 'News', 'Tools'],
        frequentCounts
      )
    },
    {
      id: 'homelab',
      label: 'Homelab',
      enableInternalLinks: false,
      openLinksInNewTab: true,
      groups: createGroups(
        ['AI', 'Core-infrastructure', 'Smart-Home', 'Monitoring', 'Arr-stack', 'Media-servers', 'Download-upload'],
        homelabCounts,
        7
      )
    }
  ]
};

await writeFile(path.join(dataDir, 'dashboard-config.json'), JSON.stringify(config, null, 2));

const child = spawn('go', ['run', '.'], {
  cwd: backendDir,
  env: {
    ...process.env,
    DASH_BIND: '127.0.0.1',
    DASH_PORT: '18788',
    DASH_DATA_DIR: dataDir,
    DASH_PRIVATE_ICONS_DIR: path.join(dataDir, 'private-icons'),
    DASH_DEFAULT_CONFIG: path.join(repoDir, 'startpage-default-config.json'),
    DASH_APP_ROOT: path.join(frontendDir, 'dist')
  },
  stdio: 'inherit'
});

let stopping = false;
async function stop(signal = 'SIGTERM') {
  if (stopping) return;
  stopping = true;
  child.kill(signal);
  await rm(dataDir, { recursive: true, force: true });
}

process.on('SIGINT', () => stop('SIGINT'));
process.on('SIGTERM', () => stop('SIGTERM'));
process.on('exit', () => {
  child.kill('SIGTERM');
});

child.on('exit', async (code) => {
  await rm(dataDir, { recursive: true, force: true });
  process.exit(code ?? 0);
});
