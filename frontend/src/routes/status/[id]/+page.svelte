<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { page } from "$app/stores";
  import { API_BASE_URL } from "$lib/config";

  let loading = true;
  let error = "";
  let data: any = null;
  let refreshInterval: any;

  const projectId = $page.params.id;

  async function fetchStatus() {
    try {
      const res = await fetch(`${API_BASE_URL}/api/v1/public/status/${projectId}`);
      if (!res.ok) {
        throw new Error("Failed to load status page");
      }
      data = await res.json();
    } catch (err: any) {
      error = err.message;
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchStatus();
    refreshInterval = setInterval(fetchStatus, 60000); // Refresh every minute
  });

  onDestroy(() => {
    if (refreshInterval) clearInterval(refreshInterval);
  });

  function getStatusColor(status: string) {
    if (status === "UP") return "text-emerald-400";
    if (status === "DOWN") return "text-rose-400";
    return "text-slate-400";
  }

  function getStatusBg(status: string) {
    if (status === "UP") return "bg-emerald-500/10 border-emerald-500/20";
    if (status === "DOWN") return "bg-rose-500/10 border-rose-500/20";
    return "bg-slate-500/10 border-slate-500/20";
  }

  $: allSystemsOperational = data?.apis?.every((a: any) => a.status === "UP");
  $: partialOutage = data?.apis?.some((a: any) => a.status === "DOWN") && data?.apis?.some((a: any) => a.status === "UP");
  $: majorOutage = data?.apis?.length > 0 && data?.apis?.every((a: any) => a.status === "DOWN");
</script>

