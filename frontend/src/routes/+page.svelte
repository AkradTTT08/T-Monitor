<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';

  onMount(() => {
    const token = localStorage.getItem("monitor_token");
    if (token) {
      window.location.href = "/dashboard";
    } else {
      window.location.href = "/login";
    }
  });
</script>

<div class="min-h-screen bg-slate-950 flex items-center justify-center font-mono">
  <div class="text-cyan-500 animate-pulse">
    INITIALIZING_RESTRICTED_ACCESS...
  </div>
</div>
