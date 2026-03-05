<script lang="ts">
  import { createEventDispatcher } from "svelte";

  export let value = "";
  export let placeholder = "";
  export let disabled = false;
  export let required = false;
  export let rows: number | undefined = undefined;
  export let variables: Record<string, string> = {}; // Available environment variables

  // Custom styling properties
  export let outerClass =
    "bg-slate-900/50 rounded-lg border border-slate-700/50 focus-within:ring-2 focus-within:ring-cyan-500/50 h-auto";
  export let innerClass = "px-4 py-3 resize-y block";
  export let textClass = "text-cyan-50";

  const dispatch = createEventDispatcher();

  let textareaEl: HTMLTextAreaElement;
  let scrollY = 0;
  let scrollX = 0;

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
        parts.push({ type: "text", text: str.slice(lastIndex, match.index) });
      }
      const varName = match[1];
      const hasVar = variables.hasOwnProperty(varName);

      parts.push({
        type: "variable",
        text: match[0],
        name: varName,
        value: hasVar ? variables[varName] : null,
        isValid: hasVar,
      });
      lastIndex = regex.lastIndex;
    }

    if (lastIndex < str.length) {
      parts.push({ type: "text", text: str.slice(lastIndex) });
    }
    return parts;
  }

  function handleScroll(e: Event) {
    scrollY = (e.target as HTMLTextAreaElement).scrollTop;
    scrollX = (e.target as HTMLTextAreaElement).scrollLeft;
  }
</script>

<div
  class={`relative w-full group font-mono text-xs transition-all overflow-hidden flex flex-col ${outerClass}`}
>
  <!-- Visual Overlay -->
  <div
    class="absolute inset-0 pointer-events-none whitespace-pre-wrap break-words z-10"
    aria-hidden="true"
  >
    <div
      class={`w-full h-full ${innerClass}`}
      style="transform: translate({-scrollX}px, {-scrollY}px); overflow: visible; display: block; resize: none; border: none; outline: none; background: transparent;"
    >
      {#if !value && placeholder}
        <span class="text-slate-400">{placeholder}</span>
      {/if}
      {#each segments as seg}
        {#if seg.type === "variable"}
          <!-- Display variable highlighting -->
          <span
            class="group/var relative inline transition-colors {!disabled
              ? seg.isValid
                ? 'text-emerald-500'
                : 'text-rose-500'
              : 'opacity-50'}"
          >
            <span class="text-transparent">{seg.text}</span>
            <span class="absolute inset-0 pointer-events-none">{seg.text}</span>
          </span>
        {:else}
          <!-- Regular text behind textarea -->
          <span class={textClass}>{seg.text}</span>
        {/if}
      {/each}
      <!-- Add empty line break to ensure scroll matches trailing newlines -->
      {#if value.endsWith("\n")}
        <br />
      {/if}
    </div>
  </div>

  <!-- Actual Input Data -->
  <textarea
    bind:this={textareaEl}
    bind:value
    {disabled}
    {required}
    {rows}
    {placeholder}
    on:scroll={handleScroll}
    class={`relative flex-1 w-full bg-transparent focus:outline-none caret-cyan-500 z-20 ${disabled ? "cursor-not-allowed opacity-50" : ""} ${innerClass}`}
    style="color: transparent;"
    spellcheck="false"
  ></textarea>
</div>
