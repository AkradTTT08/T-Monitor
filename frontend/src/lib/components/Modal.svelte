<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let open: boolean = false;
  export let title: string = '';
  export let maxWidth: string = 'max-w-md';

  const dispatch = createEventDispatcher();

  function close() {
    open = false;
    dispatch('close');
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && open) {
      close();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
<div class="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-[100] flex items-center justify-center p-4 fade-in" on:click={close}>
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="bg-white rounded-2xl shadow-xl w-full {maxWidth} overflow-hidden" on:click|stopPropagation>
    <div class="p-6 border-b border-slate-100 flex justify-between items-center">
      <h3 class="text-xl font-bold text-slate-900">{title}</h3>
      <button on:click={close} class="text-slate-400 hover:text-slate-600 transition-colors bg-slate-50 hover:bg-slate-100 rounded-lg p-1">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
      </button>
    </div>
    
    <div class="p-6">
      <slot></slot>
    </div>
  </div>
</div>
{/if}

<style>
  .fade-in {
    animation: fadeIn 0.2s ease-out;
  }
  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
