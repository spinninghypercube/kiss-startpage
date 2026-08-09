export const THEME_DEFAULTS = {
  backgroundColor: '#0f172a',
  groupBackgroundColor: '#111827',
  textColor: '#f8fafc',
  buttonTextColor: '#0f172a',
  tabColor: '#1e293b',
  tabHoverColor: '#253348',
  activeTabColor: '#2563eb',
  tabTextColor: '#cbd5e1',
  activeTabTextColor: '#ffffff',
  groupBorderColor: '#1e293b',
  buttonColorMode: 'cycle-custom',
  buttonCycleHueStep: 15,
  buttonCycleSaturation: 70,
  buttonCycleLightness: 74,
  buttonSolidColor: '#93c5fd'
};

export const BUILT_IN_THEME_PRESETS = [
  {
    id: 'builtin-default-theme', name: 'Default Theme',
    theme: {
      backgroundColor: '#101728', groupBackgroundColor: '#172644', textColor: '#f8fafc',
      buttonTextColor: '#0f172a', tabColor: '#1e293b', activeTabColor: '#2563eb',
      tabTextColor: '#cbd5e1', activeTabTextColor: '#ffffff', buttonColorMode: 'cycle-custom',
      buttonCycleHueStep: 15, buttonCycleSaturation: 70, buttonCycleLightness: 74, buttonSolidColor: '#93c5fd'
    }
  },
  {
    id: 'builtin-kiss-brand', name: 'KISS Brand',
    theme: {
      backgroundColor: '#050709', groupBackgroundColor: '#12171a', textColor: '#f2f4f5',
      buttonTextColor: '#0f172a', tabColor: '#151b20', tabHoverColor: '#1b2329',
      activeTabColor: '#00cbe5', tabTextColor: '#a6afb4', activeTabTextColor: '#041014',
      groupBorderColor: '#2a3138', buttonColorMode: 'cycle-custom', buttonCycleHueStep: 15,
      buttonCycleSaturation: 70, buttonCycleLightness: 74, buttonSolidColor: '#00cbe5'
    }
  },
  {
    id: 'builtin-paper-ink', name: 'Paper & Ink',
    theme: {
      backgroundColor: '#f8fafc', groupBackgroundColor: '#e2e8f0', textColor: '#0f172a',
      buttonTextColor: '#ffffff', tabColor: '#cbd5e1', activeTabColor: '#0f172a',
      tabTextColor: '#0f172a', activeTabTextColor: '#f8fafc', buttonColorMode: 'solid-all',
      buttonCycleHueStep: 15, buttonCycleSaturation: 70, buttonCycleLightness: 74, buttonSolidColor: '#0f172a'
    }
  },
  {
    id: 'builtin-forest-terminal', name: 'Forest Terminal',
    theme: {
      backgroundColor: '#071a12', groupBackgroundColor: '#0d261d', textColor: '#d1fae5',
      buttonTextColor: '#062f23', tabColor: '#14532d', activeTabColor: '#10b981',
      tabTextColor: '#d1fae5', activeTabTextColor: '#052e22', buttonColorMode: 'cycle-custom',
      buttonCycleHueStep: 11, buttonCycleSaturation: 66, buttonCycleLightness: 64, buttonSolidColor: '#34d399'
    }
  },
  {
    id: 'builtin-sunset-control', name: 'Sunset Control',
    theme: {
      backgroundColor: '#1f1027', groupBackgroundColor: '#2c1637', textColor: '#fae8ff',
      buttonTextColor: '#240f2d', tabColor: '#4a1d5d', activeTabColor: '#f97316',
      tabTextColor: '#f5d0fe', activeTabTextColor: '#1f0a04', buttonColorMode: 'cycle-custom',
      buttonCycleHueStep: 18, buttonCycleSaturation: 85, buttonCycleLightness: 68, buttonSolidColor: '#fb923c'
    }
  },
  {
    id: 'builtin-warm-ember', name: 'Warm Ember',
    theme: {
      backgroundColor: '#160c02', groupBackgroundColor: '#2a1600', textColor: '#fde8c4',
      buttonTextColor: '#160c02', tabColor: '#3d2006', activeTabColor: '#f59e0b',
      tabTextColor: '#fde8c4', activeTabTextColor: '#160c02', buttonColorMode: 'cycle-custom',
      buttonCycleHueStep: 25, buttonCycleSaturation: 80, buttonCycleLightness: 62, buttonSolidColor: '#f59e0b'
    }
  }
];

