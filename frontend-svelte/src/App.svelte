<script>
  import { flip } from 'svelte/animate';
  import { onDestroy, onMount, tick } from 'svelte';
  import Sortable from 'sortablejs';
  import './styles/startpage.css';
  import './styles/edit.css';
  import { StartpageCommon, iconSource } from './lib/startpage-common.js';
  import {
    BUILT_IN_THEME_PRESETS,
    THEME_DEFAULTS,
    applyThemeCssVars,
    clampInteger,
    normalizeHexColor,
    normalizeHexColorLoose,
    normalizeTheme
  } from './lib/theme.js';
  import LoginView from './components/LoginView.svelte';
  import AccountPane from './components/AccountPane.svelte';
  import ButtonModal from './components/ButtonModal.svelte';
  import ThemeEditor from './components/ThemeEditor.svelte';

  // ─── Constants ────────────────────────────────────────────────────────────────

  const DEFAULT_BUTTON_COLOR_OPTIONS = StartpageCommon.getDefaultButtonColorOptions();
  const DND_FLIP_DURATION_MS = 160;

  /** Returns #ffffff or #000000 for a high-contrast 2px border on a color swatch. */
  function contrastBorder(hex) {
    hex = (hex || '#000000').replace('#', '');
    if (hex.length === 3) hex = hex.split('').map(c => c + c).join('');
    const r = parseInt(hex.slice(0, 2), 16);
    const g = parseInt(hex.slice(2, 4), 16);
    const b = parseInt(hex.slice(4, 6), 16);
    return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.5 ? '#000000' : '#ffffff';
  }
  // ─── Shared state ─────────────────────────────────────────────────────────────

  let editMode = false;
  let appVersion = '';
  let config = { title: 'KISS Startpage', theme: {}, dashboards: [], themePresets: [] };
  let activeTabId = StartpageCommon.getActiveStartpageId();
  let activeTab = null;
  let loading = true;
  let loadError = '';

  // ─── View-mode state ──────────────────────────────────────────────────────────

  let currentLinkMode = StartpageCommon.getLinkMode();
  let viewGroups = [];
  let tabsScrollEl;
  let tabsListEl;
  let tabsRowEl;
  let tabsOverflowing = false;
  let isMobileViewport = false;
  let resizeObserver;

  // ─── Edit-mode auth state ─────────────────────────────────────────────────────

  let authenticated = false;
  let authUser = '';
  let authMustChangePassword = false;
  let authSetupRequired = false;

  // ─── Edit-mode UI state ───────────────────────────────────────────────────────

  let accountPaneOpen = false;
  let showThemeEditor = false;
  let messageText = '';
  let messageTone = 'is-success';
  let messageVisible = false;
  let messageTimer = null;

  // ─── Edit-mode draft state ────────────────────────────────────────────────────

  let pageTitleDraft = '';
  let enableInternalLinksDraft = false;
  let openLinksInNewTabDraft = true;
  let showLinkToggleDraft = true;
  let themeDraft = normalizeTheme({});
  let themePresetName = '';

  // ─── Edit-mode modal state ────────────────────────────────────────────────────

  let actionModal = {
    open: false, mode: '', tabId: '', groupId: '', buttonId: '',
    title: 'Add Group', text: 'Create a new group.', titleLabel: 'Group Title',
    titlePlaceholder: 'New Group', titleValue: '', titleFieldVisible: true,
    confirmLabel: 'Add Group', confirmTone: 'is-link'
  };
  let buttonModalOpen = false;
  let buttonModalIsNew = false;
  let buttonModalGroupId = '';
  let buttonModalButtonId = '';
  let buttonModalInitialData = { name: '', icon: '', externalUrl: '', internalUrl: '', iconData: '', iconMeta: null };

  // ─── Edit-mode computed state ─────────────────────────────────────────────────

  let builtInThemePresets = [];
  let savedThemePresets = [];
  let editorGroups = [];
  let themeButtonMode = StartpageCommon.normalizeButtonColorMode(themeDraft.buttonColorMode);

  // ─── Helpers ──────────────────────────────────────────────────────────────────

  let configRevision = 0;
  let savedConfigRevision = 0;
  let configSaveTimer = null;
  let configSaveInFlight = null;

  function wait(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  async function saveConfigSnapshot(snapshot) {
    let lastError;
    for (let attempt = 0; attempt < 2; attempt += 1) {
      try {
        return await StartpageCommon.saveConfig(snapshot);
      } catch (error) {
        lastError = error;
        if (attempt === 0) await wait(300);
      }
    }
    throw lastError || new Error('Failed to save configuration.');
  }

  async function saveNextConfigRevision() {
    const revision = configRevision;
    const snapshot = StartpageCommon.normalizeConfig(config);
    const saved = await saveConfigSnapshot(snapshot);
    savedConfigRevision = Math.max(savedConfigRevision, revision);
    if (revision === configRevision) {
      config = StartpageCommon.normalizeConfig(saved);
      ensureActiveTab();
      syncDraftsFromActiveTab();
      applyCurrentAdminThemePreview();
    }
  }

  async function flushConfigSaves() {
    if (configSaveTimer) {
      clearTimeout(configSaveTimer);
      configSaveTimer = null;
    }
    while (savedConfigRevision < configRevision) {
      if (!configSaveInFlight) {
        configSaveInFlight = saveNextConfigRevision().finally(() => {
          configSaveInFlight = null;
        });
      }
      await configSaveInFlight;
    }
  }

  function debouncedSave() {
    if (configSaveTimer) clearTimeout(configSaveTimer);
    configSaveTimer = setTimeout(() => {
      configSaveTimer = null;
      flushConfigSaves().catch((error) => {
        console.error(error);
        showMessage('Failed to save changes. Your edits are still open.', 'is-danger');
      });
    }, 500);
  }

  function clone(value) { return StartpageCommon.clone(value); }

  function normalizeLinkMode(mode) { return mode === 'internal' ? 'internal' : 'external'; }

  function clearMessageTimer() {
    if (messageTimer) { clearTimeout(messageTimer); messageTimer = null; }
  }
  function hideMessage() {
    clearMessageTimer();
    messageVisible = false; messageText = ''; messageTone = 'is-success';
  }
  function showMessage(text, tone = 'is-success') {
    clearMessageTimer();
    messageText = text || ''; messageTone = tone || 'is-success';
    messageVisible = Boolean(messageText);
    if (messageVisible) messageTimer = setTimeout(() => hideMessage(), 2000);
  }

  function touchConfig() {
    config = clone(config);
    configRevision += 1;
  }

  // ─── Tab lookup ─────────────────────────────────────────────────────────

  function getTabIndex(tabId) {
    return (config.dashboards || []).findIndex((d) => d.id === tabId);
  }

  function ensureActiveTab() {
    if (!Array.isArray(config.dashboards) || !config.dashboards.length) {
      config.dashboards = [{
        id: StartpageCommon.createId('tab'), label: 'Startpage 1',
        enableInternalLinks: false,
        openLinksInNewTab: true,
        showLinkModeToggle: true, themePresets: [], groups: []
      }];
    }
    if (!activeTabId || getTabIndex(activeTabId) < 0) {
      activeTabId = config.dashboards[0].id;
    }
  }

  function getActiveTab() {
    ensureActiveTab();
    return config.dashboards[getTabIndex(activeTabId)] || null;
  }

  function getValidStartpageId(requestedId) {
    const dashboards = Array.isArray(config?.dashboards) ? config.dashboards : [];
    if (!dashboards.length) return '';
    if (requestedId && dashboards.some((d) => d.id === requestedId)) return requestedId;
    const saved = StartpageCommon.getActiveStartpageId();
    if (saved && dashboards.some((d) => d.id === saved)) return saved;
    return dashboards[0].id;
  }

  function setActiveTab(tabId) {
    hideMessage();
    activeTabId = getValidStartpageId(tabId);
    ensureActiveTab();
    StartpageCommon.setActiveStartpageId(activeTabId);
    accountPaneOpen = false;
    if (editMode) {
      syncDraftsFromActiveTab();
      applyCurrentAdminThemePreview();
    }
    // Theme is now global — no per-tab theme application needed
  }

  // ─── Theme helpers ────────────────────────────────────────────────────────────

  function getResolvedBuiltInThemePresets() {
    return BUILT_IN_THEME_PRESETS.map((preset) => ({ ...preset, theme: normalizeTheme(preset.theme) }));
  }

  function buildDefaultThemeValues() {
    const builtInDefault = getResolvedBuiltInThemePresets().find((p) => p.id === 'builtin-default-theme');
    return builtInDefault ? { ...builtInDefault.theme } : normalizeTheme({});
  }

  function applyStartpageTheme() {
    const themeSource = editMode ? themeDraft : (config.theme || {});
    applyThemeCssVars(themeSource);
  }

  function applyCurrentAdminThemePreview() {
    applyThemeCssVars(themeDraft);
  }

  function applyPageTitle() {
    document.title = (config?.title || '').toString().trim() || 'KISS Startpage';
  }

  // ─── View-mode helpers ────────────────────────────────────────────────────────

  function setLinkMode(mode) {
    currentLinkMode = normalizeLinkMode(mode);
    StartpageCommon.setLinkMode(currentLinkMode);
  }

  function buttonResolvedHref(entry, dashboard) {
    const effectiveMode = dashboard?.enableInternalLinks && currentLinkMode === 'internal' ? 'internal' : 'external';
    const raw = entry?.links && typeof entry.links[effectiveMode] === 'string' ? entry.links[effectiveMode].trim() : '';
    return effectiveMode === 'external' ? StartpageCommon.normalizeExternalLinkHref(raw) : raw;
  }

  function buttonDecorations(dashboard) {
    const groups = Array.isArray(dashboard?.groups) ? dashboard.groups : [];
    const result = [];
    let colorIndex = 0;
    // Merge global theme into tab for button color calculations
    const themeSource = { ...dashboard, ...themeDraft };
    for (const group of groups) {
      const entries = [];
      for (const entry of Array.isArray(group?.entries) ? group.entries : []) {
        const color = StartpageCommon.getButtonColorPair(themeSource, group, colorIndex);
        colorIndex += 1;
        entries.push({ entry, color, href: buttonResolvedHref(entry, dashboard), iconSrc: iconSource(entry) });
      }
      result.push({ group, entries });
    }
    return result;
  }

  function updateTabsOverflowState() {
    if (!tabsScrollEl || !tabsListEl || (config?.dashboards || []).length <= 1) {
      tabsOverflowing = false; return;
    }
    // Compare tabs width against the full row width (not the reduced scroll container).
    // Button moves below only when tabs genuinely exceed the entire available row width.
    // This means tabs can scroll within their allocated space without triggering overflow.
    const rowWidth = tabsRowEl ? tabsRowEl.clientWidth : tabsScrollEl.clientWidth;
    tabsOverflowing = tabsListEl.scrollWidth > rowWidth;
  }
  async function refreshTabsOverflowState() { await tick(); updateTabsOverflowState(); }
  function updateViewportState() { isMobileViewport = window.innerWidth <= 768; }

  // ─── Config loading ───────────────────────────────────────────────────────────

  async function loadConfig() {
    loading = true;
    loadError = '';
    try {
      config = StartpageCommon.normalizeConfig(await StartpageCommon.getConfig());
      configRevision = 0;
      savedConfigRevision = 0;
      activeTabId = getValidStartpageId('');
      ensureActiveTab();
      StartpageCommon.setActiveStartpageId(activeTabId);
      // Initialize themeDraft from config.theme (migration: normalizeConfig handles it)
      themeDraft = normalizeTheme(config.theme || config.dashboards?.[0] || {});
      applyPageTitle();
      applyStartpageTheme();
    } catch (error) {
      console.error(error);
      loadError = 'The startpage configuration could not be loaded.';
    } finally {
      loading = false;
    }
  }

  async function persistConfig(message = '') {
    await flushConfigSaves();
  }

  // ─── Mode switching ───────────────────────────────────────────────────────────

  async function enterEditMode() {
    if (window.history?.pushState) window.history.pushState(null, '', '/edit');
    editMode = true;
    try {
      const status = await StartpageCommon.getAuthStatus();
      if (!status?.authenticated) {
        authSetupRequired = Boolean(status?.setupRequired);
        authenticated = false;
        applyCurrentAdminThemePreview();
        return;
      }
      authUser = status?.username || 'admin';
      authMustChangePassword = Boolean(status?.mustChangePassword);
      authenticated = true;
      syncDraftsFromActiveTab();
      applyCurrentAdminThemePreview();
      if (authMustChangePassword) {
        accountPaneOpen = true;
      }
    } catch (error) {
      console.error(error);
      showMessage('Admin API unavailable. Check backend service.', 'is-danger');
    }
  }

  async function exitEditMode() {
    try {
      await flushConfigSaves();
    } catch (error) {
      console.error(error);
      showMessage('Failed to save changes. Edit mode remains open.', 'is-danger');
      return false;
    }
    if (window.history?.pushState) window.history.pushState(null, '', '/');
    editMode = false;
    authenticated = false;
    accountPaneOpen = false;
    showThemeEditor = false;
    hideMessage();
    applyStartpageTheme();
    return true;
  }

  async function toggleEditMode() {
    if (editMode) { await exitEditMode(); } else { await enterEditMode(); }
  }

  async function logoutSubmit() {
    hideMessage();
    try {
      await flushConfigSaves();
    } catch (error) {
      console.error(error);
      showMessage('Failed to save changes. Log out cancelled.', 'is-danger');
      return;
    }
    try { await StartpageCommon.logout(); } catch (error) { console.warn(error); }
    authUser = '';
    await exitEditMode();
  }

  // ─── Edit-mode draft sync ─────────────────────────────────────────────────────

  function getSavedThemePresets() {
    const items = Array.isArray(config?.themePresets) ? config.themePresets : [];
    return items.map((preset, i) => ({
      id: preset?.id ? String(preset.id) : `theme-${i + 1}`,
      name: (preset?.name || `Theme ${i + 1}`).toString(),
      theme: normalizeTheme(preset?.theme)
    }));
  }

  function getNextCustomThemePresetName() {
    const names = new Set(getSavedThemePresets().map((p) => p.name.trim().toLowerCase()));
    for (let i = 1; i < 10000; i += 1) {
      const candidate = `Custom theme ${i}`;
      if (!names.has(candidate.toLowerCase())) return candidate;
    }
    return `Custom theme ${Date.now()}`;
  }

  function syncDraftsFromActiveTab() {
    const dashboard = getActiveTab();
    if (!dashboard) return;
    pageTitleDraft = dashboard.label || '';
    enableInternalLinksDraft = Boolean(dashboard.enableInternalLinks);
    openLinksInNewTabDraft = dashboard.openLinksInNewTab !== false;
    showLinkToggleDraft = dashboard.showLinkModeToggle !== false;
    // Theme is global — read from config.theme, not per-tab
    themeDraft = normalizeTheme(config.theme || {});
    if (!themePresetName.trim()) themePresetName = getNextCustomThemePresetName();
  }

  // ─── Edit-mode theme functions ────────────────────────────────────────────────

  function setThemeField(key, value, { commit = false } = {}) {
    const next = { ...themeDraft };
    if (key === 'buttonColorMode') {
      next[key] = StartpageCommon.normalizeButtonColorMode(value);
    } else if (key === 'buttonCycleHueStep') {
      next[key] = clampInteger(value, 1, 180, DEFAULT_BUTTON_COLOR_OPTIONS.buttonCycleHueStep);
    } else if (key === 'buttonCycleSaturation') {
      next[key] = clampInteger(value, 0, 100, DEFAULT_BUTTON_COLOR_OPTIONS.buttonCycleSaturation);
    } else if (key === 'buttonCycleLightness') {
      next[key] = clampInteger(value, 0, 100, DEFAULT_BUTTON_COLOR_OPTIONS.buttonCycleLightness);
    } else if (key === 'buttonSolidColor') {
      next[key] = normalizeHexColorLoose(value) || next[key] || DEFAULT_BUTTON_COLOR_OPTIONS.buttonSolidColor;
    } else {
      next[key] = normalizeHexColorLoose(value) || (commit ? (THEME_DEFAULTS[key] || '') : next[key]);
      if (!next[key]) next[key] = THEME_DEFAULTS[key] || '';
    }
    themeDraft = next;
    config = { ...config, theme: { ...themeDraft } };
    configRevision += 1;
    applyCurrentAdminThemePreview();
    if (commit) debouncedSave();
  }

  // ThemeEditor event handlers
  function handleThemeUpdate(e) {
    setThemeField(e.detail.key, e.detail.value, { commit: Boolean(e.detail.commit) });
  }

  function handleSavePreset(e) {
    themePresetName = e.detail.name || themePresetName;
    saveCurrentThemePreset().catch((err) => { console.error(err); showMessage('Failed to save theme preset.', 'is-danger'); });
  }

  function handleLoadPreset(e) {
    const preset = e.detail.preset;
    if (!preset) return;
    themePresetName = getNextCustomThemePresetName();
    themeDraft = normalizeTheme(preset.theme);
    config = { ...config, theme: { ...themeDraft } };
    configRevision += 1;
    applyCurrentAdminThemePreview();
    debouncedSave();
  }

  function handleDeletePreset(e) {
    const preset = e.detail.preset;
    if (!preset) return;
    deleteThemePreset(preset)
      .then(() => {
        // Reset to default theme after deleting a custom preset
        const defaultPreset = getResolvedBuiltInThemePresets().find((p) => p.id === 'builtin-default-theme');
        if (defaultPreset) {
          themeDraft = normalizeTheme(defaultPreset.theme);
          config.theme = { ...themeDraft };
          touchConfig();
          applyCurrentAdminThemePreview();
          debouncedSave();
        }
      })
      .catch((err) => { console.error(err); showMessage('Failed to delete theme preset.', 'is-danger'); });
  }

  // ─── Edit-mode theme preset functions ────────────────────────────────────────

  async function saveCurrentThemePreset() {
    hideMessage();
    const name = (themePresetName || '').trim();
    if (!name) { showMessage('Preset name is required.', 'is-danger'); return; }
    const saved = getSavedThemePresets();
    const existingIndex = saved.findIndex((p) => p.name.trim().toLowerCase() === name.toLowerCase());
    const presetId = existingIndex >= 0 ? saved[existingIndex].id : StartpageCommon.createId('theme');
    const nextPreset = { id: presetId, name, theme: normalizeTheme(themeDraft) };
    const nextSaved = saved.slice();
    if (existingIndex >= 0) { nextSaved.splice(existingIndex, 1, nextPreset); } else { nextSaved.push(nextPreset); }
    config.themePresets = nextSaved;
    touchConfig();
    await persistConfig(existingIndex >= 0 ? 'Theme preset updated.' : 'Theme preset saved.');
  }

  async function deleteThemePreset(preset) {
    hideMessage();
    if (!preset?.id || !getSavedThemePresets().some((item) => item.id === preset.id)) {
      showMessage('Select a saved theme preset to delete.', 'is-danger');
      return;
    }
    const saved = getSavedThemePresets();
    const nextSaved = saved.filter((p) => p.id !== preset.id);
    if (nextSaved.length === saved.length) { showMessage('Theme preset not found.', 'is-danger'); return; }
    config.themePresets = nextSaved;
    touchConfig();
    await persistConfig(`Theme preset "${preset.name}" deleted.`);
  }

  // ─── Edit-mode save functions ─────────────────────────────────────────────────

  async function saveTabTitle() {
    hideMessage();
    const dashboard = getActiveTab();
    if (!dashboard) { showMessage('Tab not found.', 'is-danger'); return; }
    const tabTitle = pageTitleDraft.trim();
    if (!tabTitle) { showMessage('Tab title cannot be empty.', 'is-danger'); pageTitleDraft = dashboard.label; return; }
    if (dashboard.label === tabTitle) return;
    dashboard.label = tabTitle;
    touchConfig();
    await persistConfig('Tab title updated.');
  }

  async function saveActiveTabSettings(options = {}) {
    hideMessage();
    const silentSuccess = Boolean(options.silentSuccess);
    const dashboard = getActiveTab();
    if (!dashboard) { showMessage('Tab not found.', 'is-danger'); return; }
    dashboard.enableInternalLinks = Boolean(enableInternalLinksDraft);
    dashboard.openLinksInNewTab = Boolean(openLinksInNewTabDraft);
    dashboard.showLinkModeToggle = Boolean(showLinkToggleDraft);
    // Theme is now global — write to config.theme, not per-tab
    config.theme = { ...themeDraft };
    touchConfig();
    await persistConfig(silentSuccess ? '' : 'Tab settings updated.');
  }

  async function updateGroupTitle(group, nextTitle) {
    const trimmed = (nextTitle || '').trim();
    if (!trimmed) { showMessage('Group title cannot be empty.', 'is-danger'); syncDraftsFromActiveTab(); return; }
    if (group.title === trimmed) return;
    group.title = trimmed;
    touchConfig();
    await persistConfig('Group updated.');
  }

  async function pasteColorIntoField(key) {
    if (!navigator.clipboard?.readText) { showMessage('Clipboard paste is not available in this browser.', 'is-danger'); return; }
    try {
      const text = await navigator.clipboard.readText();
      const normalized = normalizeHexColorLoose(text);
      if (!normalized) throw new Error('Clipboard does not contain a valid 6-character hex color.');
      setThemeField(key, normalized, { commit: true });
    } catch (error) {
      console.error(error);
      showMessage(error.message || 'Failed to paste color.', 'is-danger');
    }
  }

  function getGroupButtonSolidColor(group) {
    return normalizeHexColor(group?.buttonSolidColor) || normalizeHexColor(themeDraft.buttonSolidColor) || DEFAULT_BUTTON_COLOR_OPTIONS.buttonSolidColor;
  }

  function setGroupButtonSolidColor(group, value, commit = false) {
    group.buttonSolidColor = normalizeHexColorLoose(value) || getGroupButtonSolidColor(group);
    touchConfig();
    if (commit) persistConfig('Tab settings updated.').catch((err) => { console.error(err); showMessage('Failed to update tab settings.', 'is-danger'); });
  }

  // ─── Edit-mode action modals ──────────────────────────────────────────────────

  function openActionModal(options) {
    actionModal = { ...actionModal, ...options, open: true };
  }

  function closeActionModal() {
    actionModal = { ...actionModal, open: false, mode: '', tabId: '', groupId: '', buttonId: '' };
  }

  function openAddTabModal() {
    openActionModal({
      mode: 'add-tab', title: 'Add Tab', text: 'Create a new tab.',
      titleLabel: 'Tab Title', titlePlaceholder: `Tab ${((config.dashboards || []).length || 0) + 1}`,
      titleValue: '', titleFieldVisible: true, confirmLabel: 'Add Tab', confirmTone: 'is-link'
    });
  }

  function openAddGroupModal() {
    openActionModal({
      mode: 'add-group', tabId: activeTabId, title: 'Add Group',
      text: 'Create a new group.', titleLabel: 'Group Title', titlePlaceholder: 'New Group',
      titleValue: '', titleFieldVisible: true, confirmLabel: 'Add Group', confirmTone: 'is-link'
    });
  }

  function openDeleteGroupModal(group) {
    const dashboard = getActiveTab();
    if (!dashboard || !group) return;
    openActionModal({
      mode: 'delete-group', tabId: dashboard.id, groupId: group.id,
      title: 'Delete Group', text: `Delete group "${group.title}" and all buttons?`,
      titleValue: '', titleFieldVisible: false, confirmLabel: 'Delete Group', confirmTone: 'is-danger'
    });
  }

  function openDeleteTabModal() {
    const dashboard = getActiveTab();
    if (!dashboard) return;
    openActionModal({
      mode: 'delete-tab', tabId: dashboard.id, title: 'Delete Tab',
      text: `Delete tab "${dashboard.label}" and all its groups/buttons?`,
      titleValue: '', titleFieldVisible: false, confirmLabel: 'Delete Tab', confirmTone: 'is-danger'
    });
  }

  function openDeleteButtonModal(groupId, buttonId, buttonName) {
    openActionModal({
      mode: 'delete-button', tabId: activeTabId, groupId, buttonId,
      title: 'Delete Button', text: `Delete button "${buttonName}"?`,
      titleValue: '', titleFieldVisible: false, confirmLabel: 'Delete Button', confirmTone: 'is-danger'
    });
  }

  async function confirmActionModal() {
    hideMessage();
    const mode = actionModal.mode;

    if (mode === 'add-tab') {
      const label = (actionModal.titleValue || '').trim() || `Tab ${(config.dashboards || []).length + 1}`;
      const dashboard = { id: StartpageCommon.makeSafeTabId(label), label, enableInternalLinks: false, openLinksInNewTab: true, showLinkModeToggle: true, themePresets: [], groups: [] };
      const existingIds = new Set((config.dashboards || []).map((d) => d.id));
      let id = dashboard.id; let n = 2;
      while (!id || existingIds.has(id)) { id = `${dashboard.id || 'tab'}-${n}`; n += 1; }
      dashboard.id = id;
      config.dashboards.push(dashboard);
      activeTabId = dashboard.id;
      touchConfig();
      closeActionModal();
      await persistConfig('');
      return;
    }

    if (mode === 'add-group') {
      const title = (actionModal.titleValue || '').trim();
      if (!title) { showMessage('Group title cannot be empty.', 'is-danger'); return; }
      const dashboard = getActiveTab();
      if (!dashboard) { closeActionModal(); showMessage('Tab not found.', 'is-danger'); return; }
      dashboard.groups.push({ id: StartpageCommon.createId('group'), title, groupEnd: true, entries: [] });
      touchConfig();
      closeActionModal();
      await persistConfig('Group added.');
      return;
    }

    if (mode === 'delete-group') {
      const dashboard = config.dashboards[getTabIndex(actionModal.tabId)];
      if (!dashboard) { closeActionModal(); showMessage('Tab not found.', 'is-danger'); return; }
      const groupIndex = dashboard.groups.findIndex((g) => g.id === actionModal.groupId);
      if (groupIndex < 0) { closeActionModal(); showMessage('Group not found.', 'is-danger'); return; }
      dashboard.groups.splice(groupIndex, 1);
      touchConfig(); closeActionModal();
      await persistConfig('Group deleted.');
      return;
    }

    if (mode === 'delete-button') {
      const dashboard = config.dashboards[getTabIndex(actionModal.tabId)];
      if (!dashboard) { closeActionModal(); showMessage('Tab not found.', 'is-danger'); return; }
      const group = dashboard.groups.find((g) => g.id === actionModal.groupId);
      if (!group) { closeActionModal(); showMessage('Group not found.', 'is-danger'); return; }
      const buttonIndex = group.entries.findIndex((e) => e.id === actionModal.buttonId);
      if (buttonIndex < 0) { closeActionModal(); showMessage('Button not found.', 'is-danger'); return; }
      group.entries.splice(buttonIndex, 1);
      touchConfig(); closeActionModal(); buttonModalOpen = false;
      await persistConfig('Button deleted.');
      return;
    }

    if (mode === 'delete-tab') {
      if ((config.dashboards || []).length <= 1) { closeActionModal(); showMessage('At least one tab is required.', 'is-danger'); return; }
      const index = getTabIndex(actionModal.tabId);
      if (index < 0) { closeActionModal(); showMessage('Tab not found.', 'is-danger'); return; }
      config.dashboards.splice(index, 1);
      if (activeTabId === actionModal.tabId) activeTabId = config.dashboards[0]?.id || '';
      touchConfig(); closeActionModal();
      await persistConfig('');
    }
  }

  // ─── Edit-mode button modal ───────────────────────────────────────────────────

  function openButtonModal(groupId, buttonId = '') {
    const dashboard = getActiveTab();
    const group = dashboard?.groups?.find((item) => item.id === groupId);
    if (!group) { showMessage('Group not found.', 'is-danger'); return; }
    let buttonEntry = null;
    if (buttonId) {
      buttonEntry = group.entries.find((item) => item.id === buttonId) || null;
      if (!buttonEntry) { showMessage('Button not found.', 'is-danger'); return; }
    }
    buttonModalIsNew = !buttonId;
    buttonModalGroupId = groupId;
    buttonModalButtonId = buttonId || '';
    buttonModalInitialData = {
      name: buttonEntry?.name || '', icon: buttonEntry?.icon || '',
      externalUrl: buttonEntry?.links?.external || '', internalUrl: buttonEntry?.links?.internal || '',
      iconData: buttonEntry?.iconData || '',
      iconMeta: buttonEntry?.iconMeta || null
    };
    buttonModalOpen = true;
  }

  function findButtonPosition(dashboard, buttonId) {
    for (let g = 0; g < (dashboard?.groups || []).length; g += 1) {
      const b = dashboard.groups[g].entries.findIndex((e) => e.id === buttonId);
      if (b >= 0) return { groupIndex: g, buttonIndex: b };
    }
    return null;
  }

  async function handleButtonModalSave(event) {
    hideMessage();
    const d = event.detail;
    const dashboard = config.dashboards[getTabIndex(activeTabId)];
    if (!dashboard) { showMessage('Tab not found.', 'is-danger'); return; }
    let groupIndex = dashboard.groups.findIndex((g) => g.id === d.groupId);
    if (groupIndex < 0) { showMessage('Group not found.', 'is-danger'); return; }
    let buttonEntry; let isNew = false;
    if (d.isNew) {
      buttonEntry = { id: StartpageCommon.createId('button'), name: '', icon: '', iconData: '', iconMeta: null, links: { external: '', internal: '' } };
      isNew = true;
    } else {
      const pos = findButtonPosition(dashboard, d.buttonId);
      if (!pos) { showMessage('Button not found.', 'is-danger'); return; }
      groupIndex = pos.groupIndex;
      buttonEntry = dashboard.groups[pos.groupIndex].entries[pos.buttonIndex];
    }
    buttonEntry.name = d.name; buttonEntry.icon = d.icon;
    buttonEntry.links = buttonEntry.links && typeof buttonEntry.links === 'object' ? buttonEntry.links : {};
    buttonEntry.links.external = d.externalUrl; buttonEntry.links.internal = d.internalUrl;
    buttonEntry.iconData = d.iconData;
    buttonEntry.iconMeta = d.iconMeta || null;
    if (isNew) dashboard.groups[groupIndex].entries.push(buttonEntry);
    touchConfig(); buttonModalOpen = false;
    await persistConfig('');

  }

  function handleButtonModalDeleteRequest(event) {
    const { groupId, buttonId, buttonName } = event.detail;
    openDeleteButtonModal(groupId, buttonId, buttonName);
  }

  // ─── Edit-mode DnD (Sortable.js) ──────────────────────────────────────────────

  function moveItem(items, oldIndex, newIndex) {
    if (!Array.isArray(items) || oldIndex == null || newIndex == null || oldIndex === newIndex) return false;
    const [item] = items.splice(oldIndex, 1);
    if (!item) return false;
    items.splice(newIndex, 0, item);
    return true;
  }

  function saveDndChange(errorMessage) {
    touchConfig();
    persistConfig('').catch((error) => {
      console.error(error);
      showMessage(errorMessage, 'is-danger');
    });
  }

  function sortableTabs(node, enabled) {
    const sortable = Sortable.create(node, {
      animation: DND_FLIP_DURATION_MS,
      draggable: 'li.tab-sort-item[data-dashboard-id]',
      handle: '.tab-drag-handle',
      forceFallback: true,
      disabled: !enabled,
      onEnd: (event) => {
        if (!moveItem(config.dashboards, event.oldDraggableIndex, event.newDraggableIndex)) return;
        ensureActiveTab();
        saveDndChange('Failed to reorder tabs.');
      }
    });
    return {
      update(nextEnabled) { sortable.option('disabled', !nextEnabled); },
      destroy() { sortable.destroy(); }
    };
  }

  function sortableGroups(node) {
    const sortable = Sortable.create(node, {
      group: { name: 'kiss-groups', pull: false, put: false },
      animation: DND_FLIP_DURATION_MS,
      draggable: 'section[data-group-sort-item][data-group-id]',
      handle: '.group-drag-handle',
      scroll: true,
      scrollSensitivity: 100,
      scrollSpeed: 10,
      onEnd: (event) => {
        const dashboard = getActiveTab();
        if (!dashboard || !moveItem(dashboard.groups, event.oldDraggableIndex, event.newDraggableIndex)) return;
        saveDndChange('Failed to reorder groups.');
      }
    });
    return { destroy() { sortable.destroy(); } };
  }

  function sortableButtons(node) {
    const sortable = Sortable.create(node, {
      group: { name: 'kiss-buttons', pull: true, put: true },
      animation: DND_FLIP_DURATION_MS,
      draggable: 'div[data-button-sort-item][data-button-id]',
      handle: '.button-drag-handle',
      scroll: true,
      scrollSensitivity: 100,
      scrollSpeed: 10,
      forceFallback: true,
      fallbackOnBody: true,
      onEnd: (event) => {
        const dashboard = getActiveTab();
        const source = dashboard?.groups?.find((group) => group.id === event.from?.dataset?.groupId);
        const target = dashboard?.groups?.find((group) => group.id === event.to?.dataset?.groupId);
        if (!source || !target || event.oldDraggableIndex == null || event.newDraggableIndex == null) return;
        if (source === target) {
          if (!moveItem(source.entries, event.oldDraggableIndex, event.newDraggableIndex)) return;
        } else {
          const [entry] = source.entries.splice(event.oldDraggableIndex, 1);
          if (!entry) return;
          target.entries.splice(event.newDraggableIndex, 0, entry);
        }
        saveDndChange('Failed to reorder buttons.');
      }
    });
    return { destroy() { sortable.destroy(); } };
  }

  // ─── Edit-mode group decorations ──────────────────────────────────────────────

  function decoratedEditorGroups() {
    const dashboard = getActiveTab();
    if (!dashboard) return [];
    // Use global theme (themeDraft) for button color settings
    const previewDashboard = { ...dashboard, ...themeDraft };
    const groups = Array.isArray(dashboard.groups) ? dashboard.groups : [];
    let colorIndex = 0;
    return groups.map((group) => {
      const entries = (Array.isArray(group.entries) ? group.entries : []).map((buttonEntry) => {
        const color = StartpageCommon.getButtonColorPair(previewDashboard, group, colorIndex);
        colorIndex += 1;
        return { id: buttonEntry.id, buttonEntry, color, iconSrc: iconSource(buttonEntry) };
      });
      return { id: group.id, group, entries };
    });
  }

  function accountLinkLabel() {
    if (authMustChangePassword) return accountPaneOpen ? 'Close Setup' : 'Finish Setup';
    return accountPaneOpen ? 'Close Account' : 'Account';
  }

  // ─── Lifecycle ────────────────────────────────────────────────────────────────

  onDestroy(() => {
    if (configSaveTimer) clearTimeout(configSaveTimer);
    clearMessageTimer();
    resizeObserver?.disconnect();
  });

  onMount(() => {
    const pathname = window.location.pathname || '/';
    // Normalize legacy URLs
    if (pathname === '/index.html' && window.history?.replaceState) window.history.replaceState(null, '', '/');
    else if (/^\/(?:edit\.html|admin(?:\.html)?)/.test(pathname) && window.history?.replaceState) window.history.replaceState(null, '', '/edit');

    const startInEditMode = /^\/edit\/?$/.test(window.location.pathname);

    loadConfig().then(() => {
      if (startInEditMode) enterEditMode();
    });
    StartpageCommon.fetchVersion().then(v => { appVersion = v; });

    const onResize = () => { applyStartpageTheme(); updateViewportState(); updateTabsOverflowState(); };
    window.addEventListener('resize', onResize);
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(() => updateTabsOverflowState());
      if (tabsScrollEl) resizeObserver.observe(tabsScrollEl);
      if (tabsListEl) resizeObserver.observe(tabsListEl);
    }
    updateViewportState();

    return () => {
      window.removeEventListener('resize', onResize);
      clearMessageTimer();
    };
  });

  // ─── Reactive declarations ────────────────────────────────────────────────────

  $: { config; activeTabId; activeTab = getActiveTab(); }
  $: { activeTab; currentLinkMode; viewGroups = (!editMode && activeTab) ? buttonDecorations(activeTab) : []; }
  $: { editMode; authenticated; config; activeTabId; themeDraft; editorGroups = (editMode && authenticated) ? decoratedEditorGroups() : []; }
  $: { config; builtInThemePresets = getResolvedBuiltInThemePresets(); savedThemePresets = getSavedThemePresets(); }
  $: { themeDraft; themeButtonMode = StartpageCommon.normalizeButtonColorMode(themeDraft.buttonColorMode); }
  $: showLinkToggle = !editMode && Boolean(activeTab?.enableInternalLinks);
  $: showEditInToolbar = true;
  $: { loading; activeTab; config; if (!loading && activeTab && !editMode) { applyPageTitle(); applyStartpageTheme(); } }
