<script>
  import { COLOR_GROUPS, THEME_DEFAULTS } from '../lib/theme.js';
  import { createEventDispatcher } from 'svelte';

  export let theme = {};
  export let builtinPresets = [];
  export let savedPresets = [];
  export let presetName = '';

  const dispatch = createEventDispatcher();

  let selectedPresetId = '';
  $: selectedPreset = [...builtinPresets, ...savedPresets].find(p => p.id === selectedPresetId) || null;
  $: isCustomSelected = selectedPreset != null && savedPresets.some(p => p.id === selectedPresetId);

  function onPresetChange(e) {
    selectedPresetId = e.target.value;
    if (!selectedPresetId) return;
    const preset = [...builtinPresets, ...savedPresets].find(p => p.id === selectedPresetId);
    if (preset) dispatch('loadPreset', { preset });
  }

  function deleteSelectedPreset() {
    if (!selectedPreset) return;
    dispatch('deletePreset', { preset: selectedPreset });
    selectedPresetId = '';
  }

  function updateColor(key, value) {
    dispatch('update', { key, value });
  }

  function savePreset() {
    dispatch('savePreset', { name: presetName });
  }

  /** Returns #ffffff or #000000 depending on perceived brightness of the hex color. */
  function contrastBorder(hex) {
    hex = (hex || '#000000').replace('#', '');
    if (hex.length === 3) hex = hex.split('').map(c => c + c).join('');
    const r = parseInt(hex.slice(0, 2), 16);
    const g = parseInt(hex.slice(2, 4), 16);
    const b = parseInt(hex.slice(4, 6), 16);
    return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.5 ? '#000000' : '#ffffff';
  }
</script>