export const COLOR_GROUPS = [
  { label: 'Page', fields: [
    { key: 'backgroundColor', label: 'Background' },
    { key: 'textColor', label: 'Text' }
  ] },
  { label: 'Tabs', fields: [
    { key: 'tabColor', label: 'Tab background' },
    { key: 'tabTextColor', label: 'Tab text' },
    { key: 'tabHoverColor', label: 'Tab hover' },
    { key: 'activeTabColor', label: 'Active tab background' },
    { key: 'activeTabTextColor', label: 'Active tab text' }
  ] },
  { label: 'Groups', fields: [
    { key: 'groupBackgroundColor', label: 'Group background' },
    { key: 'groupBorderColor', label: 'Group border' }
  ] },
  { label: 'Buttons', fields: [
    { key: 'buttonTextColor', label: 'Button text' },
    { key: 'buttonSolidColor', label: 'Button color (solid mode)' }
  ] }
];

export function normalizeHexColor(value) {
  if (typeof value !== 'string') return '';
  const trimmed = value.trim();
  return /^#[0-9a-f]{6}$/i.test(trimmed) ? trimmed.toLowerCase() : '';
}

export function normalizeHexColorLoose(value) {
  const text = String(value || '').trim();
  return normalizeHexColor(text.startsWith('#') ? text : `#${text}`);
}

export function clampInteger(value, min, max, fallback) {
  const number = Number(value);
  return Number.isFinite(number) ? Math.min(max, Math.max(min, Math.round(number))) : fallback;
}

export function normalizeTheme(theme = {}) {
  return {
    backgroundColor: normalizeHexColor(theme.backgroundColor) || THEME_DEFAULTS.backgroundColor,
    groupBackgroundColor: normalizeHexColor(theme.groupBackgroundColor) || THEME_DEFAULTS.groupBackgroundColor,
    textColor: normalizeHexColor(theme.textColor) || THEME_DEFAULTS.textColor,
    buttonTextColor: normalizeHexColor(theme.buttonTextColor) || THEME_DEFAULTS.buttonTextColor,
    tabColor: normalizeHexColor(theme.tabColor) || THEME_DEFAULTS.tabColor,
    tabHoverColor: normalizeHexColor(theme.tabHoverColor) || THEME_DEFAULTS.tabHoverColor,
    activeTabColor: normalizeHexColor(theme.activeTabColor) || THEME_DEFAULTS.activeTabColor,
    tabTextColor: normalizeHexColor(theme.tabTextColor) || THEME_DEFAULTS.tabTextColor,
    activeTabTextColor: normalizeHexColor(theme.activeTabTextColor) || THEME_DEFAULTS.activeTabTextColor,
    groupBorderColor: normalizeHexColor(theme.groupBorderColor) || THEME_DEFAULTS.groupBorderColor,
    buttonColorMode: ['cycle-custom', 'solid-all', 'solid-per-group'].includes(theme.buttonColorMode)
      ? theme.buttonColorMode : THEME_DEFAULTS.buttonColorMode,
    buttonCycleHueStep: clampInteger(theme.buttonCycleHueStep, 1, 180, THEME_DEFAULTS.buttonCycleHueStep),
    buttonCycleSaturation: clampInteger(theme.buttonCycleSaturation, 0, 100, THEME_DEFAULTS.buttonCycleSaturation),
    buttonCycleLightness: clampInteger(theme.buttonCycleLightness, 0, 100, THEME_DEFAULTS.buttonCycleLightness),
    buttonSolidColor: normalizeHexColor(theme.buttonSolidColor) || THEME_DEFAULTS.buttonSolidColor
  };
}

