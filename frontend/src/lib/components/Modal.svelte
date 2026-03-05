<script lang="ts">
  import { createEventDispatcher } from "svelte";

  export let open: boolean = false;
  export let title: string = "";
  export let maxWidth: string = "max-w-md";

  const dispatch = createEventDispatcher();

  function close() {
    open = false;
    dispatch("close");
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Escape" && open) {
      close();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
  <div
    class="fixed inset-0 bg-slate-950/80 backdrop-blur-sm z-[100] flex items-center justify-center p-4 fade-in"
    on:click={close}
  >
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <div
      class="bg-slate-900/95 backdrop-blur-xl border border-cyan-500/30 rounded-2xl shadow-[0_0_30px_rgba(0,0,0,0.8),_0_0_15px_rgba(6,182,212,0.1)] w-full {maxWidth} overflow-hidden"
      on:click|stopPropagation
    >
      <div
        class="p-6 border-b border-slate-800 flex justify-between items-center bg-slate-950/50"
      >
        <h3 class="text-xl font-bold text-cyan-50 font-mono tracking-wide">
          {title}
        </h3>
        <button
          on:click={close}
          class="text-slate-500 hover:text-cyan-400 transition-colors bg-slate-900 hover:bg-slate-800 border border-slate-700 hover:border-cyan-500/50 rounded-lg p-1"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><line x1="18" y1="6" x2="6" y2="18"></line><line
              x1="6"
              y1="6"
              x2="18"
              y2="18"
            ></line></svg
          >
        </button>
      </div>

      <div class="p-6 max-h-[80vh] overflow-y-auto custom-scrollbar">
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
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }
</style>