<!-- Presets: full width at top -->
<div class="te-presets">
  <span class="te-group-label">Presets</span>
  <div class="preset-select-row">
    <div class="select is-fullwidth" style="flex:1;min-width:0">
      <select value={selectedPresetId} on:change={onPresetChange}>
        <option value="">— Select a theme preset —</option>
        {#if builtinPresets.length}
          <optgroup label="Default themes">
            {#each builtinPresets as p}
              <option value={p.id}>{p.name}</option>
            {/each}
          </optgroup>
        {/if}
        {#if savedPresets.length}
          <optgroup label="Custom themes">
            {#each savedPresets as p}
              <option value={p.id}>{p.name}</option>
            {/each}
          </optgroup>
        {/if}
      </select>
    </div>
    {#if isCustomSelected}
      <button class="button is-danger is-small" type="button" on:click={deleteSelectedPreset}>Delete</button>
    {/if}
  </div>
  <div class="theme-save-row">
    <input class="input" bind:value={presetName} placeholder="New preset name…" />
    <button class="button is-link is-small" type="button" on:click={savePreset} disabled={!presetName}>Save preset</button>
  </div>
</div>

<!-- Color groups: 2-column grid when space allows -->
<div class="theme-color-grid">
  {#each COLOR_GROUPS as group}
    <div class="te-group">
      <span class="te-group-label">{group.label}</span>
      {#each group.fields as field}
        {@const val = theme[field.key] || THEME_DEFAULTS[field.key]}
        <div class="theme-field">
          <label class="theme-field-label" for={`tf-${field.key}`}>{field.label}</label>
          <div class="theme-field-inputs">
            <input
              type="color"
              class="color-swatch-input"
              style="border: 2px solid {contrastBorder(val)};"
              aria-label={field.label}
              value={val}
              on:input={(e) => updateColor(field.key, e.target.value)}
              on:change={(e) => updateColor(field.key, e.target.value)}
            />
            <input
              id={`tf-${field.key}`}
              class="input color-hex-input"
              type="text"
              value={val}
              on:change={(e) => updateColor(field.key, e.target.value)}
              maxlength="7"
              pattern="#[0-9a-fA-F]{6}"
            />
          </div>
        </div>
      {/each}
    </div>
  {/each}
</div>

<!-- Button color mode -->
<div class="te-group">
  <span class="te-group-label">Button Color Mode</span>
  <div class="select is-fullwidth">
    <select value={theme.buttonColorMode} on:change={(e) => updateColor('buttonColorMode', e.target.value)}>
      <option value="cycle-custom">Cycle (HSL)</option>
      <option value="solid-all">Solid — same for all</option>
      <option value="solid-per-group">Solid — per group</option>
    </select>
  </div>

  {#if theme.buttonColorMode === 'cycle-custom' || !theme.buttonColorMode}
    <div class="theme-sliders-outer">
      <div class="theme-slider-box">
        <div class="slider-row">
          <label for="theme-hue-step">Hue step</label>
          <input id="theme-hue-step" type="range" min="5" max="60"
            value={theme.buttonCycleHueStep ?? 15}
            on:input={(e) => updateColor('buttonCycleHueStep', Number(e.target.value))} />
          <span class="slider-val">{theme.buttonCycleHueStep ?? 15}°</span>
        </div>
        <div class="slider-row">
          <label for="theme-saturation">Saturation</label>
          <input id="theme-saturation" type="range" min="20" max="100"
            value={theme.buttonCycleSaturation ?? 70}
            on:input={(e) => updateColor('buttonCycleSaturation', Number(e.target.value))} />
          <span class="slider-val">{theme.buttonCycleSaturation ?? 70}%</span>
        </div>
        <div class="slider-row">
          <label for="theme-lightness">Lightness</label>
          <input id="theme-lightness" type="range" min="20" max="95"
            value={theme.buttonCycleLightness ?? 74}
            on:input={(e) => updateColor('buttonCycleLightness', Number(e.target.value))} />
          <span class="slider-val">{theme.buttonCycleLightness ?? 74}%</span>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
/* ── Preset section ──────────────────────────────────────────── */
.te-presets {
  margin-bottom: 1rem;
}

.preset-select-row {
  display: flex;
  gap: 0.45rem;
  align-items: center;
  margin-bottom: 0.5rem;
}

.theme-save-row {
  display: flex;
  gap: 0.45rem;
  align-items: center;
  flex-wrap: wrap;
}

.theme-save-row .input {
  flex: 1 1 10rem;
  min-width: 0;
}

/* ── Group label (replaces section box header) ───────────────── */
.te-group-label {
  display: block;
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--admin-text, #f8fafc);
  margin-bottom: 0.45rem;
}

/* ── 2-column grid for color groups ─────────────────────────── */
.theme-color-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 0.35rem 1rem;
  margin-bottom: 0.75rem;
}

/* ── Group: flat, no box ─────────────────────────────────────── */
.te-group {
  margin-bottom: 0.75rem;
}

/* ── Individual color field: label on top, inputs below ──────── */
.theme-field {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  margin-bottom: 0.4rem;
}

.theme-field:last-child {
  margin-bottom: 0;
}

.theme-field-label {
  font-size: 0.88rem;
  color: var(--admin-text, #f8fafc);
}

.theme-field-inputs {
  display: flex;
  align-items: stretch;
  gap: 0.35rem;
}

/* Color swatch with dynamic contrast border */
.color-swatch-input {
  width: 2.5rem;
  height: 2.5rem;
  min-width: 2.5rem;
  flex-shrink: 0;
  padding: 2px;
  /* border set via inline style for contrast */
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  -webkit-appearance: none;
  box-sizing: border-box;
}

.color-swatch-input::-webkit-color-swatch-wrapper {
  padding: 0;
}

.color-swatch-input::-webkit-color-swatch {
  border: none;
  border-radius: 2px;
}

/* Hex input: fills remaining space */
.color-hex-input {
  flex: 1;
  min-width: 0;
  font-family: monospace;
}

/* ── HSL sliders box ─────────────────────────────────────────── */
.theme-sliders-outer {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.6rem;
}

.theme-slider-box {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  min-width: 15rem;
  width: 100%;
}

.slider-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.slider-row label {
  min-width: 5.5rem;
  font-size: 0.88rem;
  color: var(--admin-text, #f8fafc);
  text-align: left;
}

.slider-row input[type="range"] {
  flex: 1;
  accent-color: var(--admin-accent, #2563eb);
}

.slider-val {
  font-size: 0.88rem;
  color: var(--admin-label-text, #94a3b8);
  min-width: 2.5rem;
  text-align: right;
}
</style>