export function applyThemeCssVars(theme, documentObject = globalThis.document) {
  const t = normalizeTheme(theme);
  const root = documentObject?.documentElement;
  if (!root) return t;
  const set = (name, value) => root.style.setProperty(name, value);
  const sameSurface = t.backgroundColor === t.groupBackgroundColor;
  const navBackground = highContrastColor(t.backgroundColor);
  const navText = highContrastColor(navBackground);
  const switchKnob = highContrastColor(t.groupBackgroundColor);

  applyBrowserThemeColor(t.backgroundColor, documentObject);
  root.style.colorScheme = isDarkColor(t.backgroundColor) ? 'dark' : 'light';
  root.classList.toggle('startpage-group-shell-flat', sameSurface);
  root.classList.toggle('admin-group-shell-flat', sameSurface);

  const variables = {
    '--startpage-page-bg': t.backgroundColor,
    '--startpage-group-bg': t.groupBackgroundColor,
    '--startpage-group-border': t.groupBorderColor,
    '--startpage-group-radius': '0.85rem',
    '--startpage-group-padding': sameSurface ? '12px 0' : '24px',
    '--startpage-group-inner-padding': sameSurface ? '0' : '0.4rem',
    '--startpage-group-title-inset': sameSurface ? '0' : '0.75rem',
    '--startpage-text-color': t.textColor,
    '--startpage-button-text-color': t.buttonTextColor,
    '--startpage-tab-bg': t.tabColor,
    '--startpage-tab-hover-bg': t.tabHoverColor,
    '--startpage-tab-text': t.tabTextColor,
    '--startpage-tab-hover-text': t.tabTextColor,
    '--startpage-tab-active-bg': t.activeTabColor,
    '--startpage-tab-active-text': t.activeTabTextColor,
    '--startpage-nav-tab-bg': navBackground,
    '--startpage-nav-tab-text': navText,
    '--startpage-nav-tab-hover-bg': rgba(navBackground, 0.86),
    '--startpage-nav-tab-hover-text': navText,
    '--startpage-switch-track-bg': t.groupBackgroundColor,
    '--startpage-switch-track-border': rgba(switchKnob, 0.22),
    '--startpage-switch-knob-bg': switchKnob,
    ...deriveEditModeColors(t)
  };
  for (const [name, value] of Object.entries(variables)) set(name, value);
  return t;
}

export function applyBrowserThemeColor(value, documentObject = globalThis.document) {
  const color = normalizeHexColor(value) || THEME_DEFAULTS.backgroundColor;
  documentObject?.querySelector?.('meta[name="theme-color"]')?.setAttribute?.('content', color);
  return color;
}