<div class="z-20 w-full max-w-4xl px-6 py-12 flex flex-col items-center">
  {#if loading && !data}
    <div class="flex flex-col items-center gap-4 animate-pulse">
      <div class="w-16 h-16 rounded-full bg-slate-800 border-2 border-cyan-500/30"></div>
      <p class="text-cyan-500 font-mono text-sm tracking-widest">ESTABLISHING CONNECTION...</p>
    </div>
  {:else if error}
    <div class="bg-rose-950/20 border border-rose-500/30 p-8 rounded-2xl text-center backdrop-blur-xl">
      <h2 class="text-rose-400 font-bold text-xl mb-2">ACCESS DENIED / NOT FOUND</h2>
      <p class="text-rose-300/60 text-sm">The status page for this project could not be reached or does not exist.</p>
    </div>
  {:else if data}
    <!-- Status Header Card -->
    <div class="w-full mb-12">
        <div class="flex items-center justify-between mb-8">
            <div class="flex items-center gap-4">
                {#if data.company?.logo_url}
                    <img 
                        src={data.company.logo_url.startsWith("http") ? data.company.logo_url : `${API_BASE_URL}${data.company.logo_url}`} 
                        alt={data.company.name} 
                        class="w-12 h-12 object-contain" 
                    />
                {:else}
                    <div class="w-12 h-12 rounded-xl bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center">
                        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-cyan-400"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>
                    </div>
                {/if}
                <div>
                    <h1 class="text-2xl font-black text-slate-100 tracking-tight">{data.project_name}</h1>
                    <p class="text-slate-400 text-sm">{data.description || "System Status Monitor"}</p>
                </div>
            </div>
            <div class="text-right">
                <p class="text-[10px] text-slate-500 font-bold uppercase tracking-widest">Last Updated</p>
                <p class="text-xs text-slate-400 font-mono">{new Date().toLocaleTimeString()}</p>
            </div>
        </div>

        <!-- Major Status Badge -->
        <div class="w-full p-8 rounded-3xl border text-center transition-all duration-700 backdrop-blur-2xl
            {allSystemsOperational ? 'bg-emerald-500/10 border-emerald-500/30 shadow-[0_0_50px_rgba(16,185,129,0.1)]' : 
             majorOutage ? 'bg-rose-500/10 border-rose-500/30 shadow-[0_0_50px_rgba(244,63,94,0.1)]' : 
             'bg-amber-500/10 border-amber-500/30 shadow-[0_0_50px_rgba(245,158,11,0.1)]'}">
            <span class="text-4xl mb-4 block">
                {allSystemsOperational ? '✨' : partialOutage ? '〽️' : '🚨'}
            </span>
            <h2 class="text-3xl font-black tracking-tight mb-2
                {allSystemsOperational ? 'text-emerald-400' : partialOutage ? 'text-amber-400' : 'text-rose-400'}">
                {allSystemsOperational ? 'All Systems Operational' : partialOutage ? 'Partial System Outage' : 'Major System Outage'}
            </h2>
            <p class="text-slate-400 font-medium">
                {allSystemsOperational ? 'Everything is running smoothly. No issues detected in the last 24 hours.' : 
                 'We are investigating issues affecting some of our internal services.'}
            </p>
        </div>
    </div>

    <!-- Components List -->
    <div class="w-full space-y-4 mb-12">
        <h3 class="text-xs font-bold text-slate-500 uppercase tracking-widest ml-4 mb-2">Systems & Services</h3>
        <div class="bg-slate-800/20 backdrop-blur-xl border border-slate-700/30 rounded-3xl overflow-hidden">
            {#each data.apis as api}
                <div class="flex items-center justify-between p-5 border-b border-slate-700/20 last:border-0 hover:bg-slate-700/10 transition-colors">
                    <div class="flex flex-col gap-0.5">
                        <span class="text-slate-100 font-bold">{api.name}</span>
                        <span class="text-[10px] text-slate-500 font-mono uppercase tracking-wider">{api.folder}</span>
                    </div>
                    <div class="flex items-center gap-6">
                        <div class="hidden md:flex flex-col items-end">
                            <span class="text-[10px] text-slate-500 font-bold uppercase tracking-widest">7D Uptime</span>
                            <span class="text-xs font-mono {api.uptime_percent >= 99 ? 'text-emerald-400' : 'text-amber-400'}">
                                {api.uptime_percent}%
                            </span>
                        </div>
                        <div class="flex items-center gap-2 px-3 py-1 rounded-full border {getStatusBg(api.status)}">
                            <div class="w-2 h-2 rounded-full {getStatusColor(api.status).replace('text-', 'bg-')} animate-pulse"></div>
                            <span class="text-[10px] font-black tracking-widest uppercase {getStatusColor(api.status)}">{api.status === 'UP' ? 'Operational' : 'Outage'}</span>
                        </div>
                    </div>
                </div>
            {/each}
        </div>
    </div>

    <!-- Historical Uptime Graph (Last 7 Days) -->
    <div class="w-full">
        <h3 class="text-xs font-bold text-slate-500 uppercase tracking-widest ml-4 mb-4">Historical Performance (Last 7 Days)</h3>
        <div class="grid grid-cols-7 gap-2 h-24">
            {#each data.history as day}
                <div class="group relative flex flex-col justify-end gap-2">
                    <div class="w-full rounded-lg transition-all duration-500 cursor-default
                        {day.uptime_percent >= 99 ? 'bg-emerald-500/40 hover:bg-emerald-500/60' : 
                         day.uptime_percent >= 95 ? 'bg-amber-500/40 hover:bg-amber-500/60' : 
                         'bg-rose-500/40 hover:bg-rose-500/60'}"
                        style="height: {day.uptime_percent}%">
                    </div>
                    <span class="text-[9px] text-center text-slate-600 font-mono uppercase tracking-tighter">
                        {new Date(day.day).toLocaleDateString('en-US', {weekday: 'short'})}
                    </span>

                    <!-- Tooltip -->
                    <div class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2 py-1 bg-slate-800 border border-slate-700 rounded text-[10px] text-slate-100 whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-30 shadow-2xl">
                        {day.day}: <span class="font-bold">{day.uptime_percent}%</span>
                    </div>
                </div>
            {/each}
        </div>
        <p class="text-center text-[10px] text-slate-600 mt-8 font-mono tracking-widest uppercase">
            &copy; {new Date().getFullYear()} T-MONITOR OBSERVABILITY ENGINE
        </p>
    </div>
  {/if}
</div>

<style>
    @keyframes pulse {
        0%, 100% { opacity: 1; transform: scale(1); }
        50% { opacity: 0.8; transform: scale(0.95); }
    }
</style>
