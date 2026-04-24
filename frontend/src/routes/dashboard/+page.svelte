<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { fade, fly } from "svelte/transition";
  import { API_BASE_URL } from "$lib/config";

  let pulseData: any = null;
  let loading = true;
  let error = "";
  let refreshInterval: any;
  
  let selectedCompanyId = "";
  let selectedProjectId = "all";
  let projects: any[] = [];

  async function fetchProjects() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.ok) {
        const allProjects = await res.json();
        // filter by company
        projects = allProjects.filter((p: any) => p.company_id?.toString() === selectedCompanyId);
      }
    } catch (err) {}
  }

  async function fetchGlobalPulse() {
    try {
      error = ""; // Clear previous errors on retry
      const token = localStorage.getItem("monitor_token");
      if (!token) return;

      let url = `${API_BASE_URL}/api/v1/analytics/pulse`;
      const queryParams = new URLSearchParams();
      if (selectedCompanyId) {
         queryParams.append("company_id", selectedCompanyId);
      }
      if (selectedProjectId && selectedProjectId !== "all") {
         queryParams.append("project_id", selectedProjectId);
      }
      
      const q = queryParams.toString();
      if (q) {
         url += `?${q}`;
      }

      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` }
      });

      if (!res.ok) throw new Error("Failed to load pulse data");
      pulseData = await res.json();
    } catch (err: any) {
      error = err.message;
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    selectedCompanyId = localStorage.getItem("monitor_selected_company") || "";
    if (selectedCompanyId) {
      fetchProjects();
    }
    fetchGlobalPulse();
    // Poll every 5 seconds for real-time vibe
    refreshInterval = setInterval(fetchGlobalPulse, 5000);
    window.addEventListener('storage', handleStorageChange);
  });

  onDestroy(() => {
    if (refreshInterval) clearInterval(refreshInterval);
    window.removeEventListener('storage', handleStorageChange);
  });

  function handleStorageChange(e: StorageEvent) {
    if (e.key === 'monitor_selected_company') {
      selectedCompanyId = e.newValue || "";
      selectedProjectId = "all";
      if (selectedCompanyId) fetchProjects();
      fetchGlobalPulse();
    }
  }

  function handleFilterChange() {
    fetchGlobalPulse();
  }

  function getStatusColor(isSuccess: boolean) {
    return isSuccess ? "bg-emerald-500 shadow-[0_0_15px_rgba(16,185,129,0.8)]" : "bg-rose-500 shadow-[0_0_15px_rgba(244,63,94,0.8)]";
  }
</script>

<svelte:head>
  <title>Pulse Dashboard | T-Monitor</title>
</svelte:head>

<div class="h-full w-full overflow-y-auto p-2 pb-12 fade-in custom-scrollbar">
  <!-- Interactive Header -->
  <div class="flex flex-col lg:flex-row lg:justify-between lg:items-start mb-10 gap-6 relative z-10 px-2 lg:px-8 mt-4">
    <div class="flex flex-col gap-2">
      <h1 class="text-4xl font-black text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 via-blue-500 to-indigo-500 tracking-tighter uppercase font-mono">
        Global Pulse
      </h1>
      <p class="text-cyan-500/60 text-sm font-mono tracking-widest uppercase">
        Live aggregate telemetry across all accessible workspaces
      </p>
      {#if selectedCompanyId}
        <div class="mt-4 flex items-center gap-3">
          <label for="project-filter" class="text-cyan-400 font-mono text-xs tracking-widest uppercase">Filter by Project:</label>
          <select 
            id="project-filter"
            bind:value={selectedProjectId} 
            on:change={handleFilterChange}
            class="bg-slate-900 border border-slate-700 text-cyan-50 text-sm rounded-lg focus:ring-cyan-500 focus:border-cyan-500 block p-2 font-mono shadow-[0_0_10px_rgba(6,182,212,0.1)] outline-none"
          >
            <option value="all">All Projects</option>
            {#each projects as p}
              <option value={p.id}>{p.name}</option>
            {/each}
          </select>
        </div>
      {/if}
    </div>

    <!-- Info/Help Panel -->
    <div class="bg-cyan-950/30 border border-cyan-500/20 rounded-2xl p-4 lg:w-[450px] shadow-lg backdrop-blur-sm relative overflow-hidden group">
      <div class="absolute inset-0 bg-gradient-to-r from-cyan-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
      <div class="flex items-start gap-4 relative z-10">
        <div class="p-2 bg-cyan-500/10 rounded-lg text-cyan-400 shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>
        </div>
        <div class="text-xs text-slate-400 font-mono leading-relaxed">
          <span class="text-cyan-400 font-bold mb-1 block uppercase tracking-wider">How to use this dashboard</span>
          This is your central command center. It automatically aggregates health data from <span class="text-slate-200">all API endpoints across projects you have access to</span>. Watch the <span class="text-slate-200">Live Heartbeat</span> stream to catch incoming requests in real-time, or use the radar to quickly spot failing endpoints (red dots).
        </div>
      </div>
    </div>
  </div>

  {#if loading && !pulseData}
    <div class="flex justify-center items-center h-64">
      <svg class="animate-spin h-10 w-10 text-cyan-500 shadow-[0_0_20px_rgba(6,182,212,0.5)] rounded-full" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
    </div>
  {:else if error}
    <div class="w-full px-2 lg:px-8">
      <div class="p-8 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-rose-400 font-mono text-center backdrop-blur-xl shadow-2xl shadow-rose-900/20">
        Error loading pulse: {error}
      </div>
    </div>
  {:else if pulseData}
    <!-- Glassmorphic KPI Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8 relative z-10 px-2 lg:px-8">
      
      <!-- Global Uptime Card -->
      <div class="bg-slate-900/60 backdrop-blur-3xl rounded-3xl border border-slate-700/50 p-6 shadow-2xl relative overflow-hidden group hover:border-emerald-500/30 transition-colors duration-500">
        <div class="absolute inset-0 bg-emerald-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-700"></div>
        <div class="absolute -right-10 -top-10 w-40 h-40 bg-emerald-500/20 rounded-full blur-3xl opacity-50 group-hover:opacity-80 transition-opacity duration-700"></div>
        
        <h3 class="text-[10px] text-slate-400 font-bold uppercase tracking-widest mb-1 relative z-10">Global Uptime (24H)</h3>
        <div class="flex items-baseline gap-2 relative z-10 mt-2">
          <span class="text-5xl font-black {pulseData.global_uptime >= 99 ? 'text-emerald-400' : pulseData.global_uptime >= 95 ? 'text-amber-400' : 'text-rose-400'} font-mono tracking-tight drop-shadow-[0_0_8px_currentColor]">
            {pulseData.global_uptime}%
          </span>
        </div>
      </div>

      <!-- Average Latency Card -->
      <div class="bg-slate-900/60 backdrop-blur-3xl rounded-3xl border border-slate-700/50 p-6 shadow-2xl relative overflow-hidden group hover:border-cyan-500/30 transition-colors duration-500">
        <div class="absolute inset-0 bg-cyan-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-700"></div>
        <div class="absolute -right-10 -top-10 w-40 h-40 bg-cyan-500/20 rounded-full blur-3xl opacity-50 group-hover:opacity-80 transition-opacity duration-700"></div>

        <h3 class="text-[10px] text-slate-400 font-bold uppercase tracking-widest mb-1 relative z-10">Avg Network Latency</h3>
        <div class="flex items-baseline gap-2 relative z-10 mt-2">
          <span class="text-5xl font-black text-cyan-400 font-mono tracking-tight drop-shadow-[0_0_8px_rgba(6,182,212,0.8)]">
            {pulseData.avg_latency}
          </span>
          <span class="text-cyan-500/60 font-bold text-sm">ms</span>
        </div>
      </div>

      <!-- Active APIs Card -->
      <div class="bg-slate-900/60 backdrop-blur-3xl rounded-3xl border border-slate-700/50 p-6 shadow-2xl relative overflow-hidden group hover:border-indigo-500/30 transition-colors duration-500">
        <div class="absolute inset-0 bg-indigo-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-700"></div>
        <div class="absolute -right-10 -top-10 w-40 h-40 bg-indigo-500/20 rounded-full blur-3xl opacity-50 group-hover:opacity-80 transition-opacity duration-700"></div>

        <h3 class="text-[10px] text-slate-400 font-bold uppercase tracking-widest mb-1 relative z-10">Active Monitored APIs</h3>
        <div class="flex items-baseline gap-2 relative z-10 mt-2">
          <span class="text-5xl font-black text-indigo-400 font-mono tracking-tight drop-shadow-[0_0_8px_rgba(99,102,241,0.8)]">
            {pulseData.active_apis}
          </span>
          <span class="text-indigo-500/60 font-bold text-xs uppercase ml-1">Endpoints</span>
        </div>
      </div>
    </div>

    <!-- Live Telemetry Main Section -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 relative z-10 px-2 lg:px-8">
      
      <!-- Live Matrix (Left Col) -->
      <div class="lg:col-span-2 bg-slate-900/60 backdrop-blur-3xl rounded-3xl border border-slate-700/50 p-6 shadow-[0_8px_30px_rgb(0,0,0,0.5)] overflow-hidden relative">
        <div class="absolute top-0 right-0 w-80 h-80 bg-blue-500/10 rounded-full blur-[80px]"></div>
        
        <div class="flex justify-between items-center mb-6 relative z-10">
          <h2 class="text-sm font-bold text-slate-200 uppercase tracking-widest flex items-center gap-3">
            <div class="w-2.5 h-2.5 rounded-full bg-rose-500/80 animate-pulse shadow-[0_0_15px_rgba(239,68,68,1)] ring-2 ring-rose-500/30"></div>
            Live Heartbeat Stream
          </h2>
          <span class="text-[10px] bg-slate-800/80 border border-slate-600/50 text-cyan-400 px-3 py-1.5 rounded-lg shadow-inner uppercase font-mono flex items-center gap-1.5 backdrop-blur-sm">
            <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="animate-spin-slow"><path d="M21.5 2v6h-6M2.13 15.57a10 10 0 1 0 3.84-10.58L2 7"/><line x1="12" y1="12" x2="12" y2="12"/></svg>
            POLLING 5s
          </span>
        </div>

        <div class="w-full h-[400px] overflow-y-auto pr-3 custom-scrollbar relative z-10">
          <div class="space-y-3 pb-8">
            {#each (pulseData.recent_pings ?? []) as ping}
              <div in:fly={{ y: -20, duration: 400 }} class="group flex items-center justify-between p-4 rounded-xl bg-slate-800/40 border border-slate-700/50 hover:bg-slate-700/50 hover:border-slate-500/50 hover:-translate-y-0.5 transition-all duration-300 backdrop-blur-md shadow-lg">
                
                <div class="flex items-center gap-4 min-w-0 flex-1">
                  <!-- Blinking Node Dot -->
                  <div class="w-2.5 h-2.5 rounded-full {getStatusColor(ping.is_success)} shrink-0 ring-4 {ping.is_success ? 'ring-emerald-500/10' : 'ring-rose-500/10'}"></div>
                  
                  <!-- Info text -->
                  <div class="flex flex-col min-w-0 pr-4">
                    <span class="text-slate-100 font-bold text-sm tracking-wide truncate group-hover:text-cyan-300 transition-colors">
                      {ping.api_name}
                    </span>
                    <span class="text-[10px] text-slate-400 font-mono uppercase truncate mt-1 flex items-center gap-1.5 opacity-80">
                      <span class="px-1.5 py-0.5 rounded bg-slate-900 border border-slate-700">{ping.method}</span> 
                      <span class="text-cyan-500/80">{ping.project_name}</span>
                    </span>
                  </div>
                </div>

                <div class="flex items-center gap-5 shrink-0">
                  <!-- Latency -->
                  <div class="flex flex-col items-end">
                    <span class="text-[9px] text-slate-500 font-bold uppercase tracking-widest mb-0.5">Latency</span>
                    <span class="text-sm font-black font-mono {ping.response_time > 1000 ? 'text-amber-400' : 'text-cyan-400'} drop-shadow-[0_0_5px_currentColor]">
                      {ping.response_time}ms
                    </span>
                  </div>
                  <!-- Status Code Box -->
                  <div class="w-12 h-9 rounded-lg flex items-center justify-center font-black font-mono text-xs shadow-inner
                    {ping.is_success ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 shadow-[inset_0_0_10px_rgba(16,185,129,0.1)]' : 'bg-rose-500/10 text-rose-400 border border-rose-500/30 shadow-[inset_0_0_10px_rgba(244,63,94,0.1)]'}">
                    {ping.status_code}
                  </div>
                </div>

              </div>
            {/each}

            {#if (pulseData.recent_pings ?? []).length === 0}
              <div class="h-full flex flex-col items-center justify-center pt-24 pb-12">
                <div class="w-16 h-16 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center mb-4">
                  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-500"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
                </div>
                <p class="text-slate-500 font-mono text-sm tracking-widest uppercase">No Telemetry Received</p>
              </div>
            {/if}
          </div>
        </div>
      </div>

      <!-- Network Nodes Visualizer (Right Col) -->
      <div class="bg-slate-900/60 backdrop-blur-3xl rounded-3xl border border-slate-700/50 p-6 flex flex-col items-center justify-center relative overflow-hidden shadow-[0_8px_30px_rgb(0,0,0,0.5)]">
        <h2 class="absolute top-6 left-6 text-sm font-bold text-slate-200 uppercase tracking-widest z-20">Pulse Radar</h2>
        
        <!-- Radar Circles & Scanning line -->
        <div class="absolute inset-0 flex items-center justify-center pointer-events-none z-0">
          <!-- Glass backing -->
          <div class="absolute inset-0 bg-blue-500/5 rounded-3xl blur-[100px]"></div>
          
          <div class="w-[85%] aspect-square rounded-full border border-cyan-500/20 absolute"></div>
          <div class="w-[60%] aspect-square rounded-full border border-cyan-500/30 absolute"></div>
          <div class="w-[35%] aspect-square rounded-full border border-cyan-500/40 absolute shadow-[0_0_20px_rgba(6,182,212,0.1)] shadow-inner"></div>
          <div class="w-[10%] aspect-square rounded-full bg-cyan-500/20 absolute shadow-[0_0_15px_rgba(6,182,212,0.5)]"></div>
          <div class="w-full h-px bg-cyan-500/20 absolute"></div>
          <div class="h-full w-px bg-cyan-500/20 absolute"></div>
          
          <!-- Scanning radar beam -->
          <div class="absolute inset-2 rounded-full animate-radar-scan hidden md:block" 
               style="background: conic-gradient(from 0deg at 50% 50%, rgba(6,182,212, 0) 0%, rgba(6,182,212, 0) 75%, rgba(6,182,212, 0.4) 100%); mix-blend-mode: screen;">
          </div>
        </div>

        <!-- Simulated live dots moving around based on recent pings -->
        <div class="w-[85%] aspect-square relative z-10 mt-8 group/radar">
          {#each (pulseData.recent_pings ?? []).slice(0, 18) as ping, i}
             <!-- Randomize position mathematically for aesthetics -->
             {@const topPos = 15 + Math.abs(Math.sin((i+1)*99))*70}
             {@const leftPos = 15 + Math.abs(Math.cos((i+1)*15))*70}
             {@const animDelay = (i * 0.5) % 3}
             <div class="absolute w-2.5 h-2.5 rounded-full {getStatusColor(ping.is_success)} cursor-crosshair group/dot transform transition-transform hover:scale-150 ring-2 {ping.is_success ? 'ring-emerald-500/30' : 'ring-rose-500/30'}"
                  style="top: {topPos}%; left: {leftPos}%; animation: pulse-node 4s ease-in-out infinite alternate {animDelay}s;">
               <div class="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 bg-slate-900 border border-cyan-500/50 px-3 py-2 rounded-lg text-xs font-mono whitespace-nowrap opacity-0 group-hover/dot:opacity-100 transition-opacity z-50 shadow-[0_0_20px_rgba(15,23,42,0.9)] backdrop-blur-sm pointer-events-none text-cyan-200">
                 <div class="font-bold mb-1">{ping.api_name}</div>
                 <div class="text-[10px] text-cyan-500/80">{ping.response_time}ms • {ping.status_code}</div>
               </div>
             </div>
          {/each}
          
          <!-- Center anchor dot -->
          <div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-4 h-4 rounded-full bg-cyan-500 shadow-[0_0_20px_rgba(6,182,212,1)] z-20"></div>
        </div>
        
        <div class="absolute bottom-6 w-full text-center z-20">
            <p class="text-[10px] text-cyan-500 font-mono tracking-[0.3em] uppercase bg-slate-950/50 inline-block px-3 py-1 rounded backdrop-blur-md border border-cyan-500/20">Sensor Network Linked</p>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .custom-scrollbar::-webkit-scrollbar {
      width: 6px;
  }
  .custom-scrollbar::-webkit-scrollbar-track {
      background: rgba(15, 23, 42, 0.4);
      border-radius: 10px;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb {
      background: rgba(6, 182, 212, 0.3);
      border-radius: 10px;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb:hover {
      background: rgba(6, 182, 212, 0.6);
  }

  .animate-spin-slow {
      animation: spin 3s linear infinite;
  }

  .animate-radar-scan {
      animation: radar-scan 4s linear infinite;
  }

  @keyframes pulse-node {
      0% { transform: translate(0px, 0px) scale(0.9); opacity: 0.7; }
      50% { opacity: 1; }
      100% { transform: translate(3px, -3px) scale(1.1); opacity: 0.8; }
  }

  @keyframes radar-scan {
      from { transform: rotate(0deg); }
      to { transform: rotate(360deg); }
  }
</style>