export function deriveEditModeColors(theme) {
  const t = normalizeTheme(theme);
  const surfaceText = highContrastColor(t.groupBackgroundColor);
  const pageText = highContrastColor(t.backgroundColor);
  const panelSurface = blend(t.backgroundColor, '#0f172a', 0.72);
  const panelText = highContrastColor(panelSurface);
  const primaryHover = blend(t.activeTabColor, highContrastColor(t.activeTabColor), 0.12);
  return {
    '--admin-bg': shade(t.backgroundColor, -8),
    '--admin-panel-bg': shade(t.groupBackgroundColor, -5),
    '--admin-text': t.textColor,
    '--admin-label-text': rgba(t.textColor, 0.75),
    '--admin-accent': t.activeTabColor,
    '--admin-accent-text': t.activeTabTextColor,
    '--admin-tab-bg': t.tabColor,
    '--admin-tab-text': t.tabTextColor,
    '--admin-tab-active-bg': t.activeTabColor,
    '--admin-tab-active-text': t.activeTabTextColor,
    '--admin-group-bg': t.groupBackgroundColor,
    '--admin-group-border': t.groupBorderColor,
    '--admin-surface-text-color': surfaceText,
    '--admin-surface-text-muted': rgba(surfaceText, 0.88),
    '--admin-page-text-color': pageText,
    '--admin-page-text-muted': rgba(pageText, 0.88),
    '--admin-panel-text-color': panelText,
    '--admin-panel-text-muted': rgba(panelText, 0.88),
    '--admin-input-bg': t.backgroundColor,
    '--admin-input-text': pageText,
    '--admin-input-placeholder': rgba(pageText, 0.72),
    '--admin-input-border': rgba(pageText, 0.28),
    '--admin-input-focus-border': t.activeTabColor,
    '--admin-input-focus-ring': rgba(t.activeTabColor, 0.22),
    '--admin-modal-bg': t.groupBackgroundColor,
    '--admin-modal-chrome-bg': blend(t.groupBackgroundColor, '#0f172a', 0.08),
    '--admin-modal-border': rgba(surfaceText, 0.22),
    '--admin-button-bg': blend(t.groupBackgroundColor, surfaceText, 0.12),
    '--admin-button-hover-bg': blend(t.groupBackgroundColor, surfaceText, 0.2),
    '--admin-button-text': surfaceText,
    '--admin-button-border': rgba(surfaceText, 0.28),
    '--admin-button-primary-bg': t.activeTabColor,
    '--admin-button-primary-hover-bg': primaryHover,
    '--admin-button-primary-text': t.activeTabTextColor,
    '--admin-button-danger-bg': '#dc2626',
    '--admin-button-danger-hover-bg': '#e14343',
    '--admin-button-danger-text': '#ffffff',
    '--admin-button-warning-bg': '#d97706',
    '--admin-button-warning-text': '#ffffff',
    '--admin-entry-add-accent': surfaceText,
    '--admin-entry-add-accent-soft': rgba(surfaceText, 0.78),
    '--admin-entry-add-accent-hover-bg': rgba(surfaceText, 0.1),
    '--admin-group-add-accent': pageText,
    '--admin-group-add-accent-soft': rgba(pageText, 0.78),
    '--admin-group-add-accent-hover-bg': rgba(pageText, 0.1),
    '--admin-nav-bg': shade(t.tabColor, -10),
    '--admin-nav-text': t.tabTextColor,
    '--admin-danger': '#ef4444',
    '--admin-success': '#22c55e'
  };
}

function rgb(hex) {
  const value = normalizeHexColor(hex).slice(1);
  if (!value) return null;
  const number = parseInt(value, 16);
  return { r: (number >> 16) & 255, g: (number >> 8) & 255, b: number & 255 };
}

function toHex({ r, g, b }) {
  return `#${[r, g, b].map((part) => Math.round(part).toString(16).padStart(2, '0')).join('')}`;
}

function isDarkColor(hex) {
  const color = rgb(hex);
  return !color || (0.299 * color.r + 0.587 * color.g + 0.114 * color.b) < 150;
}

function highContrastColor(hex) {
  return isDarkColor(hex) ? '#f8fafc' : '#0f172a';
}

function rgba(hex, alpha) {
  const color = rgb(hex) || { r: 248, g: 250, b: 252 };
  return `rgba(${color.r}, ${color.g}, ${color.b}, ${Math.max(0, Math.min(1, Number(alpha) || 0))})`;
}

function blend(background, overlay, alpha) {
  const bg = rgb(background);
  const fg = rgb(overlay);
  if (!bg || !fg) return normalizeHexColor(background) || normalizeHexColor(overlay);
  const weight = Math.max(0, Math.min(1, Number(alpha) || 0));
  return toHex({
    r: bg.r * (1 - weight) + fg.r * weight,
    g: bg.g * (1 - weight) + fg.g * weight,
    b: bg.b * (1 - weight) + fg.b * weight
  });
}

function shade(hex, percent) {
  const color = rgb(hex);
  if (!color) return hex;
  const shift = 2.55 * percent;
  return toHex({
    r: Math.max(0, Math.min(255, color.r + shift)),
    g: Math.max(0, Math.min(255, color.g + shift)),
    b: Math.max(0, Math.min(255, color.b + shift))
  });
}
