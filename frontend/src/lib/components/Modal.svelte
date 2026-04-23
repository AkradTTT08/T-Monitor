<script lang="ts">
  import { createEventDispatcher } from "svelte";

  let { 
    open = $bindable(false), 
    title = "", 
    size = "md", // sm, md, lg, xl, full
    overflowVisible = false,
    children
  } = $props();

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

  const sizeClasses: Record<string, string> = {
    sm: "max-w-md h-auto",
    md: "max-w-2xl h-auto",
    lg: "max-w-4xl h-auto",
    xl: "max-w-6xl h-auto",
    "2xl": "max-w-7xl h-auto",
    full: "max-w-[95vw] w-full h-[95vh]"
  };
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- Final Centering Strategy: Relative modal in a flex container with overflow-y-auto backdrop -->
  <div
    class="fixed inset-0 z-[100] flex items-center justify-center p-2 md:p-8 overflow-y-auto modal-backdrop-fade pointer-events-none"
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div 
      class="fixed inset-0 bg-slate-950/80 backdrop-blur-md pointer-events-auto"
      onclick={close}
      aria-hidden="true"
    ></div>

    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="relative bg-slate-900 border border-slate-700/60 w-full {sizeClasses[size] || sizeClasses.md} rounded-3xl shadow-[0_20px_50px_rgba(0,0,0,0.5)] flex flex-col pointer-events-auto overflow-hidden transition-all duration-300"
      onclick={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      tabindex="-1"
    >
      <div
        class="px-6 py-4 border-b border-slate-800 flex justify-between items-center bg-slate-900/50"
      >
        <h3 id="modal-title" class="text-lg font-bold text-cyan-50 font-mono tracking-tight uppercase">
          {title}
        </h3>
        <button
          onclick={close}
          class="text-slate-500 hover:text-cyan-400 p-2 rounded-xl hover:bg-slate-800 transition-all font-bold"
          aria-label="Close modal"
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

      <div class="p-6 flex-1 {overflowVisible ? '' : 'overflow-y-auto custom-scrollbar'}">
      {#if children}
        {@render children()}
      {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop-fade {
    animation: modalFadeIn 0.2s ease-out forwards;
  }
  @keyframes modalFadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }
</style>
