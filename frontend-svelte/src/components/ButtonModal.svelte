<script>
  import { createEventDispatcher } from 'svelte';
  import { StartpageCommon, readFileAsDataUrl } from '../lib/startpage-common.js';

  export let open = false;
  export let isNew = false;
  export let groupId = '';
  export let buttonId = '';
  export let initialData = { name: '', icon: '', externalUrl: '', internalUrl: '', iconData: '', iconMeta: null };
  export let internalLinksEnabled = false;
  export let showMessage = () => {};

  const dispatch = createEventDispatcher();
  const initialSearchPrompt = 'Enter at least 2 characters, or enter a URL for its website icon.';

  let name = '';
  let icon = '';
  let externalUrl = '';
  let internalUrl = '';
  let iconData = '';
  let iconMeta = null;
  let clearIconData = false;
  let uploadedFileName = '';
  let uploadEl;

  let searchQuery = '';
  let searchGroups = [];
  let searchStatus = initialSearchPrompt;
  let searchStatusTone = 'muted';
  let searchLoading = false;
  let searchTimer = null;
  let searchSeq = 0;
  let selectedIconKey = '';
  let importingIconKey = '';
  let iconProviders = [];
  let iconifyCollections = [];
  let enabledProviders = [];
  let preferenceSaving = false;
  let iconCatalogueStatus = 'loading';

  $: if (open) initDraft();

  $: previewSrc = clearIconData ? '' : (iconData || (icon.trim() ? 'icons/' + icon.trim() : ''));
  $: iconFilenameVisible = !isNew || Boolean(icon.trim() || iconData);

  function initDraft() {
    name = initialData.name || '';
    icon = initialData.icon || '';
    externalUrl = initialData.externalUrl || '';
    internalUrl = initialData.internalUrl || '';
    iconData = initialData.iconData || '';
    iconMeta = initialData.iconMeta && typeof initialData.iconMeta === 'object' ? { ...initialData.iconMeta } : null;
    clearIconData = false;
    uploadedFileName = '';
    iconProviders = [];
    iconifyCollections = [];
    enabledProviders = [];
    preferenceSaving = false;
    iconCatalogueStatus = 'loading';
    searchQuery = '';
    searchGroups = [];
    searchStatusTone = 'muted';
    searchStatus = initialSearchPrompt;
    searchLoading = false;
    selectedIconKey = iconMeta?.provider && iconMeta?.reference ? iconResultKey(iconMeta) : '';
    importingIconKey = '';
    searchSeq += 1;
    clearSearchTimer();
    if (uploadEl) uploadEl.value = '';
    loadIconSources();
  }

  async function loadIconSources() {
    try {
      const [catalogue, preferences] = await Promise.all([
        StartpageCommon.listIconSources(),
        StartpageCommon.getIconPreferences()
      ]);
      const providers = Array.isArray(preferences?.providers)
        ? preferences.providers.filter((item) => item && item.id && item.label)
        : (Array.isArray(catalogue?.providers) ? catalogue.providers : []);
      if (providers.length) {
        iconProviders = providers;
        iconifyCollections = Array.isArray(catalogue?.items)
          ? catalogue.items.filter((item) => item?.provider === 'iconify')
          : [];
        enabledProviders = Array.isArray(preferences?.enabledProviders)
          ? preferences.enabledProviders.filter((id) => providers.some((provider) => provider.id === id))
          : providers.map((provider) => provider.id);
        iconCatalogueStatus = 'ready';
        if (searchQuery.trim().length >= 2 || externalUrl.trim() || internalUrl.trim()) {
          scheduleSearch(0);
        }
        return;
      }
      throw new Error('The icon catalogue is empty.');
    } catch (error) {
      console.warn('Icon source catalogue unavailable.', error);
      iconProviders = [];
      iconifyCollections = [];
      enabledProviders = [];
      iconCatalogueStatus = 'error';
      searchStatusTone = 'has-text-danger';
      searchStatus = 'The icon catalogue could not be loaded. Try again.';
    }
  }

  function clearSearchTimer() {
    if (searchTimer) {
      clearTimeout(searchTimer);
      searchTimer = null;
    }
  }

  function scheduleSearch(delayMs = 220) {
    clearSearchTimer();
    searchGroups = [];
    if (searchQuery.trim().length < 2 && !externalUrl.trim() && !internalUrl.trim()) {
      searchLoading = false;
      searchStatusTone = 'muted';
      searchStatus = initialSearchPrompt;
      return;
    }
    searchTimer = setTimeout(() => {
      searchTimer = null;
      runSearch().catch((error) => {
        console.error(error);
        searchStatusTone = 'has-text-danger';
        searchStatus = 'Icon search failed.';
        showMessage(error.message || 'Icon search failed.', 'is-danger');
      });
    }, delayMs);
  }

  async function runSearch() {
    const query = (searchQuery || '').trim();
    const seq = searchSeq + 1;
    searchSeq = seq;

    if (query.length < 2 && !externalUrl.trim() && !internalUrl.trim()) {
      searchGroups = [];
      searchStatusTone = 'muted';
      searchStatus = initialSearchPrompt;
      return;
    }

    if (!enabledProviders.length) {
      searchGroups = [];
      searchStatusTone = 'has-text-warning';
      searchStatus = 'Enable at least one icon source.';
      return;
    }

    if (iconCatalogueStatus !== 'ready') {
      searchGroups = [];
      searchStatusTone = 'has-text-danger';
      searchStatus = 'The icon catalogue is not ready.';
      return;
    }

    searchLoading = true;
    searchStatusTone = 'muted';
    searchStatus = query.length >= 2
      ? `Searching "${query}" across ${enabledProviders.length} enabled source${enabledProviders.length === 1 ? '' : 's'}...`
      : 'Discovering website icons...';
    try {
      const payload = await StartpageCommon.searchIcons(query, 18, {
        externalUrl: externalUrl.trim(),
        internalUrl: internalUrl.trim()
      });
      if (searchSeq !== seq) return;
      const groups = Array.isArray(payload?.groups) ? payload.groups : [];
      searchGroups = groups.filter((group) =>
        group?.status === 'error' || (Array.isArray(group?.items) && group.items.length > 0)
      );
      const itemCount = groups.reduce((total, group) => total + (Array.isArray(group?.items) ? group.items.length : 0), 0);
      const resultGroupCount = groups.filter((group) => Array.isArray(group?.items) && group.items.length > 0).length;
      const errorCount = groups.filter((group) => group?.status === 'error').length;
      searchStatusTone = itemCount ? 'muted' : (errorCount ? 'has-text-danger' : 'has-text-warning');
      searchStatus = itemCount
        ? `Found ${itemCount} icon${itemCount === 1 ? '' : 's'} across ${resultGroupCount} source${resultGroupCount === 1 ? '' : 's'}${errorCount ? `; ${errorCount} source${errorCount === 1 ? '' : 's'} unavailable` : ''}.`
        : (errorCount ? 'No icons found; one or more icon sources are unavailable.' : 'No icons found.');
    } catch (error) {
      if (searchSeq !== seq) return;
      searchGroups = [];
      searchStatusTone = 'has-text-danger';
      searchStatus = 'Icon search unavailable.';
      throw error;
    } finally {
      if (searchSeq === seq) {
        searchLoading = false;
      }
    }
  }

  async function importIcon(item) {
    const reference = String(item?.reference || '').trim();
    const provider = String(item?.provider || '').trim();
    const nextIconKey = iconResultKey(item);
    importingIconKey = nextIconKey;
    searchStatusTone = 'muted';
    searchStatus = `Importing "${reference}" from ${providerLabel(provider)}...`;
    try {
      const payload = await StartpageCommon.importIcon(provider, reference, 'svg');

      iconData = payload?.iconData || '';
      if (payload?.icon) {
        icon = payload.icon;
      } else if (reference) {
        icon = `${reference}.svg`;
      }
      iconMeta = {
        provider,
        reference: payload?.reference || reference,
        license: payload?.license || item?.license || '',
        licenseUrl: payload?.licenseUrl || item?.licenseUrl || '',
        sourceUrl: payload?.sourceUrl || item?.sourceUrl || ''
      };
      selectedIconKey = iconResultKey(iconMeta);
      clearIconData = false;
      if (uploadEl) uploadEl.value = '';
      uploadedFileName = '';
      searchStatusTone = 'has-text-success';
      searchStatus = `Selected ${payload?.name || reference}. Click Save to store it in this button.`;
    } finally {
      importingIconKey = '';
    }
  }

  function iconResultKey(item) {
    return `${String(item?.provider || '').trim()}::${String(item?.reference || '').trim()}`;
  }

  async function handleUploadChange(event) {
    const file = event.currentTarget?.files?.[0] || null;
    if (!file) {
      uploadedFileName = '';
      return;
    }
    uploadedFileName = file.name;
    if (!isNew && !icon.trim()) {
      icon = file.name;
    }
    try {
      const dataUrl = await readFileAsDataUrl(file);
      iconData = dataUrl;
      iconMeta = null;
      selectedIconKey = '';
      clearIconData = false;
    } catch (error) {
      console.error(error);
      showMessage('Failed to preview uploaded icon.', 'is-danger');
    }
  }

  async function pasteText(callback, label) {
    if (!navigator.clipboard?.readText) {
      showMessage('Clipboard paste is not available in this browser.', 'is-danger');
      return;
    }
    try {
      const text = await navigator.clipboard.readText();
      callback((text || '').trim());
      if (label) showMessage(`${label} pasted.`, 'is-success');
    } catch (error) {
      console.error(error);
      showMessage(error.message || 'Failed to paste.', 'is-danger');
    }
  }

  function providerLabel(providerId) {
    return iconProviders.find((item) => item.id === providerId)?.label || providerId || 'Icon source';
  }

  async function toggleProvider(providerId, checked) {
    const previous = [...enabledProviders];
    enabledProviders = checked
      ? iconProviders.map((provider) => provider.id).filter((id) => id === providerId || previous.includes(id))
      : previous.filter((id) => id !== providerId);
    preferenceSaving = true;
    try {
      const payload = await StartpageCommon.saveIconPreferences(enabledProviders);
      enabledProviders = Array.isArray(payload?.enabledProviders) ? payload.enabledProviders : enabledProviders;
      scheduleSearch(80);
    } catch (error) {
      enabledProviders = previous;
      showMessage(error.message || 'Failed to save icon source preferences.', 'is-danger');
    } finally {
      preferenceSaving = false;
    }
  }

  function handleClose() {
    clearSearchTimer();
    dispatch('close');
  }

  function handleSave() {
    const hasAnyInput = Boolean(
      name.trim() || icon.trim() || externalUrl.trim() || internalUrl.trim() || clearIconData || uploadedFileName
    );
    if (isNew && !hasAnyInput) {
      dispatch('close');
      return;
    }
    if (!name.trim()) {
      showMessage('Button name is required.', 'is-danger');
      return;
    }
    if (!externalUrl.trim() && !internalUrl.trim()) {
      showMessage('At least one URL (external or internal) is required.', 'is-danger');
      return;
    }
    const finalIcon = icon.trim() || (uploadedFileName && !icon.trim() ? uploadedFileName : '');
    dispatch('save', {
      isNew,
      groupId,
      buttonId,
      name: name.trim(),
      icon: finalIcon,
      externalUrl: externalUrl.trim(),
      internalUrl: internalUrl.trim(),
      iconData: clearIconData ? '' : iconData,
      iconMeta: clearIconData ? null : iconMeta,
      clearIconData
    });
  }

  function handleDelete() {
    dispatch('deleterequest', { groupId, buttonId, buttonName: name });
  }