</script>

<svelte:head>
  <style data-kiss-route="native-hover">
    .entry-button { background-color: var(--entry-btn-base, transparent); }
    .entry-button:hover { background-color: var(--entry-btn-hover, var(--entry-btn-base, transparent)); }
    .entry-preview-button { background-color: var(--entry-btn-base, transparent); }
    .entry-preview-button:hover { background-color: var(--entry-btn-hover, var(--entry-btn-base, transparent)); }
  </style>
</svelte:head>

<section class="section">
  <div class={editMode ? 'admin-shell' : 'startpage-shell'}>

    {#if editMode}
      <div id="messageBox" class={`notification ${messageVisible ? '' : 'is-hidden'} ${messageTone}`.trim()}>{messageText}</div>
    {/if}

    <!-- ─── Shared toolbar ──────────────────────────────────────────── -->
    <div class="toolbar {editMode ? 'mode-tabs-toolbar' : ''}">
      <div class="tabs is-boxed mode-tabs">
        <div bind:this={tabsRowEl} class="mode-tabs-row">
          <div bind:this={tabsScrollEl} class="mode-tabs-scroll">
            <ul bind:this={tabsListEl} id="mainTabsList" use:sortableTabs={editMode && authenticated}>
              {#each config.dashboards as dashboard (dashboard.id)}
                <li
                  class={`${dashboard.id === activeTabId ? 'is-active ' : ''}${editMode ? 'tab-sort-item' : ''}`.trim()}
                  data-tab-sort-item={editMode ? '' : undefined}
                  data-dashboard-id={editMode ? dashboard.id : undefined}
                  animate:flip={{ duration: DND_FLIP_DURATION_MS }}
                >
                  <a href="/" role="button" on:click|preventDefault={() => setActiveTab(dashboard.id)}>
                    {#if editMode && authenticated}
                      <span class="drag-handle tab-drag-handle" title={`Drag to reorder tab ${dashboard.label}`} aria-label={`Drag to reorder tab ${dashboard.label}`}>⋮⋮</span>
                    {/if}
                    <span class="tab-link-label">{dashboard.label}</span>
                  </a>
                </li>
              {/each}
              {#if editMode && authenticated}
                <li class="add-tab">
                  <a href="/edit" title="Add tab" aria-label="Add tab" on:click|preventDefault={openAddTabModal}>+</a>
                </li>
              {/if}
            </ul>
          </div>
        </div>
      </div>

      <!-- Link mode toggle + overflow edit toggle -->
      {#if showLinkToggle || showEditInToolbar}
        <div class="toolbar-mode-row">
          {#if showLinkToggle}
            <div class="toolbar-mode-left">
              <div class="mode-switch startpage-link-mode">
                <span class="mode-switch-label">Use internal links</span>
                <label class="ios-switch" for="linkModeToggle">
                  <input id="linkModeToggle" type="checkbox" checked={currentLinkMode === 'internal'} on:change={(e) => setLinkMode(e.currentTarget.checked ? 'internal' : 'external')} aria-label="Switch between external and internal links" />
                  <span class="ios-switch-slider"></span>
                </label>
              </div>
            </div>
          {/if}
          {#if showEditInToolbar}
            <div class="toolbar-mode-actions">
              <div class="mode-switch startpage-link-mode nav-edit-mode-toggle" role="button" tabindex="0"
                on:click={(e) => { if (!e.target?.closest?.('.ios-switch')) toggleEditMode(); }}
                on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleEditMode(); } }}>
                <span class="mode-switch-label">Edit</span>
                <label class="ios-switch">
                  <input type="checkbox" checked={editMode} on:change={toggleEditMode} aria-label="Toggle Edit" />
                  <span class="ios-switch-slider"></span>
                </label>
              </div>
            </div>
          {/if}
        </div>
      {/if}
    </div>

    <!-- ─── Edit-mode: admin topbar ─────────────────────────────────── -->
    {#if editMode && authenticated}
      <div class="admin-topbar-shell mt-3 mb-5">
        <div class="admin-topbar">
          <div class="admin-topbar-left">
            <h1 class="title is-4 mb-0">KISS Startpage</h1>
            <button id="accountLinkBtn" class={`button is-small account-link-btn ${authMustChangePassword ? 'is-warning' : 'is-link'} is-light`.trim()} type="button" on:click={() => (accountPaneOpen = !accountPaneOpen)}>{accountLinkLabel()}</button>
          </div>
          <div class="admin-topbar-actions">
            <button id="logoutBtn" class="button is-danger is-light is-small" type="button" on:click={logoutSubmit}>Log Out</button>
          </div>
        </div>
      </div>
    {/if}

    <!-- ─── View mode: startpage ─────────────────────────────────────── -->
    {#if !editMode}
      <div id="startpageContent">
        {#if loading}
          <div class="empty-state">Loading startpage…</div>
        {:else if loadError}
          <div class="empty-state load-error-state">
            <p>{loadError}</p>
            <button class="button is-link is-light mt-3" type="button" on:click={loadConfig}>Try again</button>
          </div>
        {:else if !activeTab || !viewGroups.length}
          <div class="empty-state">No groups configured yet. Flip the Edit toggle to add one.</div>
        {:else}
          {#each viewGroups as item (item.group.id)}
            <section class="group-box" class:group-end={item.group?.groupEnd} data-group-id={item.group.id}>
              <h2 class="title is-6 group-title">{item.group?.title || 'Untitled Group'}</h2>
              <div class="columns is-mobile is-multiline entry-grid">
                {#each item.entries as decorated (decorated.entry.id)}
                  <div class="column is-half-mobile is-one-third-tablet is-one-quarter-desktop" data-button-id={decorated.entry.id}>
                    <a
                      class="button is-fullwidth entry-button"
                      href={decorated.href || '#'}
                      target={decorated.href && activeTab?.openLinksInNewTab !== false ? '_blank' : undefined}
                      rel={decorated.href && activeTab?.openLinksInNewTab !== false ? 'noopener noreferrer' : undefined}
                      class:is-static={!decorated.href}
                      style={`--entry-btn-base:${decorated.color.base};--entry-btn-hover:${decorated.color.hover};`}
                      on:click={(e) => { if (!decorated.href) e.preventDefault(); }}
                    >
                      <span class="icon entry-icon">
                        {#if decorated.iconSrc}<img src={decorated.iconSrc} alt={`${decorated.entry?.name || 'Button'} icon`} />{/if}
                      </span>
                      <span class="entry-label">{decorated.entry?.name || 'Unnamed'}</span>
                    </a>
                  </div>
                {/each}
              </div>
            </section>
          {/each}
        {/if}
      </div>
    {/if}

    <!-- ─── Edit mode: login or admin panel ──────────────────────────── -->
    {#if editMode}
      {#if !authenticated}
        <LoginView
          bind:authSetupRequired
          showMessage={showMessage}
          on:loginsuccess={async (e) => {
            authUser = e.detail.user;
            authMustChangePassword = Boolean(e.detail.mustChangePassword);
            authSetupRequired = false;
            authenticated = true;
            syncDraftsFromActiveTab();
            applyCurrentAdminThemePreview();
            if (authMustChangePassword) {
              accountPaneOpen = true;
            }
          }}
        />
      {:else}
        <div id="editorPane">
          <div class="box panel-box startpage-settings-box">
            <div class="settings-head">
              <h2 class="title is-5">Tab Settings</h2>
              <div class="settings-actions">
                <button id="liveColorEditorToggleBtn" class="button is-link is-light is-small" type="button" on:click={() => (showThemeEditor = !showThemeEditor)}>{showThemeEditor ? 'Hide theme editor' : 'Show theme editor'}</button>
                <button id="deleteDashboardBtn" class="button is-danger is-light is-small delete-startpage-btn" type="button" disabled={(config.dashboards || []).length <= 1} on:click={openDeleteTabModal}>Delete tab</button>
              </div>
            </div>
            <div class="field mb-3">
              <label class="label" for="pageTitleInput">Tab Title</label>
              <div class="title-editor">
                <input id="pageTitleInput" class="input" type="text" maxlength="80" placeholder="Startpage 1" bind:value={pageTitleDraft} on:keydown={(e)=> e.key === 'Enter' && e.currentTarget.blur()} on:blur={() => saveTabTitle().catch((err) => { console.error(err); showMessage('Failed to update tab title.', 'is-danger'); })} />
              </div>
            </div>
            <div class="field mb-3">
              <div class="mode-switch mode-switch-plain">
                <label class="ios-switch" for="enableInternalLinksCheckbox">
                  <input id="enableInternalLinksCheckbox" type="checkbox" bind:checked={enableInternalLinksDraft} on:change={() => saveActiveTabSettings().catch((err) => { console.error(err); showMessage('Failed to update tab settings.', 'is-danger'); })} aria-label="Use internal and external links" />
                  <span class="ios-switch-slider"></span>
                </label>
                <span class="mode-switch-label">Use internal and external links</span>
              </div>
            </div>
            <div class="field mb-3">
              <div class="mode-switch mode-switch-plain">
                <label class="ios-switch" for="openLinksInNewTabCheckbox">
                  <input id="openLinksInNewTabCheckbox" type="checkbox" bind:checked={openLinksInNewTabDraft} on:change={() => saveActiveTabSettings().catch((err) => { console.error(err); showMessage('Failed to update tab settings.', 'is-danger'); })} aria-label="Open links in a new tab" />
                  <span class="ios-switch-slider"></span>
                </label>
                <span class="mode-switch-label">Open links in a new tab</span>
              </div>
            </div>

            <div id="liveColorEditorPanel" class={`live-color-editor-panel ${showThemeEditor ? '' : 'is-hidden'}`.trim()}>
              <div class="live-color-editor-head">
                <h3 class="title is-5">Theme Editor</h3>

              </div>
              <ThemeEditor
                theme={themeDraft}
                builtinPresets={builtInThemePresets}
                savedPresets={savedThemePresets}
                presetName={themePresetName}
                on:update={handleThemeUpdate}
                on:savePreset={handleSavePreset}
                on:loadPreset={handleLoadPreset}
                on:deletePreset={handleDeletePreset}
              />
            </div>
          </div>

          <div id="groupsEditor" use:sortableGroups>
              {#each editorGroups as row (row.group.id)}
                <section
                  class={`box group-box ${row.group.groupEnd ? 'group-end' : ''}`.trim()}
                  data-group-sort-item=""
                  data-group-id={row.group.id}
                  animate:flip={{ duration: DND_FLIP_DURATION_MS }}
                >
                  <div class="group-head">
                    <div class="group-head-main">
                      <button type="button" class="drag-handle group-drag-handle" title={`Drag to reorder group ${row.group.title || 'group'}`} aria-label={`Drag to reorder group ${row.group.title || 'group'}`}>⋮⋮</button>
                      <input
                        type="text" class="input group-title-input" maxlength="80"
                        value={row.group.title} placeholder="Group title"
                        on:keydown={(e)=> e.key === 'Enter' && e.currentTarget.blur()}
                        on:blur={(e)=> updateGroupTitle(row.group, e.currentTarget.value).catch((err) => { console.error(err); showMessage('Failed to update group title.', 'is-danger'); })}
                      />
                      {#if themeButtonMode === 'solid-per-group'}
                        <div class="group-button-color-inline">
                          <span class="group-button-color-inline-label">Button Color</span>
                          <input type="color" aria-label={`Button color for group ${row.group.title || 'group'}`} value={getGroupButtonSolidColor(row.group)} style="border: 2px solid {contrastBorder(getGroupButtonSolidColor(row.group))};" on:input={(e)=> setGroupButtonSolidColor(row.group, e.currentTarget.value, false)} on:change={(e)=> setGroupButtonSolidColor(row.group, e.currentTarget.value, true)} />
                          <input class="input" type="text" maxlength="7" inputmode="text" placeholder={DEFAULT_BUTTON_COLOR_OPTIONS.buttonSolidColor} value={getGroupButtonSolidColor(row.group)} on:keydown={(e)=> e.key === 'Enter' && e.currentTarget.blur()} on:input={(e)=> { const n = normalizeHexColorLoose(e.currentTarget.value); if (n) setGroupButtonSolidColor(row.group, n, false); }} on:blur={(e)=> setGroupButtonSolidColor(row.group, e.currentTarget.value, true)} aria-label={`Button color code for group ${row.group.title || 'group'}`} />
                        </div>
                      {/if}
                      <button type="button" class="button is-danger is-light is-small group-delete-btn" on:click={() => openDeleteGroupModal(row.group)}>Delete group</button>
                    </div>
                  </div>

                  <div class="columns is-mobile is-multiline entry-grid" data-button-sort-container="" data-group-id={row.group.id} use:sortableButtons>
                    {#each row.entries as cell (cell.buttonEntry.id)}
                      <div
                        class="column is-half-mobile is-one-third-tablet is-one-quarter-desktop"
                        data-button-sort-item=""
                        data-button-id={cell.buttonEntry.id}
                      >
                        <div class="entry-admin-card">
                          <button
                            type="button" class="button is-fullwidth entry-preview-button"
                            title={`Edit button: ${cell.buttonEntry.name}`}
                            aria-label={`Edit button ${cell.buttonEntry.name}`}
                            style={`--entry-btn-base:${cell.color.base};--entry-btn-hover:${cell.color.hover};`}
                            on:click={() => openButtonModal(row.group.id, cell.buttonEntry.id)}
                          >
                            <span class="drag-handle button-drag-handle" title={`Drag to reorder button ${cell.buttonEntry.name || 'button'}`} aria-label={`Drag to reorder button ${cell.buttonEntry.name || 'button'}`}>⋮⋮</span>
                            <span class="icon entry-icon">{#if cell.iconSrc}<img src={cell.iconSrc} alt={`${cell.buttonEntry.name} icon`} />{/if}</span>
                            <span class="entry-label">{cell.buttonEntry.name}</span>
                            <span class="entry-preview-edit-icon" aria-hidden="true">✎</span>
                          </button>
                        </div>
                      </div>
                    {/each}
                    <div class="column is-half-mobile is-one-third-tablet is-one-quarter-desktop" data-entry-add-slot="">
                      <div class="entry-admin-card">
                        <button type="button" class="button is-fullwidth entry-add-button" title="Add button" aria-label="Add button" on:click={() => openButtonModal(row.group.id)}>+</button>
                      </div>
                    </div>
                  </div>
                </section>
              {/each}
          </div>

          <div class="box panel-box group-add-panel">
            <button id="addGroupBtn" class="button is-fullwidth group-add-panel-button" type="button" title="Add group" aria-label="Add group" on:click={openAddGroupModal}>+</button>
          </div>
        </div>
      {/if}
    {/if}

  {#if appVersion && editMode}
    <div class="app-version-badge">v{appVersion}</div>
  {/if}
  </div>
</section>

<!-- ─── Modals (edit mode only) ──────────────────────────────────────── -->
{#if editMode}
  <ButtonModal
    open={buttonModalOpen}
    isNew={buttonModalIsNew}
    groupId={buttonModalGroupId}
    buttonId={buttonModalButtonId}
    initialData={buttonModalInitialData}
    internalLinksEnabled={enableInternalLinksDraft}
    showMessage={showMessage}
    on:close={() => (buttonModalOpen = false)}
    on:save={(e) => handleButtonModalSave(e).catch((err) => { console.error(err); showMessage('Failed to save button.', 'is-danger'); })}
    on:deleterequest={handleButtonModalDeleteRequest}
  />

  <AccountPane
    open={accountPaneOpen}
    authUser={authUser}
    authMustChangePassword={authMustChangePassword}
    showMessage={showMessage}
    on:close={() => (accountPaneOpen = false)}
    on:usernamechanged={(e) => { authUser = e.detail.user; }}
    on:passwordchanged={(e) => { authMustChangePassword = Boolean(e.detail.mustChangePassword); if (!authMustChangePassword) accountPaneOpen = false; }}
  />

  <div id="groupActionModal" class={`modal ${actionModal.open ? 'is-active' : ''}`.trim()}>
    <button type="button" class="modal-background" aria-label="Close dialog" on:click={closeActionModal}></button>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">{actionModal.title}</p>
      </header>
      <section class="modal-card-body">
        <p class="mb-3">{actionModal.text}</p>
        <div class={`field ${actionModal.titleFieldVisible ? '' : 'is-hidden'}`.trim()}>
          <label class="label" for="groupActionModalTitleInput">{actionModal.titleLabel}</label>
          <input id="groupActionModalTitleInput" class="input" type="text" maxlength="80" placeholder={actionModal.titlePlaceholder} bind:value={actionModal.titleValue} on:keydown={(e)=> { if (e.key === 'Enter') { e.preventDefault(); confirmActionModal().catch((err) => { console.error(err); showMessage('Action failed.', 'is-danger'); }); } }} />
        </div>
      </section>
      <footer class="modal-card-foot is-justify-content-flex-end">
        <div class="buttons">
          <button class="button" type="button" on:click={closeActionModal}>Cancel</button>
          <button class={`button ${actionModal.confirmTone}`.trim()} type="button" on:click={() => confirmActionModal().catch((err) => { console.error(err); showMessage('Action failed.', 'is-danger'); })}>{actionModal.confirmLabel}</button>
        </div>
      </footer>
    </div>
  </div>
{/if}
