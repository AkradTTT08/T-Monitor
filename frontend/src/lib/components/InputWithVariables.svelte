<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let value = '';
  export let placeholder = '';
  export let disabled = false;
  export let required = false;
  export let variables: Record<string, string> = {}; // Available environment variables

  const dispatch = createEventDispatcher();

  let inputEl: HTMLInputElement;
  let isFocused = false;

  // Split string into text and variable segments
  $: segments = parseSegments(value);

  function parseSegments(str: string) {
    if (!str) return [];
    const parts = [];
    const regex = /\{\{([^}]+)\}\}/g;
    let lastIndex = 0;
    let match;

    while ((match = regex.exec(str)) !== null) {
      if (match.index > lastIndex) {
        parts.push({ type: 'text', text: str.slice(lastIndex, match.index) });
      }
      const varName = match[1];
      const hasVar = variables.hasOwnProperty(varName);

      parts.push({ 
        type: 'variable', 
        text: match[0], 
        name: varName,
        value: hasVar ? variables[varName] : null,
        isValid: hasVar
      });
      lastIndex = regex.lastIndex;
    }

    if (lastIndex < str.length) {
      parts.push({ type: 'text', text: str.slice(lastIndex) });
    }
    return parts;
  }
</script>

<div class="relative w-full h-full flex items-center bg-transparent group font-mono text-sm">
  <!-- Visual Overlay -->
  <div class="absolute inset-0 px-4 py-2 pointer-events-none whitespace-pre overflow-hidden text-transparent flex items-center" aria-hidden="true" style="z-index: 10;">
    {#if !value && placeholder}
      <span class="text-slate-400">{placeholder}</span>
    {/if}
    <div class="flex-1 w-full truncate">
      {#each segments as seg}
        {#if seg.type === 'variable'}
          <span class="group/var relative inline-flex items-center transition-colors {!disabled ? (seg.isValid ? 'text-emerald-500' : 'text-rose-500') : 'opacity-50'}">
            {seg.text}
            <!-- Tooltip -->
            {#if seg.isValid && !disabled}
              <div class="opacity-0 group-hover/var:opacity-100 transition-opacity absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2 py-1 bg-slate-800 text-white text-xs rounded break-words max-w-xs shadow-lg whitespace-normal z-50 pointer-events-none w-max font-sans tracking-normal font-normal">
              <span class="font-bold text-emerald-400">{seg.name}</span>: {seg.value}
              <!-- Triangle -->
              <div class="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-slate-800"></div>
            </div>
          {/if}
        </span>
          {:else}
            <!-- Keep text color transparent, only background/spans are visible for variables -->
            <span class="text-transparent">{seg.text}</span>
          {/if}
        {/each}
      </div>
  </div>

  <!-- Actual Input Data -->
  <!-- Set text color to essentially transparent if it overlaps perfectly, but we need the caret to show!
       Therefore, text is visible, but we overlay the colored variable spans exactly on top or behind. 
       Actually, standard approach: make the standard text colored normally, but the overlay provides the background boxes. -->
  <input 
    bind:this={inputEl}
    bind:value
    {disabled}
    {required}
    type="text"
    placeholder={placeholder}
    on:focus={() => isFocused = true}
    on:blur={() => isFocused = false}
    on:paste
    class="relative z-20 w-full h-full px-4 py-2 bg-transparent focus:outline-none transition-all text-slate-700 caret-blue-500 {disabled ? 'cursor-not-allowed opacity-50' : ''}" 
    style="color: transparent;" 
    spellcheck="false"
  />

  <!-- Visible Text Layer (sits exactly behind input, so the caret works, but text aligns perfectly) -->
  <div class="absolute inset-0 px-4 py-2 pointer-events-none whitespace-pre overflow-hidden flex items-center z-10" aria-hidden="true">
    <div class="flex-1 w-full truncate text-slate-800">
       {#each segments as seg}
         {#if seg.type === 'variable'}
            <!-- Invisible text here so the visual overlay layer takes over rendering the variable -->
            <span class="text-transparent">{seg.text}</span>
         {:else}
            <span>{seg.text}</span>
         {/if}
       {/each}
    </div>
  </div>
</div>