</script>

<div id="entryModal" class={`modal ${open ? 'is-active' : ''}`.trim()}>
  <button type="button" class="modal-background" aria-label="Close dialog" on:click={handleClose}></button>
  <div class="modal-card">
    <header class="modal-card-head">
      <div class="entry-modal-head">
        <p id="entryModalTitle" class="modal-card-title">{isNew ? 'Add Button' : `Edit Button: ${name || ''}`}</p>
        <div class="entry-modal-head-actions">
          <button id="entryCloseBtn" class="button icon-action" type="button" title="Close" aria-label="Close" on:click={handleClose}>✕</button>
        </div>
      </div>
    </header>
    <section class="modal-card-body">
      <div class="columns is-multiline">
        <div class="column is-12">
          <label class="label" for="entryNameInput">Button Name</label>
          <input id="entryNameInput" class="input" type="text" maxlength="80" required bind:value={name} />
        </div>

        <div id="entryExternalUrlColumn" class={`column ${internalLinksEnabled ? 'is-6' : 'is-12'}`}>
          <label class="label" for="entryExternalUrlInput">External URL</label>
          <div class="field url-input-row">
            <p class="control is-expanded"><input id="entryExternalUrlInput" class="input" type="url" placeholder="https://example.com" bind:value={externalUrl} on:input={() => scheduleSearch(450)} /></p>
          </div>
        </div>

        <div id="entryInternalUrlColumn" class={`column is-6 ${internalLinksEnabled ? '' : 'is-hidden'}`.trim()}>
          <label class="label" for="entryInternalUrlInput">Internal URL</label>
          <div class="field url-input-row">
            <p class="control is-expanded"><input id="entryInternalUrlInput" class="input" type="url" placeholder="http://192.168.x.x:port" bind:value={internalUrl} on:input={() => scheduleSearch(450)} disabled={!internalLinksEnabled} /></p>
          </div>
        </div>

        <div id="entryUrlHelpColumn" class="column is-12">
          <p id="entryUrlHelpText" class="help">
            {internalLinksEnabled ? 'At least one URL (external or internal) is required.' : 'External URL is required (internal links are disabled for this tab).'}
          </p>
        </div>

        <div id="entryIconFilenameColumn" class={`column is-6 ${iconFilenameVisible ? '' : 'is-hidden'}`.trim()}>
          <label class="label" for="entryIconInput">Icon Filename</label>
          <input id="entryIconInput" class="input" type="text" maxlength="120" placeholder="example.svg" bind:value={icon} disabled={isNew} class:is-disabled={isNew} />
          <p id="entryIconInputHint" class={`help mt-2 ${isNew ? '' : 'is-hidden'}`.trim()}>Available after the button is saved.</p>
          <p id="entryIconPreview" class="help mt-2">
            {#if previewSrc}
              <img class="icon-preview" src={previewSrc} alt="icon preview" /> {iconData ? 'icon embedded' : 'icon path'}
            {:else}
              No icon configured
            {/if}
          </p>
        </div>

        <div class="column is-6">
          <label class="label" for="entryIconUpload">Upload Icon</label>
          <input id="entryIconUpload" class="input" type="file" accept="image/*" bind:this={uploadEl} on:change={handleUploadChange} />
          <label class="checkbox mt-2">
            <input id="entryClearIconData" type="checkbox" bind:checked={clearIconData} on:change={() => { if (clearIconData) selectedIconKey = ''; }} />
            Clear embedded uploaded icon
          </label>
        </div>

        <div class="column is-12">
          <label class="label" for="entryIconSearchInput">Search Icon Libraries</label>
          {#if iconCatalogueStatus === 'error'}
            <div class="icon-catalogue-error">
              <p class="help has-text-danger">The icon catalogue could not be loaded.</p>
              <button class="button is-small" type="button" on:click={loadIconSources}>Try again</button>
            </div>
          {:else}
            <div class="icon-provider-options" aria-label="Icon sources">
              {#each iconProviders as provider (provider.id)}
                <label class="icon-provider-option" title={provider.description || provider.label}>
                  <input
                    type="checkbox"
                    checked={enabledProviders.includes(provider.id)}
                    disabled={iconCatalogueStatus !== 'ready' || preferenceSaving}
                    on:change={(event) => toggleProvider(provider.id, event.currentTarget.checked)}
                  />
                  <span>
                    <span class="icon-provider-option-label">{provider.label}</span>
                    {#if provider.id === 'iconify'}
                      <span class="icon-provider-option-detail">{iconifyCollections.length} collections</span>
                    {/if}
                  </span>
                </label>
              {/each}
            </div>
            <p class="help icon-provider-save-status">{preferenceSaving ? 'Saving source selection…' : 'Source selection is stored for your account on this server.'}</p>
          {/if}
          <div class="icon-search-controls">
            <div class="control is-expanded">
              <input id="entryIconSearchInput" class="input" type="search" maxlength="80" placeholder="Search icons (e.g. home, music, cloud, server)" bind:value={searchQuery} on:input={() => scheduleSearch(220)} disabled={iconCatalogueStatus !== 'ready'} />
            </div>
          </div>
          <p class="help mt-2">All enabled sources are searched together. The selected icon is embedded locally in your startpage config.</p>
          <p id="entryIconSearchStatus" class={`help mt-2 ${searchStatusTone}`.trim()}>{searchStatus}</p>
          <div id="entryIconSearchResults" class="icon-search-groups">
            {#each searchGroups as group (group.provider)}
              <section class="icon-search-group">
                <div class="icon-search-group-head">
                  <h4>{group.label}</h4>
                  {#if group.status === 'error'}
                    <span class="icon-provider-status is-error">{group.message || 'Source unavailable.'}</span>
                  {/if}
                </div>
                {#if Array.isArray(group.items) && group.items.length}
                  <div class="icon-search-results">
                    {#each group.items as item (iconResultKey(item))}
                      <button
                        type="button"
                        class="icon-search-result"
                        class:is-selected={selectedIconKey === iconResultKey(item)}
                        class:is-importing={importingIconKey === iconResultKey(item)}
                        title={selectedIconKey === iconResultKey(item) ? `${item.name} selected` : `Use ${item.name}`}
                        aria-label={`Use icon ${item.name}`}
                        aria-pressed={selectedIconKey === iconResultKey(item)}
                        disabled={Boolean(importingIconKey)}
                        on:click={() => importIcon(item).catch((error) => { console.error(error); searchStatusTone = 'has-text-danger'; searchStatus = 'Failed to import icon.'; showMessage(error.message || 'Failed to import icon.', 'is-danger'); })}
                      >
                        <span class="icon-search-result-media">
                          {#if item.previewUrl}
                            <img src={item.previewUrl} alt="" loading="lazy" />
                          {:else}
                            ◻
                          {/if}
                        </span>
                        <span class="icon-search-result-meta">
                          <span class="icon-search-result-name">{item.name || item.reference || 'Icon'}</span>
                          <span class="icon-search-result-sub">{[item.reference, item.category, item.license].filter(Boolean).join(' · ')}</span>
                        </span>
                        {#if selectedIconKey === iconResultKey(item)}
                          <span class="icon-search-result-selection" aria-hidden="true">✓ Selected</span>
                        {:else if importingIconKey === iconResultKey(item)}
                          <span class="icon-search-result-selection" aria-hidden="true">Selecting…</span>
                        {/if}
                      </button>
                    {/each}
                  </div>
                {/if}
              </section>
            {/each}
          </div>
        </div>
      </div>
    </section>

    <footer class="modal-card-foot entry-modal-footer">
      <div class="entry-modal-footer-top">
        <button id="entryDeleteBtn" class="button is-danger is-light entry-modal-delete-btn" type="button" disabled={isNew} on:click={handleDelete}>Delete</button>
        <button id="entryCancelBtn" class="button" type="button" on:click={handleClose}>Cancel</button>
      </div>
      <div class="entry-modal-footer-bottom">
        <button id="entrySaveBtn" class="button is-link" type="button" on:click={handleSave}>{isNew ? 'Add Button' : 'Save'}</button>
      </div>
    </footer>
  </div>
</div>
